package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/myCalendar/internal/app"
	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestSendSummary_Handle(t *testing.T) {
	errTest := errors.New("test error")
	day := time.Date(2026, time.November, 16, 8, 0, 0, 0, fixtures.Madrid)
	weekly := calendar.WeeklyPeriod
	expectedPeriod := fixtures.PeriodBuilder{Kind: &weekly, Day: &day}.Build()
	events := []calendar.Event{fixtures.EventBuilder{}.Build()}
	tasks := []calendar.Task{fixtures.TaskBuilder{}.Build()}
	tests := []struct {
		name                string
		cmd                 cqs.Command
		eventsErr, tasksErr error
		expectedErr         error
		expectedEventsCalls int
		expectedTasksCalls  int
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "with an invalid period kind, then it returns an invalid period kind error",
			cmd:         app.SendSummaryCmd{Kind: "monthly", Day: day},
			expectedErr: calendar.InvalidPeriodKindError{},
		},
		{
			name:                "with a valid command and events repository returns an error, then it returns same error",
			cmd:                 app.SendSummaryCmd{Kind: calendar.WeeklyPeriod, Day: day},
			eventsErr:           errTest,
			expectedErr:         errTest,
			expectedEventsCalls: 1,
		},
		{
			name:                "with a valid command and tasks repository returns an error, then it returns same error",
			cmd:                 app.SendSummaryCmd{Kind: calendar.WeeklyPeriod, Day: day},
			tasksErr:            errTest,
			expectedErr:         errTest,
			expectedEventsCalls: 1,
			expectedTasksCalls:  1,
		},
		{
			name:                "with a valid command and both repositories succeed, then it returns a summary generated event",
			cmd:                 app.SendSummaryCmd{Kind: calendar.WeeklyPeriod, Day: day},
			expectedEventsCalls: 1,
			expectedTasksCalls:  1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a SendSummary command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			eventsRepo := &EventsRepositoryMock{
				FindByPeriodFunc: func(_ context.Context, _ calendar.Period) ([]calendar.Event, error) {
					return events, tt.eventsErr
				},
			}
			tasksRepo := &TasksRepositoryMock{
				FindByPeriodFunc: func(_ context.Context, _ calendar.Period) ([]calendar.Task, error) {
					return tasks, tt.tasksErr
				},
			}

			recorded, err := app.NewSendSummary(eventsRepo, tasksRepo, tracer()).Handle(t.Context(), tt.cmd)

			require.Len(t, eventsRepo.FindByPeriodCalls(), tt.expectedEventsCalls)
			require.Len(t, tasksRepo.FindByPeriodCalls(), tt.expectedTasksCalls)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				require.Empty(t, recorded)
				return
			}
			require.NoError(t, err)
			require.Equal(t, expectedPeriod, eventsRepo.FindByPeriodCalls()[0].Period)
			require.Equal(t, expectedPeriod, tasksRepo.FindByPeriodCalls()[0].Period)
			require.Len(t, recorded, 1)
			generated, ok := recorded[0].(calendar.SummaryGenerated)
			require.True(t, ok)
			require.Equal(t, expectedPeriod, generated.Period)
			require.Equal(t, events, generated.Events)
			require.Equal(t, tasks, generated.Tasks)
		})
	}
}
