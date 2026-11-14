package google

import (
	"context"
	"fmt"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	gtasks "google.golang.org/api/tasks/v1"
)

const (
	maxTasksPerPage = 100
	endOfDay        = 24*time.Hour - time.Second
)

type TasksRepository struct {
	svc    *gtasks.Service
	tracer trace.Tracer
}

func (t TasksRepository) FindByPeriod(ctx context.Context, period calendar.Period) ([]calendar.Task, error) {
	ctx, span := t.tracer.Start(ctx, "TasksRepository.FindByPeriod")
	defer span.End()
	listIDs, err := t.taskListIDs(ctx)
	if err != nil {
		return nil, recordError(span, err)
	}
	var tasks []calendar.Task
	for _, listID := range listIDs {
		err = t.svc.Tasks.List(listID).
			DueMin(period.FirstDay().Format(time.RFC3339)).
			DueMax(period.LastDay().Add(endOfDay).Format(time.RFC3339)).
			ShowCompleted(false).
			ShowDeleted(false).
			ShowHidden(false).
			MaxResults(maxTasksPerPage).
			Pages(ctx, func(page *gtasks.Tasks) error {
				for _, item := range page.Items {
					due, err := time.Parse(time.RFC3339, item.Due)
					if err != nil {
						span.AddEvent("task skipped", trace.WithAttributes(
							attribute.String("task.id", item.Id),
							attribute.String("error", err.Error()),
						))
						continue
					}
					tasks = append(tasks, calendar.NewTask(item.Title, due, item.WebViewLink))
				}
				return nil
			})
		if err != nil {
			return nil, recordError(span, fmt.Errorf("failed to get tasks from list %s: %w", listID, err))
		}
	}
	span.SetStatus(codes.Ok, "tasks found")
	return tasks, nil
}

func (t TasksRepository) taskListIDs(ctx context.Context) ([]string, error) {
	var ids []string
	err := t.svc.Tasklists.List().Pages(ctx, func(page *gtasks.TaskLists) error {
		for _, list := range page.Items {
			ids = append(ids, list.Id)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get task lists: %w", err)
	}
	return ids, nil
}

func NewTasksRepository(svc *gtasks.Service, tracer trace.Tracer) *TasksRepository {
	return &TasksRepository{svc: svc, tracer: tracer}
}
