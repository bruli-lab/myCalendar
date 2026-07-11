package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type GetEvents struct {
	eventsRepo EventsRepository
	eventsPub  Publisher
	authRepo   auth.AuthenticationRepository
	tracer     trace.Tracer
}

func (e GetEvents) Get(ctx context.Context, from, to time.Time, messageTitle string, eventType SlotType) error {
	ctx, span := e.tracer.Start(ctx, "GetEvents")
	defer span.End()
	tokenstr, err := e.authRepo.Read(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	accessToken := tokenstr.AccessToken()
	tokenType := tokenstr.TokenType()
	evnts, err := e.eventsRepo.GetEvents(ctx, from, to, accessToken, tokenType, eventType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if len(evnts) == 0 {
		messageTitle = "📅 No events found"
	}
	if err := e.eventsPub.Publish(ctx, messageTitle); err != nil {
		return err
	}
	for _, evnt := range evnts {
		msg := fmt.Sprintf("%s\nStart at: %s\nLink -> %s", evnt.Summary(), evnt.Start(), evnt.Link())
		if err = e.eventsPub.Publish(ctx, msg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	span.SetStatus(codes.Ok, "Events published")
	return nil
}

func NewGetEvents(
	eventsRepo EventsRepository,
	eventsPub Publisher,
	authRepo auth.AuthenticationRepository,
	tracer trace.Tracer,
) *GetEvents {
	return &GetEvents{eventsRepo: eventsRepo, eventsPub: eventsPub, authRepo: authRepo, tracer: tracer}
}
