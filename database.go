package clocked

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"gopkg.in/v1/yaml"
)

const ActiveCodeFilename = "activeCode"
const TasksFolder = "tasks"

type Database struct {
	taskIndex     []Task
	taskCodeIndex map[string]struct{}
	activeCode    string
	rootFolder    string
	log           *logrus.Logger
}

func (d *Database) ActiveCode() string {
	return d.activeCode
}

func NewDatabase(path string, log *logrus.Logger) (*Database, error) {
	d := Database{
		rootFolder:    path,
		log:           log,
		taskCodeIndex: make(map[string]struct{}),
	}
	return &d, nil
}

func (d *Database) LoadState() error {
	d.log.Infof("Loading state")
	activeCodeFile := filepath.Join(d.rootFolder, ActiveCodeFilename)
	tasksFolder := filepath.Join(d.rootFolder, TasksFolder)

	activeCodeData, err := ioutil.ReadFile(activeCodeFile)
	if err != nil {
		if os.IsNotExist(err) {
			d.activeCode = ""
		} else {
			return err
		}
	} else {
		d.activeCode = strings.TrimSpace(string(activeCodeData))
	}

	files, err := filepath.Glob(filepath.Join(tasksFolder, "*.yml"))
	if err != nil {
		return err
	}
	for _, f := range files {
		d.log.Infof("Loading task from %s", f)
		t, err := d.loadTask(f)
		if err != nil {
			return err
		}
		d.taskIndex = append(d.taskIndex, *t)
		d.taskCodeIndex[t.Code] = struct{}{}
	}
	return nil
}

func (d *Database) loadTask(path string) (*Task, error) {
	baseName := filepath.Base(path)
	segments := strings.Split(baseName, ".")
	if len(segments) < 2 {
		return nil, fmt.Errorf("%s doesn't match the filename pattern {code}.yml", path)
	}
	code := segments[0]
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Task
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	t.Code = code
	return &t, nil
}

func (d *Database) saveTask(tsk *Task) error {
	path := filepath.Join(d.rootFolder, "tasks", fmt.Sprintf("%s.yml", tsk.Code))
	data, err := yaml.Marshal(tsk)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0600)
}

func (d *Database) AddTask(t Task) error {
	if d.taskCodeIndex == nil {
		d.taskCodeIndex = make(map[string]struct{})
	}
	if _, found := d.taskCodeIndex[t.Code]; found {
		return fmt.Errorf("The database already contains a task with this code.")
	}
	d.taskIndex = append(d.taskIndex, t)
	d.taskCodeIndex[t.Code] = struct{}{}
	return d.saveTask(&t)
}

func (d *Database) ClockInto(code string) error {
	// If another task is active, clock out of that first
	if d.activeCode != "" {
		if err := d.ClockOutOf(d.activeCode); err != nil {
			return err
		}
	}
	if _, ok := d.taskCodeIndex[code]; !ok {
		return fmt.Errorf("Task %s not found", code)
	}
	for idx := range d.taskIndex {
		if err := d.setActiveCode(code); err != nil {
			return err
		}
		task := (&d.taskIndex[idx])
		if err := task.Start(time.Now()); err != nil {
			return err
		}
		return d.saveTask(task)
	}
	return nil
}

func (d *Database) ClockOutOf(code string) error {
	if _, ok := d.taskCodeIndex[code]; !ok {
		return fmt.Errorf("Task %s not found", code)
	}
	for idx := range d.taskIndex {
		if err := d.setActiveCode(""); err != nil {
			return err
		}
		task := (&d.taskIndex[idx])
		if err := task.Stop(time.Now()); err != nil {
			return err
		}
		return d.saveTask(task)
	}
	return nil
}

func (d *Database) AllTasks() ([]Task, error) {
	return d.taskIndex, nil
}

func (d *Database) setActiveCode(code string) error {
	path := filepath.Join(d.rootFolder, ActiveCodeFilename)
	if err := os.MkdirAll(d.rootFolder, 0700); err != nil {
		return err
	}
	d.activeCode = code
	return ioutil.WriteFile(path, []byte(code), 0600)
}
