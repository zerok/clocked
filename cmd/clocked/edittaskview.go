package main

import (
	"strings"

	termbox "github.com/nsf/termbox-go"
	"github.com/zerok/clocked"
	"github.com/zerok/clocked/internal/form"
)

type editTaskView struct {
	app  *application
	form *form.Form
	task clocked.Task
}

func newEditTaskView(app *application) *editTaskView {
	return &editTaskView{
		app: app,
		form: form.NewForm([]form.Field{
			{
				Code:       "code",
				Label:      "Code",
				IsRequired: true,
			},
			{
				Code:       "title",
				Label:      "Title",
				IsRequired: false,
			},
			{
				Code:       "tags",
				Label:      "Tags",
				IsRequired: false,
			},
		}),
	}
}

func (v *editTaskView) SetTask(task clocked.Task) {
	v.task = task
	v.form.SetValue("code", task.Code)
	v.form.SetValue("title", task.Title)
	v.form.SetValue("tags", strings.Join(task.Tags, " "))
}

func (v *editTaskView) Render(area Area) error {
	v.app.redrawForm(area, v.form)
	return nil
}

func (v *editTaskView) HandleKeyEvent(evt termbox.Event) error {
	switch {
	case evt.Key == termbox.KeyEsc:
		return ErrCloseView
	case evt.Key == termbox.KeyTab:
		v.form.Next()
	case evt.Key == termbox.KeyEnter:
		t := convertToTask(v.form)
		if err := v.app.db.UpdateTask(v.task.Code, t); err != nil {
			v.app.err = err
		} else {
			return ErrCloseView
		}

	default:
		v.app.handleFieldInput(v.form, evt)
	}
	return nil
}

func (v *editTaskView) KeyMapping() []KeyMap {
	return []KeyMap{
		{Label: "Quit", Key: "^c"},
		{Label: "Focus next field", Key: "TAB"},
		{Label: "Save changes", Key: "ENTER"},
		{Label: "Cancel", Key: "ESC"},
	}
}
