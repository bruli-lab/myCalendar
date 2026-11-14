package calendar

import (
	"fmt"

	"github.com/bruli-lab/go-core/event"
	"github.com/google/uuid"
)

const SummaryGeneratedEventName = "summary_generated"

type SummaryGenerated struct {
	event.BasicEvent
	Period Period
	Events []Event
	Tasks  []Task
}

type Summary struct {
	event.BasicAggregateRoot
	period Period
	events []Event
	tasks  []Task
}

func (s *Summary) ID() string {
	return fmt.Sprintf("%s-%s", s.period.Kind(), s.period.From().Format("2006-01-02"))
}

func (s *Summary) Period() Period {
	return s.period
}

func (s *Summary) CalendarEvents() []Event {
	return s.events
}

func (s *Summary) Tasks() []Task {
	return s.tasks
}

func NewSummary(period Period, events []Event, tasks []Task) *Summary {
	s := &Summary{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		period:             period,
		events:             events,
		tasks:              tasks,
	}
	s.Record(SummaryGenerated{
		BasicEvent: event.NewBasicEvent(SummaryGeneratedEventName, uuid.New(), s.ID()),
		Period:     period,
		Events:     events,
		Tasks:      tasks,
	})
	return s
}
