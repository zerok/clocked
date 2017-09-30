package main

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/Sirupsen/logrus"
	termbox "github.com/nsf/termbox-go"
	"github.com/zerok/clocked"
	"github.com/zerok/clocked/internal/backup"
	"github.com/zerok/clocked/internal/database"
	"github.com/zerok/clocked/internal/form"
)

const (
	newTaskMode   = iota
	selectionMode = iota
	summaryMode   = iota
	filterMode    = iota
)

type application struct {
	summaryViewDate      *time.Time
	log                  *logrus.Logger
	form                 *form.Form
	backup               *backup.Backup
	err                  error
	mode                 int
	area                 Area
	filterLineHeight     int
	taskStatusLineHeight int
	errorLineHeight      int
	focusedField         string
	taskListView         *ScrollableList
	db                   database.Database
	numRows              int
	filter               string
	visibleTaskCodes     []string
}

func selectByCode(code string) ItemMatcherFunc {
	return func(i ScrollableListItem) bool {
		task, ok := i.(clocked.Task)
		if !ok {
			return false
		}
		return task.Code == code
	}
}

func newApplication() *application {
	a := &application{}
	a.taskListView = NewScrollableList(Area{})
	return a
}

func (a *application) start() {
	a.reset()
	a.updateTaskList()
	a.redrawAll()
	for {
		a.handleResize()
		switch evt := termbox.PollEvent(); evt.Type {
		case termbox.EventResize:
			a.handleResize()
		case termbox.EventKey:
			if !a.handleKey(evt) {
				return
			}
		}
		a.redrawAll()
	}
}

func (a *application) handleResize() {
	w, h := termbox.Size()
	a.area.Width = w
	a.area.Height = h

	a.taskListView.UpdateArea(Area{
		X:      a.area.X,
		Y:      a.area.Y,
		Width:  a.area.Width,
		Height: h - a.taskStatusLineHeight - a.filterLineHeight - a.errorLineHeight,
	})
}

func (a *application) handleKey(evt termbox.Event) bool {
	if evt.Key == termbox.KeyCtrlC {
		return false
	}
	if evt.Key == termbox.KeyCtrlS {
		a.switchMode(summaryMode)
		return true
	}
	switch a.mode {
	case selectionMode:
		a.handleSelectionModeKey(evt)
	case newTaskMode:
		a.handleNewTaskModeKey(evt)
	case summaryMode:
		a.handleSummaryModeKey(evt)
	case filterMode:
		a.handleFilterModeKey(evt)
	}
	return true
}

func convertToTask(f *form.Form) clocked.Task {
	return clocked.Task{
		Code:  f.Value("code"),
		Title: f.Value("title"),
	}
}

func (a *application) handleFilterModeKey(evt termbox.Event) {
	switch evt.Key {
	case termbox.KeyEsc:
		a.clearFilter()
		a.switchMode(selectionMode)
	case termbox.KeyCtrlG:
		a.clearFilter()
		a.switchMode(selectionMode)
	case termbox.KeyEnter:
		a.switchMode(selectionMode)
	case termbox.KeyBackspace:
		a.popFilter()
	case termbox.KeyBackspace2:
		a.popFilter()
	default:
		a.pushFilter(evt.Ch)
	}
}

func (a *application) handleSummaryModeKey(evt termbox.Event) {
	switch evt.Key {
	case termbox.KeyEsc:
		a.switchMode(selectionMode)
		a.summaryViewDate = nil
	default:
		dateDelta := 0

		switch evt.Ch {
		case 'j':
			dateDelta = 1
		case 'k':
			dateDelta = -1
		}

		if dateDelta != 0 {
			nextDate := time.Now()
			if a.summaryViewDate == nil {
				nextDate = nextDate.AddDate(0, 0, dateDelta)
			} else {
				nextDate = a.summaryViewDate.AddDate(0, 0, dateDelta)
			}
			a.summaryViewDate = &nextDate
		}
	}
}

