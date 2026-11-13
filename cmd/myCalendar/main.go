package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/myCalendar/internal/app"
	"github.com/bruli-lab/myCalendar/internal/config"
	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"github.com/bruli-lab/myCalendar/internal/infra/disk"
	googleinfra "github.com/bruli-lab/myCalendar/internal/infra/google"
	httpinfra "github.com/bruli-lab/myCalendar/internal/infra/http"
	"github.com/bruli-lab/myCalendar/internal/infra/listener"
	"github.com/bruli-lab/myCalendar/internal/infra/ntfy"
	"github.com/bruli-lab/myCalendar/internal/infra/tracing"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"golang.org/x/oauth2"
)

const (
	serviceName     = "myCalendar"
	shutdownTimeout = 5 * time.Second
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conf, err := config.New()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error reading config: %s\n", err)
		return err
	}
	log := buildLog(conf.LogLevel)

	tracingProv, err := tracing.InitTracing(ctx, serviceName)
	if err != nil {
		log.ErrorContext(ctx, "Error initializing tracing", "err", err)
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()
		if err := tracingProv.Shutdown(shutdownCtx); err != nil {
			log.ErrorContext(ctx, "Error shutting down tracing", "err", err)
		}
	}()
	tracer := otel.Tracer(serviceName)

	serverErr, stopServer, err := startHealthServer(ctx, log, conf.ServerHost)
	if err != nil {
		return err
	}
	defer stopServer()

	publisher, err := ntfy.NewPublisher(conf.NtfyUser, conf.NtfyPassword, conf.NtfyURL, conf.NtfyTopic, tracer)
	if err != nil {
		log.ErrorContext(ctx, "Error initializing ntfy publisher", "err", err)
		return err
	}

	oauthCfg := googleinfra.NewOAuthConfig(conf.ClientID, conf.ClientSecret, conf.CallbackURL)
	authRepo := disk.NewAuthenticationRepository(conf.TokensFile)
	tokenRepo := googleinfra.NewTokenRepository(oauthCfg)
	tokenSource := oauth2.ReuseTokenSource(nil, googleinfra.NewPersistentTokenSource(ctx, oauthCfg, authRepo))

	calendarSvc, err := googleinfra.NewCalendarService(ctx, tokenSource)
	if err != nil {
		log.ErrorContext(ctx, "Error creating calendar service", "err", err)
		return err
	}
	tasksSvc, err := googleinfra.NewTasksService(ctx, tokenSource)
	if err != nil {
		log.ErrorContext(ctx, "Error creating tasks service", "err", err)
		return err
	}

	eventBus := event.NewBus()
	eventBus.Subscribe(calendar.SummaryGeneratedEventName, listener.NewPublishSummary(publisher, tracer))
	withEvents := cqs.NewCommandHandlerEventBusMiddleware(&eventBus, tracer)

	commandBus := cqs.NewCommandBus()
	commandBus.Subscribe(app.RefreshTokenCmdName, withEvents(app.NewRefreshToken(authRepo, tokenRepo, tracer)))
	commandBus.Subscribe(app.CreateTokensCmdName, withEvents(app.NewCreateTokens(tokenRepo, authRepo, tracer)))
	commandBus.Subscribe(app.SendSummaryCmdName, withEvents(app.NewSendSummary(
		googleinfra.NewEventsRepository(calendarSvc, conf.CalendarID, tracer),
		googleinfra.NewTasksRepository(tasksSvc, tracer),
		tracer,
	)))

	authorizer := httpinfra.NewAuthorizer(oauthCfg, publisher, commandBus, conf.CallbackHost)
	if err = ensureAuthenticated(ctx, log, commandBus, authorizer); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}

	scheduler, err := scheduleSummaries(ctx, log, commandBus, authorizer, conf)
	if err != nil {
		log.ErrorContext(ctx, "Error scheduling summaries", "err", err)
		return err
	}
	scheduler.Start()
	defer func() {
		<-scheduler.Stop().Done()
	}()

	select {
	case err := <-serverErr:
		log.ErrorContext(ctx, "Error running server", "err", err)
		return err
	case <-ctx.Done():
		log.InfoContext(ctx, "Shutdown signal received")
		return nil
	}
}

