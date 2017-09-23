package main

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	termbox "github.com/nsf/termbox-go"
	"github.com/zerok/clocked"
	"github.com/zerok/clocked/internal/form"
)

const (
	newTaskMode   = iota
	selectionMode = iota
)

type application struct {
	log              *logrus.Logger
	form             *form.Form
	err              error
	mode             int
	width            int
	height           int
	minXOffset       int
	maxXOffset       int
	minYOffset       int
	maxYOffset       int
	focusedField     string
	db               *clocked.Database
	selectedTaskCode string
	numRows          int
	filter           string
	visibleTaskCodes []string
}

func (a *application) start() {
	a.reset()
	a.redrawAll()
	for {
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
	a.setWidth(w)
	a.setHeight(h)
}

func (a *application) handleKey(evt termbox.Event) bool {
	if evt.Key == termbox.KeyCtrlC {
		return false
	}
	switch a.mode {
	case selectionMode:
		a.handleSelectionModeKey(evt)
	case newTaskMode:
		a.handleNewTaskModeKey(evt)
	}
	return true
}

func convertToTask(f *form.Form) clocked.Task {
	return clocked.Task{
		Code:  f.Value("code"),
		Title: f.Value("title"),
	}
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
			a.log.Infof("%s added", t)
			a.switchMode(selectionMode)
			a.selectedTaskCode = t.Code
		}
	default:
		a.handleFieldInput(evt)
	}
}

func (a *application) handleSelectionModeKey(evt termbox.Event) {
	switch evt.Key {
	case termbox.KeyBackspace:
		a.popFilter()
	case termbox.KeyBackspace2:
		a.popFilter()
	case termbox.KeyArrowDown:
		a.selectNextRow()
	case termbox.KeyArrowUp:
		a.selectPreviousRow()
	case termbox.KeyTab:
	case termbox.KeyEnter:
		if a.db.ActiveCode() == a.selectedTaskCode {
			if err := a.db.ClockOutOf(a.selectedTaskCode); err != nil {
				a.err = err
			}
			return
		}
		if err := a.db.ClockInto(a.selectedTaskCode); err != nil {
			a.err = err
			return
		}
	default:
		if evt.Ch == 43 {
			a.switchMode(newTaskMode)
			a.form = generateNewTaskForm()
			return
		}
		a.pushFilter(evt.Ch)
	}
}

func (a *application) setWidth(w int) {
	a.width = w
	a.maxXOffset = w - 2
	a.minXOffset = 1
}

func (a *application) setHeight(h int) {
	a.height = h
	a.maxYOffset = h - 2
	a.minYOffset = 2
	if a.minYOffset > a.maxYOffset {
		a.minYOffset = a.maxYOffset
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
		v = v + string(evt.Ch)
	}
	a.form.SetValue(a.focusedField, v)
}

func (a *application) reset() {
	termbox.Clear(termbox.ColorWhite, termbox.ColorDefault)
}

func (a *application) redrawFilter(xOffset, maxXOffset, yOffset int) {
	yOffset = a.height - 1
	a.drawLine(yOffset - 1)
	xOffset = a.drawLabel(xOffset, yOffset, "Search:", false)
	a.drawFieldValue(xOffset+1, maxXOffset, yOffset, a.filter, false)
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
		if xOffset+idx == a.maxXOffset {
			termbox.SetCell(xOffset+idx, yOffset, '\u2026', fg, bg)
			break
		} else {
			termbox.SetCell(xOffset+idx, yOffset, c, fg, bg)
		}
	}
	return xOffset + drawnChars
}

func (a *application) redrawTasklist(xOffset, maxXOffset, yOffset int) {
	tasks, _ := a.db.AllTasks()
	num := 0
	normalizedFilter := strings.ToLower(a.filter)
	idx := 0
	visibleTaskCodes := make([]string, 0, 10)
	for _, task := range tasks {
		normalizedTask := strings.ToLower(task.Title)
		if len(a.filter) == 0 || strings.Contains(normalizedTask, normalizedFilter) {
			visibleTaskCodes = append(visibleTaskCodes, task.Code)
			a.drawTaskLine(&task, xOffset, maxXOffset, yOffset+idx+1, task.Code == a.selectedTaskCode, a.db.ActiveCode() == task.Code)
			idx++
			num++
		}
	}
	a.numRows = num
	a.visibleTaskCodes = visibleTaskCodes
}

func (a *application) redrawAll() {
	a.reset()
	yOffset := a.redrawError(2, 1)
	if a.mode == selectionMode {
		a.redrawFilter(a.minXOffset, a.maxXOffset, yOffset)
		a.redrawTasklist(a.minXOffset, a.maxXOffset/2, yOffset)
		a.redrawActiveTask(a.minXOffset, a.maxYOffset-2)
	} else if a.mode == newTaskMode {
		a.redrawForm(a.minXOffset, yOffset)
	}
	termbox.Flush()
}

func (a *application) redrawActiveTask(xOffset, yOffset int) int {
	if a.db.ActiveCode() == "" {
		return 0
	}
	task, ok := a.db.ActiveTask()
	if !ok {
		return 0
	}
	a.drawLine(yOffset)
	xOffset = a.drawLabel(xOffset, yOffset+1, "Active task: ", false)
	xOffset = a.drawText(xOffset, yOffset+1, a.db.ActiveCode(), termbox.AttrBold|termbox.ColorGreen, termbox.ColorDefault)
	a.drawText(xOffset+1, yOffset+1, fmt.Sprintf("(%s)", task.Title), termbox.ColorWhite, termbox.ColorDefault)
	return 3
}

func (a *application) redrawError(xOffset, yOffset int) int {
	if a.err != nil {
		a.drawError(xOffset, yOffset, a.err.Error())
		return 1
	}
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
		a.drawFieldValue(inputStartOffset, a.width-2, yOffset, fld.Value, fld.Code == a.focusedField)
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

func (a *application) getCurrentTaskCodeIndex() int {
	for idx, code := range a.visibleTaskCodes {
		if code == a.selectedTaskCode {
			return idx
		}
	}
	return -1
}

func (a *application) selectNextRow() {
	cur := a.getCurrentTaskCodeIndex()
	if len(a.visibleTaskCodes) == 0 {
		return
	}
	if cur == -1 || cur+1 > len(a.visibleTaskCodes)-1 {
		a.selectedTaskCode = a.visibleTaskCodes[0]
		return
	}
	a.selectedTaskCode = a.visibleTaskCodes[cur+1]
}

func (a *application) selectPreviousRow() {
	cur := a.getCurrentTaskCodeIndex()
	if len(a.visibleTaskCodes) == 0 {
		return
	}
	if cur == -1 || cur-1 < 0 {
		a.selectedTaskCode = a.visibleTaskCodes[len(a.visibleTaskCodes)-1]
		return
	}
	a.selectedTaskCode = a.visibleTaskCodes[cur-1]
}

func (a *application) pushFilter(c rune) {
	a.filter += string(c)
}

func (a *application) popFilter() {
	if len(a.filter) == 0 {
		return
	}
	a.filter = a.filter[0 : len(a.filter)-1]
}

func (a *application) drawLine(yOffset int) {
	for i := 0; i < a.width; i++ {
		termbox.SetCell(i, yOffset, '\u2500', termbox.ColorBlue, termbox.ColorDefault)
	}
}

func (a *application) switchMode(mode int) {
	a.mode = mode
	a.form = nil
	a.focusedField = ""
	a.err = nil
}
