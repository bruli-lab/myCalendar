package fixtures

import (
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
)

var Madrid = mustLoadLocation("Europe/Madrid")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type PeriodBuilder struct {
	Kind *calendar.PeriodKind
	Day  *time.Time
}

func (b PeriodBuilder) Build() calendar.Period {
	p, err := calendar.NewPeriod(
		setData(calendar.DailyPeriod, b.Kind),
		setData(time.Date(2026, time.November, 13, 8, 0, 0, 0, Madrid), b.Day),
	)
	if err != nil {
		panic(err)
	}
	return p
}

type EventBuilder struct {
	Summary *string
	Start   *time.Time
	AllDay  bool
	Link    *string
}

func (b EventBuilder) Build() calendar.Event {
	return calendar.NewEvent(
		setData("Team meeting", b.Summary),
		setData(time.Date(2026, time.November, 13, 9, 30, 0, 0, Madrid), b.Start),
		b.AllDay,
		setData("https://calendar.google.com/event?eid=1", b.Link),
	)
}

type TaskBuilder struct {
	Title *string
	Due   *time.Time
	Link  *string
}

func (b TaskBuilder) Build() calendar.Task {
	return calendar.NewTask(
		setData("Buy milk", b.Title),
		setData(time.Date(2026, time.November, 13, 0, 0, 0, 0, time.UTC), b.Due),
		setData("https://tasks.google.com/task/1", b.Link),
	)
}
