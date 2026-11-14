package calendar_test

import (
	"testing"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestNewPeriod(t *testing.T) {
	friday := time.Date(2026, time.November, 13, 8, 15, 0, 0, fixtures.Madrid)
	dstChangeDay := time.Date(2026, time.March, 29, 8, 0, 0, 0, fixtures.Madrid)
	tests := []struct {
		name                       string
		kind                       calendar.PeriodKind
		day                        time.Time
		expectedFrom, expectedTo   time.Time
		expectedFirst, expectedEnd time.Time
		expectedErr                error
	}{
		{
			name:        "with an unknown kind, then it returns an invalid period kind error",
			kind:        "monthly",
			day:         friday,
			expectedErr: calendar.InvalidPeriodKindError{},
		},
		{
			name:          "with a daily kind, then it covers the whole local day",
			kind:          calendar.DailyPeriod,
			day:           friday,
			expectedFrom:  time.Date(2026, time.November, 13, 0, 0, 0, 0, fixtures.Madrid),
			expectedTo:    time.Date(2026, time.November, 14, 0, 0, 0, 0, fixtures.Madrid),
			expectedFirst: time.Date(2026, time.November, 13, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2026, time.November, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "with a weekly kind, then it covers seven local days",
			kind:          calendar.WeeklyPeriod,
			day:           friday,
			expectedFrom:  time.Date(2026, time.November, 13, 0, 0, 0, 0, fixtures.Madrid),
			expectedTo:    time.Date(2026, time.November, 20, 0, 0, 0, 0, fixtures.Madrid),
			expectedFirst: time.Date(2026, time.November, 13, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2026, time.November, 19, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "with a daily kind on a DST change day, then it ends at the next local midnight",
			kind:          calendar.DailyPeriod,
			day:           dstChangeDay,
			expectedFrom:  time.Date(2026, time.March, 29, 0, 0, 0, 0, fixtures.Madrid),
			expectedTo:    time.Date(2026, time.March, 30, 0, 0, 0, 0, fixtures.Madrid),
			expectedFirst: time.Date(2026, time.March, 29, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2026, time.March, 29, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(`Given a period constructor,
		when NewPeriod function is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			period, err := calendar.NewPeriod(tt.kind, tt.day)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.kind, period.Kind())
			require.True(t, tt.expectedFrom.Equal(period.From()), "from %s", period.From())
			require.True(t, tt.expectedTo.Equal(period.To()), "to %s", period.To())
			require.Equal(t, tt.expectedFirst, period.FirstDay())
			require.Equal(t, tt.expectedEnd, period.LastDay())
			require.Equal(t, fixtures.Madrid, period.Location())
		})
	}
}
