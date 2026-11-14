package listener

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	hourLayout = "15:04"
	dayLayout  = "Monday 2 January"
)

type InvalidEventError struct {
	expected, had string
}

func (e InvalidEventError) Error() string {
	return fmt.Sprintf("invalid event, expected %q but found %q", e.expected, e.had)
}

type PublishSummary struct {
	publisher MessagePublisher
	tracer    trace.Tracer
}

func (p PublishSummary) Listen(ctx context.Context, ev event.Event) error {
	ctx, span := p.tracer.Start(ctx, "PublishSummary")
	defer span.End()
	summary, ok := ev.(calendar.SummaryGenerated)
	if !ok {
		err := InvalidEventError{expected: calendar.SummaryGeneratedEventName, had: ev.EventName()}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := p.publisher.Publish(ctx, summaryTitle(summary.Period), summaryBody(&summary)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "summary published")
	return nil
}

func NewPublishSummary(publisher MessagePublisher, tracer trace.Tracer) *PublishSummary {
	return &PublishSummary{publisher: publisher, tracer: tracer}
}

func summaryTitle(period calendar.Period) string {
	if period.Kind() == calendar.WeeklyPeriod {
		return "📅 Week of " + period.From().Format("2 January")
	}
	return "📅 " + period.From().Format(dayLayout)
}

func summaryBody(summary *calendar.SummaryGenerated) string {
	weekly := summary.Period.Kind() == calendar.WeeklyPeriod
	var b strings.Builder
	b.WriteString("**Events**\n")
	if len(summary.Events) == 0 {
		b.WriteString("_No events_\n")
	}
	for _, ev := range summary.Events {
		b.WriteString("- " + eventWhen(ev, weekly) + " · " + linked(ev.Summary(), ev.Link()) + "\n")
	}
	b.WriteString("\n**Tasks**\n")
	if len(summary.Tasks) == 0 {
		b.WriteString("_No tasks_\n")
	}
	for _, task := range summary.Tasks {
		b.WriteString("- " + taskWhen(task, weekly) + linked(task.Title(), task.Link()) + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func eventWhen(ev calendar.Event, weekly bool) string {
	when := ev.Start().Format(hourLayout)
	if ev.AllDay() {
		when = "All day"
	}
	if weekly {
		return weekday(ev.Start()) + " " + when
	}
	return when
}

func taskWhen(task calendar.Task, weekly bool) string {
	if weekly {
		return weekday(task.Due()) + " · "
	}
	return ""
}

func weekday(t time.Time) string {
	return t.Weekday().String()
}

func linked(text, link string) string {
	escaped := markdownEscaper.Replace(text)
	if link == "" {
		return "**" + escaped + "**"
	}
	return "[**" + escaped + "**](" + link + ")"
}

var markdownEscaper = strings.NewReplacer(
	`\`, `\\`,
	"*", `\*`,
	"_", `\_`,
	"[", `\[`,
	"]", `\]`,
	"`", "\\`",
)
