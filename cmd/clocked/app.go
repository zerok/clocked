package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sirupsen/logrus"
	termbox "github.com/nsf/termbox-go"
	"github.com/zerok/clocked"
	"github.com/zerok/clocked/internal/backup"
	"github.com/zerok/clocked/internal/database"
	"github.com/zerok/clocked/internal/form"
	"github.com/zerok/clocked/internal/jira"
)

const (
	newTaskMode   = iota
	selectionMode = iota
	summaryMode   = iota
	filterMode    = iota
	syncMode      = iota
	editTaskMode  = iota
)

type application struct {
	summaryViewDate      *time.Time
	termLog              *logrus.Logger
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
	db                   database.Database
	numRows              int
	filter               string
	visibleTaskCodes     []string
	jiraClient           *jira.Client
	views                map[int]View
	activeView           View
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
	a := &application{
		termLog: logrus.New(),
	}
	a.views = map[int]View{
		summaryMode: &summaryView{
			app: a,
		},
		selectionMode: newTasklistView(a),
		newTaskMode:   newCreateTaskView(a),
		syncMode:      newSyncView(a),
		editTaskMode:  newEditTaskView(a),
	}
	return a
}

func (a *application) start() {
	a.reset()
	a.switchMode(selectionMode)
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
}

func (a *application) handleKey(evt termbox.Event) bool {
	if evt.Key == termbox.KeyCtrlC {
		return false
	}
	if evt.Key == termbox.KeyCtrlS {
		a.switchMode(summaryMode)
		return true
	}
	if a.activeView != nil {
		if err := a.activeView.HandleKeyEvent(evt); err != nil {
			if err == ErrCloseView {
				a.switchMode(selectionMode)
			}
		}
		return true
	}
	return true
}

func convertToTask(f *form.Form) clocked.Task {
	return clocked.Task{
		Code:  f.Value("code"),
		Title: f.Value("title"),
		Tags:  strings.Split(f.Value("tags"), " "),
	}
}

func (a *application) fatalError(err error, msg string, args ...interface{}) {
	termbox.Close()
	a.log.WithError(err).Fatalf(msg, args...)
}

func (a *application) drawHeadline(x, y int, text string) {
	a.drawText(x, y, text, termbox.ColorBlue|termbox.AttrBold, termbox.ColorDefault)
}

func (a *application) handleFieldInput(frm *form.Form, evt termbox.Event) {
	focusedField := frm.FocusedField()
	if focusedField == "" {
		return
	}
	v := frm.Value(focusedField)

	if evt.Key == termbox.KeyBackspace || evt.Key == termbox.KeyBackspace2 {
		if len(v) == 0 {
			return
		}
		v = v[0 : len(v)-1]
	} else {
		v = appendRune(v, evt.Ch)
	}
	frm.SetValue(focusedField, v)
}

func (a *application) selectTask(task clocked.Task) {
	if a.activeView == nil {
		return
	}
	if tcv, ok := a.activeView.(TaskCentricView); ok {
		tcv.SetTask(task)
	}
}

func (a *application) reset() {
	termbox.Clear(termbox.ColorWhite, termbox.ColorDefault)
	termbox.HideCursor()
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

func (a *application) drawKeyMapping(area Area, mapping []KeyMap) int {
	if len(mapping) == 0 {
		return 0
	}
	max := 0
	xOffset := area.XMin()
	maxX := area.XMax()
	line := 0
	lines := make([][]KeyMap, 0, 1)
	lines = append(lines, []KeyMap{})
	margin := 3

	for _, m := range mapping {
		if l := len(m.Key) + len(m.Label); l > max {
			max = l
		}
	}

	for _, m := range mapping {
		padding := max + 2 - len(m.Key) - len(m.Label)
		s := fmt.Sprintf("[%s]%s%s", m.Key, strings.Repeat(" ", padding), m.Label)
		l := len(s)
		currentMargin := 0
		if len(lines[line]) > 0 {
			currentMargin = margin
		}
		if currentMargin+xOffset+l > maxX {
			xOffset = 0
			line++
			lines = append(lines, []KeyMap{})
			currentMargin = 0
		}
		lines[line] = append(lines[line], m)
		xOffset += currentMargin + l
	}

	a.drawLine(area.YMax() - len(lines))
	for idx, l := range lines {
		xOffset := area.XMin()
		yOffset := area.YMax() - len(lines) + idx + 1
		for cellIdx, m := range l {
			if cellIdx > 0 {
				xOffset += margin
			}
			padding := max + 2 - len(m.Key) - len(m.Label)
			xOffset = a.drawText(xOffset, yOffset, fmt.Sprintf("[%s]%s", m.Key, strings.Repeat(" ", padding)), termbox.AttrBold|termbox.ColorWhite, termbox.ColorBlack)
			xOffset = a.drawText(xOffset, yOffset, m.Label, termbox.ColorWhite, termbox.ColorBlack)
		}
	}

	return 1 + len(lines)
}

func (a *application) redrawAll() {
	a.reset()
	yOffset := a.redrawError(2, 1)

	if a.activeView != nil {
		contentArea := Area(a.area)
		contentArea.Y = yOffset

		if km, ok := a.activeView.(Keymapper); ok {
			contentArea.Height -= a.drawKeyMapping(contentArea, km.KeyMapping())
		}

		a.activeView.Render(contentArea)
	}

	termbox.Flush()
}
func formatTime(t *time.Time) string {
	if t == nil {
		return "..."
	}
	return t.Format("15:04:05")
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

func (a *application) redrawForm(area Area, frm *form.Form) {
	xOffset := area.XMin() + 1
	yOffset := area.YMin() + 1
	inputStartOffset := 0
	a.focusedField = ""
	for _, fld := range frm.Fields() {
		isFocused := frm.IsFocused(fld.Code)
		if isFocused && a.focusedField != fld.Code {
			a.focusedField = fld.Code
		}
		afterLabel := a.drawLabel(xOffset, yOffset, fld.Label, isFocused)
		if afterLabel+1 > inputStartOffset {
			inputStartOffset = afterLabel + 1
		}
		yOffset += 3
	}

	yOffset = area.YMin() + 1
	for _, fld := range frm.Fields() {
		isFocused := frm.IsFocused(fld.Code)
		a.drawFieldValue(inputStartOffset, area.Width-2, yOffset, fld.Value, isFocused)
		if isFocused {
			termbox.SetCursor(inputStartOffset+len(fld.Value)+1, yOffset)
		}
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

func appendRune(s string, r rune) string {
	b := make([]byte, 3)
	l := utf8.EncodeRune(b, r)
	return s + string(b[0:l])
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
	view, ok := a.views[mode]
	if ok {
		a.activeView = view
	} else {
		a.err = fmt.Errorf("unknown view requested")
	}
	if focusable, ok := view.(Focusable); ok {
		if err := focusable.BeforeFocus(); err != nil {
			a.err = err
		}
	}
}
