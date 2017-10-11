package main

import (
	"github.com/nsf/termbox-go"
	"github.com/zerok/clocked/internal/backup"
)

type snapshotView struct {
	app       *application
	snapshots []backup.Snapshot
	list      *ScrollableList
}

func newSnapshotView(app *application) *snapshotView {
	return &snapshotView{
		app:  app,
		list: NewScrollableList(Area{}),
	}
}

func (v *snapshotView) BeforeFocus() error {
	snapshots, err := v.app.backup.Snapshots()
	if err != nil {
		return err
	}
	v.snapshots = snapshots
	items := make([]ScrollableListItem, 0, len(snapshots))
	for _, i := range snapshots {
		items = append(items, i)
	}
	v.list.UpdateItems(items)
	return nil
}

func (v *snapshotView) Render(area Area) error {
	v.list.UpdateArea(area)
	v.list.Render()
	return nil
}

func (v *snapshotView) KeyMapping() []KeyMap {
	var result []KeyMap
	result = append(result, KeyMap{Label: "Back", Key: "q/ESC"})
	if _, ok := v.list.SelectedItem(); ok {
		result = append(result, KeyMap{Label: "Restore", Key: "ENTER"})
	}
	result = append(result, KeyMap{Label: "Next", Key: "j"}, KeyMap{Label: "Previous", Key: "k"})
	return result
}

func (v *snapshotView) HandleKeyEvent(evt termbox.Event) error {
	switch {
	case evt.Ch == 'q' || evt.Key == termbox.KeyEsc:
		return ErrCloseView
	case evt.Ch == 'j':
		v.list.Next()
	case evt.Ch == 'k':
		v.list.Previous()
	case evt.Key == termbox.KeyEnter:
		if item, ok := v.list.SelectedItem(); ok {
			if snapshot, ok := item.(backup.Snapshot); ok {
				if err := v.app.backup.Restore(snapshot.ID); err != nil {
					v.app.err = err
				} else {
					if err := v.app.db.LoadState(); err != nil {
						v.app.err = err
					} else {
						return ErrCloseView
					}
				}
			}
		}
	}
	return nil
}