func startHealthServer(ctx context.Context, log *slog.Logger, host string) (serverErr <-chan error, stop func(), err error) {
	ln, err := new(net.ListenConfig).Listen(ctx, "tcp", host)
	if err != nil {
		log.ErrorContext(ctx, "Error starting server", "err", err)
		return nil, nil, err
	}
	log.InfoContext(ctx, "Starting server", "host", host)
	srv := httpinfra.NewServer(host)
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	return errCh, func() { shutdownServer(ctx, log, srv) }, nil
}

func shutdownServer(ctx context.Context, log *slog.Logger, srv *http.Server) {
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.ErrorContext(ctx, "Error shutting down server", "addr", srv.Addr, "err", err)
	}
}

func scheduleSummaries(
	ctx context.Context,
	log *slog.Logger,
	bus cqs.CommandBus,
	authorizer *httpinfra.Authorizer,
	conf *config.Config,
) (*cron.Cron, error) {
	c := cron.New(cron.WithLocation(conf.Location))
	schedules := map[calendar.PeriodKind]string{
		calendar.DailyPeriod:  conf.DailySchedule,
		calendar.WeeklyPeriod: conf.WeeklySchedule,
	}
	for kind, schedule := range schedules {
		if _, err := c.AddFunc(schedule, func() {
			sendSummary(ctx, log, bus, authorizer, kind, conf)
		}); err != nil {
			return nil, fmt.Errorf("invalid %s schedule %q: %w", kind, schedule, err)
		}
		log.InfoContext(ctx, "Summary scheduled", "kind", kind, "schedule", schedule)
	}
	return c, nil
}

func sendSummary(
	ctx context.Context,
	log *slog.Logger,
	bus cqs.CommandBus,
	authorizer *httpinfra.Authorizer,
	kind calendar.PeriodKind,
	conf *config.Config,
) {
	jobCtx, cancel := context.WithTimeout(ctx, conf.JobTimeout)
	defer cancel()
	log.InfoContext(jobCtx, "Sending summary", "kind", kind)
	_, err := bus.Handle(jobCtx, app.SendSummaryCmd{Kind: kind, Day: time.Now().In(conf.Location)})
	if err == nil {
		return
	}
	log.ErrorContext(jobCtx, "Error sending summary", "kind", kind, "err", err)
	if errors.As(err, &auth.RefreshError{}) {
		go func() {
			if err := authorize(ctx, log, authorizer); err != nil && !errors.Is(err, context.Canceled) {
				log.ErrorContext(ctx, "Error re-authorizing", "err", err)
			}
		}()
	}
}

func ensureAuthenticated(ctx context.Context, log *slog.Logger, bus cqs.CommandBus, authorizer *httpinfra.Authorizer) error {
	_, err := bus.Handle(ctx, app.RefreshTokenCmd{})
	if err == nil {
		return nil
	}
	if !errors.As(err, &auth.RefreshError{}) {
		log.ErrorContext(ctx, "Error refreshing token", "err", err)
		return err
	}
	if err = authorize(ctx, log, authorizer); err != nil && !errors.Is(err, context.Canceled) {
		log.ErrorContext(ctx, "Error authorizing", "err", err)
	}
	return err
}

func authorize(ctx context.Context, log *slog.Logger, authorizer *httpinfra.Authorizer) error {
	log.WarnContext(ctx, "Authorization required, waiting for the user to authorize")
	if err := authorizer.Authorize(ctx); err != nil {
		if errors.Is(err, httpinfra.ErrAuthorizationInProgress) {
			log.InfoContext(ctx, "Authorization already in progress")
			return nil
		}
		return err
	}
	log.InfoContext(ctx, "Authorization completed, tokens created")
	return nil
}

func buildLog(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With("service", serviceName)
}
