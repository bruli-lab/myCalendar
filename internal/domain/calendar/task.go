package calendar

import "time"

type Task struct {
	title string
	due   time.Time
	link  string
}

func (t Task) Title() string {
	return t.title
}

func (t Task) Due() time.Time {
	return t.due
}

func (t Task) Link() string {
	return t.link
}

func NewTask(title string, due time.Time, link string) Task {
	if title == "" {
		title = untitled
	}
	return Task{title: title, due: dateInUTC(due), link: link}
}
