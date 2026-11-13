package calendar_test

import (
	"testing"

	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestNewSummary(t *testing.T) {
	t.Run(`Given a period with events and tasks,
	when NewSummary function is called,
	then it records a summary generated event with the same data and clears it once read`, func(t *testing.T) {
		t.Parallel()
		period := fixtures.PeriodBuilder{}.Build()
		events := []calendar.Event{fixtures.EventBuilder{}.Build()}
		tasks := []calendar.Task{fixtures.TaskBuilder{}.Build()}

		summary := calendar.NewSummary(period, events, tasks)

		require.Equal(t, "daily-2026-11-13", summary.ID())
		require.Equal(t, period, summary.Period())
		require.Equal(t, events, summary.CalendarEvents())
		require.Equal(t, tasks, summary.Tasks())
		recorded := summary.Events()
		require.Len(t, recorded, 1)
		generated, ok := recorded[0].(calendar.SummaryGenerated)
		require.True(t, ok)
		require.Equal(t, calendar.SummaryGeneratedEventName, generated.EventName())
		require.Equal(t, summary.ID(), generated.AggregateRootID())
		require.Equal(t, period, generated.Period)
		require.Equal(t, events, generated.Events)
		require.Equal(t, tasks, generated.Tasks)
		require.Empty(t, summary.Events())
	})
}
