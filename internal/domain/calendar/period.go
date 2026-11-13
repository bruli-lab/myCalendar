package calendar

import (
	"fmt"
	"time"
)

const (
	DailyPeriod  PeriodKind = "daily"
	WeeklyPeriod PeriodKind = "weekly"

	daysInWeek = 7
)

type PeriodKind string

type InvalidPeriodKindError struct {
	kind PeriodKind
}

func (e InvalidPeriodKindError) Error() string {
	return fmt.Sprintf("invalid period kind %q", e.kind)
}

type Period struct {
	kind PeriodKind
	from time.Time
	to   time.Time
}

func (p Period) Kind() PeriodKind {
	return p.kind
}

func (p Period) From() time.Time {
	return p.from
}

func (p Period) To() time.Time {
	return p.to
}

func (p Period) Location() *time.Location {
	return p.from.Location()
}

func (p Period) FirstDay() time.Time {
	return dateInUTC(p.from)
}

func (p Period) LastDay() time.Time {
	return dateInUTC(p.to.AddDate(0, 0, -1))
}

func NewPeriod(kind PeriodKind, day time.Time) (Period, error) {
	from := startOfDay(day)
	switch kind {
	case DailyPeriod:
		return Period{kind: kind, from: from, to: from.AddDate(0, 0, 1)}, nil
	case WeeklyPeriod:
		return Period{kind: kind, from: from, to: from.AddDate(0, 0, daysInWeek)}, nil
	default:
		return Period{}, InvalidPeriodKindError{kind: kind}
	}
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func dateInUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
