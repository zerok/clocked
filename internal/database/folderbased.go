package database

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/zerok/clocked"
	"gopkg.in/yaml.v2"
)

const ActiveCodeFilename = "activeCode"
const TasksFolder = "tasks"

type FolderBasedDatabase struct {
	taskIndex     []clocked.Task
	taskCodeIndex map[string]struct{}
	activeCode    string
	rootFolder    string
	log           *logrus.Logger
}

func (d *FolderBasedDatabase) ActiveCode() string {
	return d.activeCode
}

func (d *FolderBasedDatabase) ActiveTask() (clocked.Task, bool) {
	if d.activeCode == "" {
		return clocked.Task{}, false
	}
	for _, task := range d.taskIndex {
		if task.Code == d.activeCode {
			return task, true
		}
	}
	return clocked.Task{}, false
}

func NewDatabase(path string, log *logrus.Logger) (Database, error) {
	d := FolderBasedDatabase{
		rootFolder:    path,
		log:           log,
		taskCodeIndex: make(map[string]struct{}),
	}
	return &d, nil
}

func (d *FolderBasedDatabase) LoadState() error {
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

func (d *FolderBasedDatabase) loadTask(path string) (*clocked.Task, error) {
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
	var t clocked.Task
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	t.Code = code
	return &t, nil
}

func (d *FolderBasedDatabase) saveTask(tsk *clocked.Task) error {
	path := filepath.Join(d.rootFolder, "tasks", fmt.Sprintf("%s.yml", tsk.Code))
	data, err := yaml.Marshal(tsk)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0600)
}

func (d *FolderBasedDatabase) AddTask(t clocked.Task) error {
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

func (d *FolderBasedDatabase) ClockInto(code string) error {
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
		task := (&d.taskIndex[idx])
		if code != task.Code {
			continue
		}
		if err := d.setActiveCode(code); err != nil {
			return err
		}
		if err := task.Start(time.Now()); err != nil {
			return err
		}
		return d.saveTask(task)
	}
	return nil
}

type Summary struct {
	Bookings []TaskBooking
	Totals   map[string]time.Duration
	Total    time.Duration
}

type TaskBooking struct {
	Code  string
	Start *time.Time
	Stop  *time.Time
}

type ByStart []TaskBooking

func (b ByStart) Len() int {
	return len(b)
}
func (b ByStart) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b ByStart) Less(i, j int) bool {
	aStart := b[i].Start
	bStart := b[j].Start
	if bStart == nil && aStart != nil {
		return true
	}
	if aStart == nil && bStart != nil {
		return false
	}
	if aStart == nil && bStart == nil {
		return false
	}
	return aStart.Before(*bStart)
}

func (d *FolderBasedDatabase) Empty() bool {
	return d.taskIndex == nil || len(d.taskIndex) == 0
}

func (d *FolderBasedDatabase) GenerateDailySummary(t time.Time) Summary {
	summary := Summary{}
	summary.Totals = make(map[string]time.Duration)
	summary.Bookings = make([]TaskBooking, 0, 10)
	for _, tsk := range d.taskIndex {
		if tsk.Bookings != nil {
			for _, b := range tsk.Bookings {
				start := b.StartTime()
				if start != nil {
					if isSameDay(*start, t) {
						summary.Bookings = append(summary.Bookings, TaskBooking{
							Code:  tsk.Code,
							Start: start,
							Stop:  b.StopTime(),
						})
						stop := b.StopTime()
						if stop != nil {
							prev := summary.Totals[tsk.Code]
							dur := stop.Sub(*start)
							summary.Totals[tsk.Code] = prev + dur
							summary.Total += dur
						}
					}
				}
			}
		}
	}
	sort.Sort(ByStart(summary.Bookings))
	return summary
}

func isSameDay(a time.Time, b time.Time) bool {
	aYear, aMonth, aDay := a.Date()
	bYear, bMonth, bDay := b.Date()
	return aYear == bYear && aMonth == bMonth && aDay == bDay
}

func (d *FolderBasedDatabase) ClockOutOf(code string) error {
	if _, ok := d.taskCodeIndex[code]; !ok {
		return fmt.Errorf("Task %s not found", code)
	}
	for idx := range d.taskIndex {
		task := (&d.taskIndex[idx])
		if code != task.Code {
			continue
		}
		if err := d.setActiveCode(""); err != nil {
			return err
		}
		if err := task.Stop(time.Now()); err != nil {
			return err
		}
		return d.saveTask(task)
	}
	return nil
}

func (d *FolderBasedDatabase) AllTasks() ([]clocked.Task, error) {
	return d.taskIndex, nil
}

func (d *FolderBasedDatabase) FilteredTasks(f string) ([]clocked.Task, error) {
	if f == "" {
		return d.AllTasks()
	}
	q := strings.ToLower(f)
	result := make([]clocked.Task, 0, 5)
	for _, t := range d.taskIndex {
		l := strings.ToLower(t.Label())
		if strings.Contains(l, q) {
			result = append(result, t)
		}
	}
	return result, nil
}

func (d *FolderBasedDatabase) setActiveCode(code string) error {
	path := filepath.Join(d.rootFolder, ActiveCodeFilename)
	if err := os.MkdirAll(d.rootFolder, 0700); err != nil {
		return err
	}
	d.activeCode = code
	return ioutil.WriteFile(path, []byte(code), 0600)
}
