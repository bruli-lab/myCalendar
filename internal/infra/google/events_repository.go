package google

import (
	"context"
	"fmt"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	gcalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type EventsRepository struct {
	tracer trace.Tracer
}

func (e EventsRepository) GetEvents(ctx context.Context, from, to time.Time, accessToken, tokenType string, eventType calendar.SlotType) ([]calendar.Event, error) {
	ctx, span := e.tracer.Start(ctx, "EventsRepository.GetEvents")
	defer span.End()
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   tokenType,
	})
	client := oauth2.NewClient(ctx, ts)
	svc, err := gcalendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		span.RecordError(err)
		err = fmt.Errorf("failed to create calendar client: %s", err.Error())
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	resp, err := svc.Events.List("primary").
		TimeMin(from.Format(time.RFC3339)).
		TimeMax(to.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		err = fmt.Errorf("failed to get events: %s", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	events := make([]calendar.Event, len(resp.Items))
	for i, ev := range resp.Items {
		var (
			description *string
			location    *string
		)
		if ev.Description != "" {
			description = &ev.Description
		}
		if ev.Location != "" {
			location = &ev.Location
		}
		start, _ := time.Parse(time.RFC3339, ev.Start.DateTime)
		end, _ := time.Parse(time.RFC3339, ev.End.DateTime)
		events[i] = *calendar.NewEvent(ev.Summary, description, location, start, end, ev.HtmlLink, eventType)
	}
	return events, nil
}

func NewEventsRepository(tracer trace.Tracer) *EventsRepository {
	return &EventsRepository{tracer: tracer}
}
