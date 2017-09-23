package clocked

import "time"

type Task struct {
	Code     string    `yaml:"code"`
	Title    string    `yaml:"title"`
	Tags     []string  `yaml:"tags"`
	Bookings []Booking `yaml:"bookings"`
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
