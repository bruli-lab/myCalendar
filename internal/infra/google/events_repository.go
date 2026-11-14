package google

import (
	"context"
	"fmt"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	gcalendar "google.golang.org/api/calendar/v3"
)

const allDayLayout = time.DateOnly

type EventsRepository struct {
	svc        *gcalendar.Service
	calendarID string
	tracer     trace.Tracer
}

func (e EventsRepository) FindByPeriod(ctx context.Context, period calendar.Period) ([]calendar.Event, error) {
	ctx, span := e.tracer.Start(ctx, "EventsRepository.FindByPeriod")
	defer span.End()
	var events []calendar.Event
	err := e.svc.Events.List(e.calendarID).
		TimeMin(period.From().Format(time.RFC3339)).
		TimeMax(period.To().Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Pages(ctx, func(page *gcalendar.Events) error {
			for _, item := range page.Items {
				ev, err := toEvent(item, period.Location())
				if err != nil {
					span.AddEvent("event skipped", trace.WithAttributes(
						attribute.String("event.id", item.Id),
						attribute.String("error", err.Error()),
					))
					continue
				}
				events = append(events, ev)
			}
			return nil
		})
	if err != nil {
		return nil, recordError(span, fmt.Errorf("failed to get events: %w", err))
	}
	span.SetStatus(codes.Ok, "events found")
	return events, nil
}

func toEvent(item *gcalendar.Event, loc *time.Location) (calendar.Event, error) {
	if item.Start == nil {
		return calendar.Event{}, fmt.Errorf("event %q has no start", item.Id)
	}
	if item.Start.DateTime != "" {
		start, err := time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			return calendar.Event{}, fmt.Errorf("invalid start date time: %w", err)
		}
		return calendar.NewEvent(item.Summary, start.In(loc), false, item.HtmlLink), nil
	}
	start, err := time.ParseInLocation(allDayLayout, item.Start.Date, loc)
	if err != nil {
		return calendar.Event{}, fmt.Errorf("invalid start date: %w", err)
	}
	return calendar.NewEvent(item.Summary, start, true, item.HtmlLink), nil
}

func NewEventsRepository(svc *gcalendar.Service, calendarID string, tracer trace.Tracer) *EventsRepository {
	return &EventsRepository{svc: svc, calendarID: calendarID, tracer: tracer}
}
