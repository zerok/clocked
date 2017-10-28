package clocked

import (
	"fmt"
	"strings"
	"time"
)

type Task struct {
	Code     string    `yaml:"code"`
	Title    string    `yaml:"title"`
	Tags     []string  `yaml:"tags"`
	Bookings []Booking `yaml:"bookings"`
}

func (t *Task) HasTag(tag string) bool {
	for _, tg := range t.Tags {
		if tg == tag {
			return true
		}
	}
	return false
}

func (t *Task) Start(tm time.Time) error {
	b := Booking{}
	b.SetStart(tm)
	t.Bookings = append(t.Bookings, b)
	return nil
}

func (t *Task) Stop(tm time.Time) error {
	b := &t.Bookings[len(t.Bookings)-1]
	b.SetStop(tm)
	return nil
}

func (t Task) Label() string {
	return fmt.Sprintf("%s %s", t.Code, t.Title)
}

type Booking struct {
	Start string `yaml:"start"`
	Stop  string `yaml:"stop"`
}

func (b *Booking) SetStart(t time.Time) {
	b.Start = t.Format(time.RFC3339)
}

func (b *Booking) SetStop(t time.Time) {
	b.Stop = t.Format(time.RFC3339)
}

func (b *Booking) StartTime() *time.Time {
	if b.Start == "" {
		return nil
	}
	t, _ := time.Parse(time.RFC3339, b.Start)
	return &t
}

func (b *Booking) StopTime() *time.Time {
	if b.Stop == "" {
		return nil
	}
	t, _ := time.Parse(time.RFC3339, b.Stop)
	return &t
}

type ByCode []Task

func (l ByCode) Len() int {
	return len(l)
}

func (l ByCode) Less(i int, j int) bool {
	return strings.Compare(l[i].Code, l[j].Code) < 0
}

func (l ByCode) Swap(i int, j int) {
	l[i], l[j] = l[j], l[i]
}