func (a *application) fatalError(err error, msg string, args ...interface{}) {
	termbox.Close()
	a.log.WithError(err).Fatalf(msg, args...)
}

func (a *application) handleNewTaskModeKey(evt termbox.Event) {
	switch evt.Key {
	case termbox.KeyEsc:
		a.switchMode(selectionMode)
	case termbox.KeyTab:
		a.form.Next()
	case termbox.KeyEnter:
		if a.form.Validate() {
			t := convertToTask(a.form)
			if err := a.db.AddTask(t); err != nil {
				a.err = err
				break
			}
			if a.backup != nil {
				if err := a.backup.CreateSnapshot(); err != nil {
					a.fatalError(err, "failed to create snapshot")
				}
			}
			a.log.Infof("%s added", t)
			a.switchMode(selectionMode)
			a.updateTaskList()
			a.taskListView.SelectMatchingItem(selectByCode(t.Code))
		}
	default:
		a.handleFieldInput(evt)
	}
}

func (a *application) handleSelectionModeKey(evt termbox.Event) {
	switch {
	case evt.Key == termbox.KeyCtrlN || evt.Ch == 'n':
		a.switchMode(newTaskMode)
		a.form = generateNewTaskForm()
	case evt.Key == termbox.KeyCtrlA || evt.Ch == 'a':
		a.clearFilter()
		a.jumpToActiveTask()
	case evt.Ch == 'g':
		a.clearFilter()
	case evt.Key == termbox.KeyCtrlF || evt.Ch == 'f':
		a.switchMode(filterMode)
	case evt.Key == termbox.KeyArrowDown || evt.Ch == 'j':
		a.selectNextRow()
	case evt.Key == termbox.KeyArrowUp || evt.Ch == 'k':
		a.selectPreviousRow()
	case evt.Key == termbox.KeyEnter:
		selectedItem, selected := a.taskListView.SelectedItem()
		if !selected {
			return
		}
		selectedTask := selectedItem.(clocked.Task)
		if a.db.ActiveCode() == selectedTask.Code {
			if err := a.db.ClockOutOf(a.db.ActiveCode()); err != nil {
				a.err = err
			}
			return
		}
		if err := a.db.ClockInto(selectedTask.Code); err != nil {
			a.err = err
			return
		}
		if a.backup != nil {
			if err := a.backup.CreateSnapshot(); err != nil {
				a.fatalError(err, "failed to create snapshot")
			}
		}
		a.clearFilter()
	default:
	}
}

func (a *application) handleFieldInput(evt termbox.Event) {
	v := a.form.Value(a.focusedField)

	if evt.Key == termbox.KeyBackspace || evt.Key == termbox.KeyBackspace2 {
		if len(v) == 0 {
			return
		}
		v = v[0 : len(v)-1]
	} else {
		v = appendRune(v, evt.Ch)
	}
	a.form.SetValue(a.focusedField, v)
}

func (a *application) reset() {
	termbox.Clear(termbox.ColorWhite, termbox.ColorDefault)
}

func (a *application) redrawFilter(xOffset, maxXOffset, yOffset int) {
	yOffset = a.area.YMax()
	a.drawLine(yOffset - 1)
	xOffset = a.drawLabel(xOffset, yOffset, "Search:", a.mode == filterMode)
	a.drawFieldValue(xOffset+1, maxXOffset, yOffset, a.filter, a.mode == filterMode)
	a.filterLineHeight = 2
}

func (a *application) drawLabel(xOffset, yOffset int, text string, focused bool) int {
	fg := termbox.ColorWhite
	bg := termbox.ColorDefault
	if focused {
		fg |= termbox.AttrBold
	}
	return a.drawText(xOffset, yOffset, text, fg, bg)
}

func (a *application) drawText(xOffset, yOffset int, text string, fg, bg termbox.Attribute) int {
	var drawnChars int
	for idx, c := range text {
		drawnChars++
		if xOffset+idx == a.area.XMax() {
			termbox.SetCell(xOffset+idx, yOffset, '\u2026', fg, bg)
			break
		} else {
			termbox.SetCell(xOffset+idx, yOffset, c, fg, bg)
		}
	}
	return xOffset + drawnChars
}

