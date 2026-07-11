package google

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	gtasks "google.golang.org/api/tasks/v1"
)

type TasksRepository struct {
	log    *slog.Logger
	tracer trace.Tracer
}

func (t TasksRepository) GetTasks(ctx context.Context, from, to time.Time, accessToken, tokenType string, slotType calendar.SlotType) ([]calendar.Task, error) {
	ctx, span := t.tracer.Start(ctx, "TasksRepository.GetTasks")
	defer span.End()
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   tokenType,
	})
	client := oauth2.NewClient(ctx, ts)
	svc, err := gtasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		err = fmt.Errorf("failed to create tasks client: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	taskListsResp, err := svc.Tasklists.List().Do()
	if err != nil {
		err = fmt.Errorf("failed to get task lists: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	tasks := make([]calendar.Task, 0, len(taskListsResp.Items))
	for _, list := range taskListsResp.Items {
		call := svc.Tasks.List(list.Id).
			DueMin(from.Format(time.RFC3339)).
			DueMax(to.Format(time.RFC3339)).
			ShowCompleted(false).
			ShowDeleted(false).
			ShowHidden(false)

		resp, err := call.Do()
		if err != nil {
			err = fmt.Errorf("failed to get tasks from list %s: %w", list.Id, err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		for _, item := range resp.Items {
			var due *time.Time
			if item.Due != "" {
				dueParsed, err := time.Parse(time.RFC3339, item.Due)
				if err == nil {
					due = &dueParsed
				}
			}
			tasks = append(tasks, *calendar.NewTask(item.Title, due, item.SelfLink, slotType))
		}
	}
	return tasks, nil
}

func NewTasksRepository(log *slog.Logger, tracer trace.Tracer) *TasksRepository {
	return &TasksRepository{log: log, tracer: tracer}
}
