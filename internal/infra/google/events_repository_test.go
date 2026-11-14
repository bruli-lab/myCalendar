//go:build infra

package google_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/bruli-lab/myCalendar/internal/infra/google"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gcalendar "google.golang.org/api/calendar/v3"
)

func TestEventsRepository_FindByPeriod(t *testing.T) {
	t.Run(`Given a calendar API with two pages of events including an all day event and an invalid one,
	when FindByPeriod method is called,
	then it requests the period in the configured calendar, follows the pages and returns the valid events in the period location`, func(t *testing.T) {
		t.Parallel()
		var (
			mu      sync.Mutex
			queries []url.Values
		)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/calendars/work@example.com/events", r.URL.Path)
			mu.Lock()
			queries = append(queries, r.URL.Query())
			mu.Unlock()
			if r.URL.Query().Get("pageToken") == "" {
				writeJSON(t, w, gcalendar.Events{
					Items: []*gcalendar.Event{
						{Id: "1", Summary: "Holiday", HtmlLink: "https://link/1", Start: &gcalendar.EventDateTime{Date: "2026-11-13"}},
						{Id: "2", Summary: "Broken", Start: &gcalendar.EventDateTime{DateTime: "not a date"}},
					},
					NextPageToken: "page-2",
				})
				return
			}
			writeJSON(t, w, gcalendar.Events{
				Items: []*gcalendar.Event{
					{Id: "3", Summary: "Team meeting", HtmlLink: "https://link/3", Start: &gcalendar.EventDateTime{DateTime: "2026-11-13T08:30:00Z"}},
				},
			})
		}))
		defer srv.Close()
		repo := google.NewEventsRepository(calendarService(t, srv), "work@example.com", tracer())

		events, err := repo.FindByPeriod(t.Context(), fixtures.PeriodBuilder{}.Build())

		require.NoError(t, err)
		require.Len(t, queries, 2)
		require.Equal(t, "2026-11-13T00:00:00+01:00", queries[0].Get("timeMin"))
		require.Equal(t, "2026-11-14T00:00:00+01:00", queries[0].Get("timeMax"))
		require.Equal(t, "true", queries[0].Get("singleEvents"))
		require.Equal(t, "startTime", queries[0].Get("orderBy"))
		require.Equal(t, "page-2", queries[1].Get("pageToken"))
		require.Len(t, events, 2)
		require.Equal(t, "Holiday", events[0].Summary())
		require.True(t, events[0].AllDay())
		require.True(t, time.Date(2026, time.November, 13, 0, 0, 0, 0, fixtures.Madrid).Equal(events[0].Start()))
		require.Equal(t, "https://link/1", events[0].Link())
		require.Equal(t, "Team meeting", events[1].Summary())
		require.False(t, events[1].AllDay())
		require.Equal(t, "09:30", events[1].Start().Format("15:04"))
	})

	t.Run(`Given a calendar API that fails,
	when FindByPeriod method is called,
	then it returns an error`, func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		repo := google.NewEventsRepository(calendarService(t, srv), "primary", tracer())

		_, err := repo.FindByPeriod(t.Context(), fixtures.PeriodBuilder{}.Build())

		require.Error(t, err)
	})
}
