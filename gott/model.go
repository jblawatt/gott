package gott

import (
	"time"

	uuid "github.com/nu7hatch/gouuid"
)

const (
	StatusStarted = "started"
	StatusEnded   = "ended"
)

type Interval struct {
	// Interval Status
	ID         string
	Begin      time.Time
	End        time.Time
	Duration   time.Duration
	Tags       []string
	Project    string
	Ref        string
	Annotation string
	Raw        string
	UDA        map[string]interface{}
	Status     string
}

func (i *Interval) GetDuration() time.Duration {
	// completely the same. duration only
	if i.End.Equal(i.Begin) {
		return i.Duration
	}
	if i.End.IsZero() {
		return time.Now().Sub(i.Begin)
	}
	return i.End.Sub(i.Begin)
}

func (i *Interval) Stop() {
	i.End = time.Now()
	i.Status = StatusEnded
}

func NewInterval(raw []string) (interval Interval) {
	id, _ := uuid.NewV4()
	interval.ID = id.String()
	lexInterval(raw, &interval)
	return interval
}
