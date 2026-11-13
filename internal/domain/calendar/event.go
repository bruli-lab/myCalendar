package calendar

import "time"

const untitled = "(No title)"

type Event struct {
	summary string
	start   time.Time
	allDay  bool
	link    string
}

func (e Event) Summary() string {
	return e.summary
}

func (e Event) Start() time.Time {
	return e.start
}

func (e Event) AllDay() bool {
	return e.allDay
}

func (e Event) Link() string {
	return e.link
}

func NewEvent(summary string, start time.Time, allDay bool, link string) Event {
	if summary == "" {
		summary = untitled
	}
	return Event{summary: summary, start: start, allDay: allDay, link: link}
}
