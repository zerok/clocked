package main

import (
	"fmt"

	termbox "github.com/nsf/termbox-go"
	"github.com/zerok/clocked"
)

type tasklistView struct {
	app                  *application
	list                 *ScrollableList
	taskStatusLineHeight int
	filterLineHeight     int
	filter               string
	filterFocused        bool
}

func (v *tasklistView) BeforeFocus() error {
	v.updateTaskList()
	return nil
}

func newTasklistView(app *application) *tasklistView {
	return &tasklistView{
		app:  app,
		list: NewScrollableList(Area{}),
	}
}

func (v *tasklistView) updateTaskList() {
	a := v.app
	tasks, _ := a.db.FilteredTasks(v.filter)
	items := make([]ScrollableListItem, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, t)
	}
	v.list.UpdateItems(items)
}

func (v *tasklistView) Render(area Area) error {
	if err := v.renderFilter(area); err != nil {
		return err
	}
	if err := v.renderActiveTask(area); err != nil {
		return err
	}
	if err := v.recalculateListDimensions(area); err != nil {
		return err
	}
	if err := v.renderList(); err != nil {
		return err
	}
	return nil
}

func (v *tasklistView) recalculateListDimensions(area Area) error {
	v.list.UpdateArea(Area{
		X:      area.X,
		Y:      area.Y,
		Width:  area.Width,
		Height: area.Height - v.taskStatusLineHeight - v.filterLineHeight,
	})
	return nil
}

// renderActiveTask renders the currently active task on top of the filter
// line.
func (v *tasklistView) renderActiveTask(area Area) error {
	yOffset := area.YMax() - v.filterLineHeight - 1
	xOffset := area.XMin()
	if v.app.db.ActiveCode() == "" {
		v.taskStatusLineHeight = 0
		return nil
	}
	task, ok := v.app.db.ActiveTask()
	if !ok {
		v.taskStatusLineHeight = 0
		return nil
	}
	v.app.drawLine(yOffset)
	xOffset = v.app.drawLabel(xOffset, yOffset+1, "Active task: ", false)
	xOffset = v.app.drawText(xOffset, yOffset+1, v.app.db.ActiveCode(), termbox.AttrBold|termbox.ColorGreen, termbox.ColorDefault)
	v.app.drawText(xOffset+1, yOffset+1, fmt.Sprintf("(%s)", task.Title), termbox.ColorWhite, termbox.ColorDefault)
	v.taskStatusLineHeight = 2
	return nil
}

func (v *tasklistView) renderFilter(area Area) error {
	a := v.app
	yOffset := area.YMax()
	a.drawLine(yOffset - 1)
	xOffset := a.drawLabel(area.XMin(), yOffset, "Search:", v.filterFocused)
	a.drawFieldValue(xOffset+1, area.XMax(), yOffset, v.filter, v.filterFocused)
	v.filterLineHeight = 2
	return nil
}

func (v *tasklistView) selectFirstRow() {
	v.list.SelectItemByIndex(0)
}

func (v *tasklistView) selectNextRow() {
	v.list.Next()
}

func (v *tasklistView) selectPreviousRow() {
	v.list.Previous()
}

func (v *tasklistView) renderList() error {
	v.list.Render()
	return nil
}

func (v *tasklistView) pushFilter(c rune) {
	v.filter = appendRune(v.filter, c)
	v.updateTaskList()
	v.selectFirstRow()
}

func (v *tasklistView) clearFilter() {
	v.filter = ""
	v.updateTaskList()
	v.selectFirstRow()
}

func (v *tasklistView) popFilter() {
	if len(v.filter) == 0 {
		return
	}
	v.filter = v.filter[0 : len(v.filter)-1]
	v.updateTaskList()
	v.selectFirstRow()
}

func (v *tasklistView) jumpToActiveTask() bool {
	task, ok := v.app.db.ActiveTask()
	if !ok {
		return false
	}
	_, ok = v.list.SelectItemByLabel(task.Label())
	return ok
}

func (v *tasklistView) HandleKeyEvent(evt termbox.Event) error {
	a := v.app
	switch {
	case v.filterFocused && evt.Key == termbox.KeyEsc:
		v.filterFocused = false
		v.clearFilter()
	case v.filterFocused && evt.Key == termbox.KeyEnter:
		v.filterFocused = false
	case v.filterFocused && (evt.Key == termbox.KeyBackspace || evt.Key == termbox.KeyBackspace2):
		v.popFilter()
	case v.filterFocused:
		v.pushFilter(evt.Ch)
	case evt.Key == termbox.KeyCtrlN || evt.Ch == 'n':
		a.switchMode(newTaskMode)
	case evt.Key == termbox.KeyCtrlA || evt.Ch == 'a':
		v.clearFilter()
		v.jumpToActiveTask()
	case evt.Ch == 'g':
		v.clearFilter()
	case evt.Key == termbox.KeyCtrlF || evt.Ch == 'f':
		v.filterFocused = true
	case evt.Key == termbox.KeyArrowDown || evt.Ch == 'j':
		v.selectNextRow()
	case evt.Key == termbox.KeyArrowUp || evt.Ch == 'k':
		v.selectPreviousRow()
	case evt.Key == termbox.KeyEnter:
		selectedItem, selected := v.list.SelectedItem()
		if !selected {
			return nil
		}
		selectedTask := selectedItem.(clocked.Task)
		if a.db.ActiveCode() == selectedTask.Code {
			if err := a.db.ClockOutOf(a.db.ActiveCode()); err != nil {
				a.err = err
			}
			return nil
		}
		if err := a.db.ClockInto(selectedTask.Code); err != nil {
			a.err = err
			return nil
		}
		if a.backup != nil {
			if err := a.backup.CreateSnapshot(); err != nil {
				a.fatalError(err, "failed to create snapshot")
			}
		}
		v.clearFilter()
	default:
	}
	return nil
}