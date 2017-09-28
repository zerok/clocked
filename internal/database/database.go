package database

import (
	"time"

	"github.com/zerok/clocked"
)

// Database is the main abstraction for the datastore used by clocked. There
// exist multiple implementions mostly for making testing the frontend
// application easier.
type Database interface {
	ActiveCode() string
	ActiveTask() (clocked.Task, bool)
	LoadState() error
	AddTask(clocked.Task) error
	ClockInto(code string) error
	ClockOutOf(code string) error
	AllTasks() ([]clocked.Task, error)
	FilteredTasks(f string) ([]clocked.Task, error)
	GenerateDailySummary(time.Time) Summary
	Empty() bool
}
