package backup

import (
	"fmt"
	"time"
)

type Snapshot struct {
	RawTime  string   `json:"time"`
	Tree     string   `json:"tree"`
	Paths    []string `json:"paths"`
	Hostname string   `json:"hostname"`
	ID       string   `json:"id"`
}

func (s *Snapshot) Time() (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s.RawTime)
}

func (s Snapshot) Label() string {
	t, err := s.Time()
	if err != nil {
		return fmt.Sprintf("<Error: %s>", err.Error())
	}
	return t.Format(time.RFC850)
}
