package app

import (
	"context"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const SendSummaryCmdName = "send_summary"

type SendSummaryCmd struct {
	Kind calendar.PeriodKind
	Day  time.Time
}

func (SendSummaryCmd) Name() string {
	return SendSummaryCmdName
}

type SendSummary struct {
	eventsRepo EventsRepository
	tasksRepo  TasksRepository
	tracer     trace.Tracer
}

func (s SendSummary) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := s.tracer.Start(ctx, "SendSummary")
	defer span.End()
	co, ok := cmd.(SendSummaryCmd)
	if !ok {
		return nil, recordError(span, cqs.NewInvalidCommandError(SendSummaryCmdName, cmd.Name()))
	}
	period, err := calendar.NewPeriod(co.Kind, co.Day)
	if err != nil {
		return nil, recordError(span, err)
	}
	span.SetAttributes(
		attribute.String("period.kind", string(period.Kind())),
		attribute.String("period.from", period.From().Format(time.RFC3339)),
		attribute.String("period.to", period.To().Format(time.RFC3339)),
	)
	events, err := s.eventsRepo.FindByPeriod(ctx, period)
	if err != nil {
		return nil, recordError(span, err)
	}
	tasks, err := s.tasksRepo.FindByPeriod(ctx, period)
	if err != nil {
		return nil, recordError(span, err)
	}
	summary := calendar.NewSummary(period, events, tasks)
	span.SetStatus(codes.Ok, "summary generated")
	return summary.Events(), nil
}

func NewSendSummary(eventsRepo EventsRepository, tasksRepo TasksRepository, tracer trace.Tracer) *SendSummary {
	return &SendSummary{eventsRepo: eventsRepo, tasksRepo: tasksRepo, tracer: tracer}
}
