package listener_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/bruli-lab/myCalendar/internal/infra/listener"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func summaryGenerated(kind calendar.PeriodKind, events []calendar.Event, tasks []calendar.Task) event.Event {
	day := time.Date(2026, time.November, 16, 8, 0, 0, 0, fixtures.Madrid)
	period := fixtures.PeriodBuilder{Kind: &kind, Day: &day}.Build()
	return calendar.NewSummary(period, events, tasks).Events()[0]
}

func TestPublishSummary_Listen(t *testing.T) {
	errTest := errors.New("test error")
	tuesdayMeeting := time.Date(2026, time.November, 17, 9, 30, 0, 0, fixtures.Madrid)
	holiday := time.Date(2026, time.November, 16, 0, 0, 0, 0, fixtures.Madrid)
	friday := time.Date(2026, time.November, 20, 0, 0, 0, 0, time.UTC)
	trickyTitle := "Fix *bold* [link]"
	noLink := ""
	events := []calendar.Event{
		fixtures.EventBuilder{Start: &holiday, AllDay: true, Summary: &trickyTitle, Link: &noLink}.Build(),
		fixtures.EventBuilder{Start: &tuesdayMeeting}.Build(),
	}
	tasks := []calendar.Task{fixtures.TaskBuilder{Due: &friday}.Build()}
	tests := []struct {
		name                        string
		ev                          event.Event
		publishErr                  error
		expectedErr                 error
		expectedTitle, expectedBody string
	}{
		{
			name:        "with an unexpected event, then it returns an invalid event error",
			ev:          event.NewBasicEvent("other", uuid.New(), "id"),
			expectedErr: listener.InvalidEventError{},
		},
		{
			name:        "with a summary and publisher returns an error, then it returns same error",
			ev:          summaryGenerated(calendar.DailyPeriod, nil, nil),
			publishErr:  errTest,
			expectedErr: errTest,
		},
		{
			name:          "with an empty daily summary, then it publishes a single message saying there is nothing",
			ev:            summaryGenerated(calendar.DailyPeriod, nil, nil),
			expectedTitle: "📅 Monday 16 November",
			expectedBody:  "**Events**\n_No events_\n\n**Tasks**\n_No tasks_",
		},
		{
			name:          "with a daily summary, then it publishes hours, all day events and escaped titles in a single message",
			ev:            summaryGenerated(calendar.DailyPeriod, events, tasks),
			expectedTitle: "📅 Monday 16 November",
			expectedBody: "**Events**\n" +
				"- All day · **Fix \\*bold\\* \\[link\\]**\n" +
				"- 09:30 · [**Team meeting**](https://calendar.google.com/event?eid=1)\n" +
				"\n**Tasks**\n" +
				"- [**Buy milk**](https://tasks.google.com/task/1)",
		},
		{
			name:          "with a weekly summary, then it publishes the weekday of every event and task",
			ev:            summaryGenerated(calendar.WeeklyPeriod, events, tasks),
			expectedTitle: "📅 Week of 16 November",
			expectedBody: "**Events**\n" +
				"- Monday All day · **Fix \\*bold\\* \\[link\\]**\n" +
				"- Tuesday 09:30 · [**Team meeting**](https://calendar.google.com/event?eid=1)\n" +
				"\n**Tasks**\n" +
				"- Friday · [**Buy milk**](https://tasks.google.com/task/1)",
		},
	}
	for _, tt := range tests {
		t.Run(`Given a PublishSummary listener,
		when Listen method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			publisher := &MessagePublisherMock{
				PublishFunc: func(_ context.Context, _, _ string) error {
					return tt.publishErr
				},
			}

			err := listener.NewPublishSummary(publisher, noop.NewTracerProvider().Tracer("test")).Listen(t.Context(), tt.ev)

			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, publisher.PublishCalls(), 1)
			require.Equal(t, tt.expectedTitle, publisher.PublishCalls()[0].Title)
			require.Equal(t, tt.expectedBody, publisher.PublishCalls()[0].Body)
		})
	}
}
