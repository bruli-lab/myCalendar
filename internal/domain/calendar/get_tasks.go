package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type GetTasks struct {
	tasksRepo TasksRepository
	publisher Publisher
	authRepo  auth.AuthenticationRepository
	tracer    trace.Tracer
}

func (e GetTasks) Get(ctx context.Context, from, to time.Time, messageTitle string, eventType SlotType) error {
	ctx, span := e.tracer.Start(ctx, "GetTasks")
	defer span.End()
	tokenstr, err := e.authRepo.Read(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	accessToken := tokenstr.AccessToken()
	tokenType := tokenstr.TokenType()
	tasks, err := e.tasksRepo.GetTasks(ctx, from, to, accessToken, tokenType, eventType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if len(tasks) == 0 {
		messageTitle = "✅ No tasks found"
	}
	if err = e.publisher.Publish(ctx, messageTitle); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	for _, evnt := range tasks {
		msg := fmt.Sprintf("%s\nDue at: %s\nLink -> %s", evnt.Title(), evnt.Due(), evnt.Link())
		if err = e.publisher.Publish(ctx, msg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	span.SetStatus(codes.Ok, "Tasks published")
	return nil
}

func NewGetTasks(
	taskRepo TasksRepository,
	eventsPub Publisher,
	authRepo auth.AuthenticationRepository,
	tracer trace.Tracer,
) *GetTasks {
	return &GetTasks{tasksRepo: taskRepo, publisher: eventsPub, authRepo: authRepo, tracer: tracer}
}
