//go:build infra

package google_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/bruli-lab/myCalendar/internal/infra/google"
	"github.com/stretchr/testify/require"
	gtasks "google.golang.org/api/tasks/v1"
)

func TestTasksRepository_FindByPeriod(t *testing.T) {
	t.Run(`Given a tasks API with paginated task lists and tasks including one with an invalid due,
	when FindByPeriod method is called,
	then it filters by the period days in UTC, follows every page and returns the valid tasks with their web link`, func(t *testing.T) {
		t.Parallel()
		var (
			mu          sync.Mutex
			taskQueries []url.Values
		)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/tasks/v1/users/@me/lists":
				if r.URL.Query().Get("pageToken") == "" {
					writeJSON(t, w, gtasks.TaskLists{Items: []*gtasks.TaskList{{Id: "home"}}, NextPageToken: "lists-2"})
					return
				}
				writeJSON(t, w, gtasks.TaskLists{Items: []*gtasks.TaskList{{Id: "work"}}})
			case "/tasks/v1/lists/home/tasks":
				mu.Lock()
				taskQueries = append(taskQueries, r.URL.Query())
				mu.Unlock()
				if r.URL.Query().Get("pageToken") == "" {
					writeJSON(t, w, gtasks.Tasks{
						Items:         []*gtasks.Task{{Id: "1", Title: "Buy milk", Due: "2026-11-16T00:00:00.000Z", WebViewLink: "https://tasks/1"}},
						NextPageToken: "tasks-2",
					})
					return
				}
				writeJSON(t, w, gtasks.Tasks{Items: []*gtasks.Task{{Id: "2", Title: "Broken", Due: "tomorrow"}}})
			case "/tasks/v1/lists/work/tasks":
				writeJSON(t, w, gtasks.Tasks{Items: []*gtasks.Task{{Id: "3", Title: "Send report", Due: "2026-11-22T00:00:00.000Z", WebViewLink: "https://tasks/3"}}})
			default:
				t.Errorf("unexpected path %s", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()
		weekly := calendar.WeeklyPeriod
		day := time.Date(2026, time.November, 16, 8, 0, 0, 0, fixtures.Madrid)
		repo := google.NewTasksRepository(tasksService(t, srv), tracer())

		tasks, err := repo.FindByPeriod(t.Context(), fixtures.PeriodBuilder{Kind: &weekly, Day: &day}.Build())

		require.NoError(t, err)
		require.Len(t, taskQueries, 2)
		require.Equal(t, "2026-11-16T00:00:00Z", taskQueries[0].Get("dueMin"))
		require.Equal(t, "2026-11-22T23:59:59Z", taskQueries[0].Get("dueMax"))
		require.Equal(t, "false", taskQueries[0].Get("showCompleted"))
		require.Equal(t, "tasks-2", taskQueries[1].Get("pageToken"))
		require.Len(t, tasks, 2)
		require.Equal(t, "Buy milk", tasks[0].Title())
		require.Equal(t, time.Date(2026, time.November, 16, 0, 0, 0, 0, time.UTC), tasks[0].Due())
		require.Equal(t, "https://tasks/1", tasks[0].Link())
		require.Equal(t, "Send report", tasks[1].Title())
	})

	t.Run(`Given a tasks API that fails listing the task lists,
	when FindByPeriod method is called,
	then it returns an error`, func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		repo := google.NewTasksRepository(tasksService(t, srv), tracer())

		_, err := repo.FindByPeriod(t.Context(), fixtures.PeriodBuilder{}.Build())

		require.Error(t, err)
	})
}