func (a *application) updateTaskList() {
	tasks, _ := a.db.FilteredTasks(a.filter)
	items := make([]ScrollableListItem, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, t)
	}
	a.taskListView.UpdateItems(items)
}

func (a *application) redrawAll() {
	a.reset()
	yOffset := a.redrawError(2, 1)
	switch {
	case a.mode == summaryMode:
		a.redrawSummary()
	case a.mode == selectionMode || a.mode == filterMode:
		a.redrawFilter(a.area.XMin(), a.area.Width, yOffset)
		a.redrawActiveTask(a.area.XMin(), a.area.YMax()-3)
		// TODO: Only update the task list on actual actions that change the
		//       list of selected tasks.
		a.recalculateDimensions()
		a.taskListView.Render()
	case a.mode == newTaskMode:
		a.redrawForm(a.area.XMin(), yOffset)
	}
	termbox.Flush()
}
func formatTime(t *time.Time) string {
	if t == nil {
		return "..."
	}
	return t.Format("15:04:05")
}
func (a *application) redrawSummary() {
	var now time.Time
	if a.summaryViewDate != nil {
		now = *a.summaryViewDate
	} else {
		now = time.Now()
	}
	a.drawText(a.area.XMin(), a.area.YMin(), fmt.Sprintf("Summary for %s", now.Format("Mon, 2 Jan 2006")), termbox.ColorBlue|termbox.AttrBold, termbox.ColorDefault)
	summary := a.db.GenerateDailySummary(now)
	for idx, b := range summary.Bookings {
		a.drawText(a.area.XMin(), a.area.YMin()+1+idx, fmt.Sprintf("%s - %s (%s)", formatTime(b.Start), formatTime(b.Stop), b.Code), termbox.ColorDefault, termbox.ColorDefault)
	}

	idx := 0
	for key, dur := range summary.Totals {
		a.drawText(a.area.XMin()+a.area.Width/2, a.area.YMin()+1+idx, fmt.Sprintf("%s: %s", key, dur), termbox.ColorDefault, termbox.ColorDefault)
		idx++
	}
	idx++
	a.drawText(a.area.XMin()+a.area.Width/2, a.area.YMin()+1+idx, "Total: ", termbox.ColorDefault|termbox.AttrBold, termbox.ColorDefault)
	a.drawText(a.area.XMin()+a.area.Width/2+7, a.area.YMin()+1+idx, summary.Total.String(), termbox.ColorDefault, termbox.ColorDefault)
}

func (a *application) recalculateDimensions() {
	a.taskListView.UpdateArea(Area{
		X:      0,
		Y:      0,
		Width:  a.area.Width,
		Height: a.area.Height - a.taskStatusLineHeight - a.filterLineHeight - a.errorLineHeight,
	})
}

func (a *application) redrawActiveTask(xOffset, yOffset int) int {
	if a.db.ActiveCode() == "" {
		a.taskStatusLineHeight = 0
		return 0
	}
	task, ok := a.db.ActiveTask()
	if !ok {
		a.taskStatusLineHeight = 0
		return 0
	}
	a.drawLine(yOffset)
	xOffset = a.drawLabel(xOffset, yOffset+1, "Active task: ", false)
	xOffset = a.drawText(xOffset, yOffset+1, a.db.ActiveCode(), termbox.AttrBold|termbox.ColorGreen, termbox.ColorDefault)
	a.drawText(xOffset+1, yOffset+1, fmt.Sprintf("(%s)", task.Title), termbox.ColorWhite, termbox.ColorDefault)
	a.taskStatusLineHeight = 2
	return 2
}

func (a *application) redrawError(xOffset, yOffset int) int {
	if a.err != nil {
		a.drawError(xOffset, yOffset, a.err.Error())
		a.errorLineHeight = 1
		return 1
	}
	a.errorLineHeight = 0
	return 0
}

