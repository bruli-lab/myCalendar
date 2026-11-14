package http

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/myCalendar/internal/app"
	"golang.org/x/oauth2"
)

//go:generate go tool moq -out authorizer_mock_test.go -pkg http_test . MessagePublisher

const (
	authorizationTitle = "🔑 MyCalendar authorization"
	shutdownTimeout    = 5 * time.Second
)

var ErrAuthorizationInProgress = errors.New("authorization already in progress")

type MessagePublisher interface {
	Publish(ctx context.Context, title, body string) error
}

type Authorizer struct {
	oauthCfg  *oauth2.Config
	publisher MessagePublisher
	commands  cqs.CommandHandler
	host      string
	running   sync.Mutex
}

func (a *Authorizer) Authorize(ctx context.Context) error {
	if !a.running.TryLock() {
		return ErrAuthorizationInProgress
	}
	defer a.running.Unlock()

	state := rand.Text()
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	ln, err := new(net.ListenConfig).Listen(ctx, "tcp", a.host)
	if err != nil {
		return fmt.Errorf("failed to listen on callback host %s: %w", a.host, err)
	}
	srv := NewCallbackServer(a.host, state, codeCh, errCh)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			notify(errCh, err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	authURL := a.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	if err = a.publisher.Publish(ctx, authorizationTitle, "Click -> "+authURL); err != nil {
		return fmt.Errorf("failed to publish authorization url: %w", err)
	}

	select {
	case code := <-codeCh:
		if _, err = a.commands.Handle(ctx, app.CreateTokensCmd{Code: code}); err != nil {
			return fmt.Errorf("failed to create tokens: %w", err)
		}
		return nil
	case err = <-errCh:
		return fmt.Errorf("authorization failed: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func NewAuthorizer(oauthCfg *oauth2.Config, publisher MessagePublisher, commands cqs.CommandHandler, host string) *Authorizer {
	return &Authorizer{oauthCfg: oauthCfg, publisher: publisher, commands: commands, host: host}
}
