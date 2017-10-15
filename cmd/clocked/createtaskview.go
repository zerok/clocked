package main

import (
	termbox "github.com/nsf/termbox-go"
	"github.com/zerok/clocked/internal/form"
)

func newCreateTaskForm() *form.Form {
	return form.NewForm([]form.Field{
		{
			Code:       "code",
			Label:      "Code:",
			IsRequired: true,
		}, {
			Code:       "title",
			Label:      "Title:",
			IsRequired: true,
		}, {
			Code:       "tags",
			Label:      "Tags:",
			IsRequired: false,
		},
	})
}

type createTaskView struct {
	app  *application
	form *form.Form
}

func newCreateTaskView(app *application) *createTaskView {
	return &createTaskView{
		app: app,
	}
}

func (v *createTaskView) KeyMapping() []KeyMap {
	return []KeyMap{
		{Label: "Quit", Key: "^c"},
		{Label: "Switch field", Key: "TAB"},
		{Label: "Create task", Key: "ENTER"},
		{Label: "Cancel", Key: "ESC"},
	}
}

func (v *createTaskView) BeforeFocus() error {
	v.form = newCreateTaskForm()
	return nil
}

func (v *createTaskView) Render(area Area) error {
	v.app.redrawForm(area, v.form)
	return nil
}

func (v *createTaskView) HandleKeyEvent(evt termbox.Event) error {
	a := v.app
	switch evt.Key {
	case termbox.KeyEsc:
		a.switchMode(selectionMode)
	case termbox.KeyTab:
		v.form.Next()
	case termbox.KeyEnter:
		if v.form.Validate() {
			t := convertToTask(v.form)
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
			if view, ok := a.activeView.(*tasklistView); ok {
				view.updateTaskList()
				view.list.SelectMatchingItem(selectByCode(t.Code))
			}
		}
	default:
		a.handleFieldInput(v.form, evt)
	}
	return nil
}