func (a *application) redrawForm(xOffset, initialYOffset int) {
	xOffset++
	yOffset := initialYOffset + 1
	inputStartOffset := 0
	a.focusedField = ""
	for _, fld := range a.form.Fields() {
		isFocused := a.form.IsFocused(fld.Code)
		if isFocused && a.focusedField != fld.Code {
			a.focusedField = fld.Code
		}
		afterLabel := a.drawLabel(xOffset, yOffset, fld.Label, isFocused)
		if afterLabel+1 > inputStartOffset {
			inputStartOffset = afterLabel + 1
		}
		yOffset += 3
	}

	yOffset = initialYOffset + 1
	for _, fld := range a.form.Fields() {
		a.drawFieldValue(inputStartOffset, a.area.Width-2, yOffset, fld.Value, fld.Code == a.focusedField)
		a.drawError(inputStartOffset, yOffset+1, fld.Error)
		yOffset += 3
	}
}

func (a *application) drawError(xOffset, yOffset int, msg string) int {
	for idx, c := range msg {
		termbox.SetCell(xOffset+idx, yOffset, c, termbox.ColorRed, termbox.ColorDefault)
	}
	return xOffset + len(msg)
}

func (a *application) drawFieldValue(xOffset int, xOffsetEnd int, yOffset int, value string, focused bool) int {
	bg := termbox.ColorBlack
	fg := termbox.ColorWhite
	valueIndex := -1
	for xOffset < xOffsetEnd {
		c := ' '
		if valueIndex >= 0 && valueIndex < len(value) {
			c = rune(value[valueIndex])
		}
		termbox.SetCell(xOffset, yOffset, c, fg, bg)
		xOffset++
		valueIndex++
	}
	return xOffsetEnd
}

func (a *application) jumpToActiveTask() bool {
	task, ok := a.db.ActiveTask()
	if !ok {
		return false
	}
	_, ok = a.taskListView.SelectItemByLabel(task.Label())
	return ok
}

func (a *application) drawTaskLine(task *clocked.Task, xOffset, maxXOffset, yOffset int, isSelected bool, isActive bool) {
	if isSelected {
		termbox.SetCell(xOffset+1, yOffset, '>', termbox.ColorWhite|termbox.AttrBold, termbox.ColorDefault)
	}
	xOffset += 2
	var fg termbox.Attribute
	if isActive {
		fg = termbox.ColorGreen
	}
	if isSelected {
		fg |= termbox.AttrBold
	}

	xOffset = a.drawText(xOffset+1, yOffset, task.Code, fg, termbox.ColorDefault)
	a.drawLabel(xOffset+1, yOffset, task.Title, isSelected)
}

func (a *application) selectFirstRow() {
	a.taskListView.SelectItemByIndex(0)
}

func (a *application) selectNextRow() {
	a.taskListView.Next()
}

func (a *application) selectPreviousRow() {
	a.taskListView.Previous()
}

func appendRune(s string, r rune) string {
	b := make([]byte, 3)
	l := utf8.EncodeRune(b, r)
	return s + string(b[0:l])
}

func (a *application) pushFilter(c rune) {
	a.filter = appendRune(a.filter, c)
	a.updateTaskList()
	a.selectFirstRow()
}

func (a *application) clearFilter() {
	a.filter = ""
	a.updateTaskList()
	a.selectFirstRow()
}

func (a *application) popFilter() {
	if len(a.filter) == 0 {
		return
	}
	a.filter = a.filter[0 : len(a.filter)-1]
	a.updateTaskList()
	a.selectFirstRow()
}

func (a *application) drawLine(yOffset int) {
	for i := a.area.XMin(); i <= a.area.XMax(); i++ {
		termbox.SetCell(i, yOffset, '\u2500', termbox.ColorBlue, termbox.ColorDefault)
	}
}

func (a *application) switchMode(mode int) {
	a.mode = mode
	a.form = nil
	a.focusedField = ""
	a.err = nil
}
