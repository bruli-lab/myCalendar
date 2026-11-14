package calendar_test

import (
	"testing"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestNewEvent(t *testing.T) {
	start := time.Date(2026, time.November, 13, 9, 30, 0, 0, fixtures.Madrid)
	tests := []struct {
		name, summary, expectedSummary string
	}{
		{
			name:            "with a summary, then it keeps the summary",
			summary:         "Team meeting",
			expectedSummary: "Team meeting",
		},
		{
			name:            "with an empty summary, then it uses an untitled summary",
			expectedSummary: "(No title)",
		},
	}
	for _, tt := range tests {
		t.Run(`Given an event constructor,
		when NewEvent function is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			ev := calendar.NewEvent(tt.summary, start, true, "https://link")

			require.Equal(t, tt.expectedSummary, ev.Summary())
			require.Equal(t, start, ev.Start())
			require.True(t, ev.AllDay())
			require.Equal(t, "https://link", ev.Link())
		})
	}
}

func TestNewTask(t *testing.T) {
	tests := []struct {
		name, title, expectedTitle string
		due, expectedDue           time.Time
	}{
		{
			name:          "with a title and a midnight UTC due, then it keeps both",
			title:         "Buy milk",
			due:           time.Date(2026, time.November, 13, 0, 0, 0, 0, time.UTC),
			expectedTitle: "Buy milk",
			expectedDue:   time.Date(2026, time.November, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "with an empty title and a due with time, then it uses an untitled title and keeps only the date",
			due:           time.Date(2026, time.November, 13, 17, 45, 0, 0, time.UTC),
			expectedTitle: "(No title)",
			expectedDue:   time.Date(2026, time.November, 13, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(`Given a task constructor,
		when NewTask function is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			task := calendar.NewTask(tt.title, tt.due, "https://link")

			require.Equal(t, tt.expectedTitle, task.Title())
			require.Equal(t, tt.expectedDue, task.Due())
			require.Equal(t, "https://link", task.Link())
		})
	}
}
