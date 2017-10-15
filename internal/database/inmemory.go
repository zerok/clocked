package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/zerok/clocked"
)

type InMemory struct {
	tasks      []clocked.Task
	taskmap    map[string]int
	activeCode string
}

func NewInMemory() *InMemory {
	db := &InMemory{
		tasks:   make([]clocked.Task, 0, 10),
		taskmap: make(map[string]int),
	}
	return db
}

func (d *InMemory) TaskByCode(code string) (clocked.Task, bool) {
	idx, found := d.taskmap[code]
	if !found {
		return clocked.Task{}, false
	}
	return d.tasks[idx], false
}

func (d *InMemory) ActiveCode() string {
	return d.activeCode
}

func (d *InMemory) ActiveTask() (clocked.Task, bool) {
	if d.activeCode == "" {
		return clocked.Task{}, false
	}
	for _, t := range d.tasks {
		if t.Code == d.activeCode {
			return t, true
		}
	}
	return clocked.Task{}, false
}

func (d *InMemory) AddTask(t clocked.Task) error {
	_, exists := d.taskmap[t.Code]
	if exists {
		return fmt.Errorf("a task with this code already exists")
	}
	d.tasks = append(d.tasks, t)
	d.taskmap[t.Code] = len(d.tasks) - 1
	return nil
}

func (d *InMemory) UpdateTask(oldCode string, task clocked.Task) error {
	return fmt.Errorf("not implemented")
}

func (d *InMemory) AllTasks() ([]clocked.Task, error) {
	return d.tasks, nil
}

func (d *InMemory) Empty() bool {
	return d.tasks == nil || len(d.tasks) == 0
}

func (d *InMemory) ClockInto(code string) error {
	taskIdx, exists := d.taskmap[code]
	if !exists {
		return fmt.Errorf("the requested task does not exist")
	}
	if d.activeCode != "" {
		if err := d.ClockOutOf(code); err != nil {
			return err
		}
	}
	(&d.tasks[taskIdx]).Start(time.Now())
	d.activeCode = code
	return nil
}

func (d *InMemory) ClockOutOf(code string) error {
	taskIdx, exists := d.taskmap[code]
	if !exists {
		return fmt.Errorf("the requested task does not exist")
	}
	(&d.tasks[taskIdx]).Stop(time.Now())
	d.activeCode = ""
	return nil
}

func (d *InMemory) LoadState() error {
	return nil
}

func (d *InMemory) GenerateDailySummary(t time.Time) Summary {
	return Summary{}
}

func (d *InMemory) FilteredTasks(filter string) ([]clocked.Task, error) {
	res := make([]clocked.Task, 0, 5)
	f := strings.ToLower(filter)
	for _, t := range d.tasks {
		if strings.Contains(strings.ToLower(t.Label()), f) {
			res = append(res, t)
		}
	}
	return res, nil
}
