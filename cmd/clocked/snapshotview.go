package main

import (
	"github.com/nsf/termbox-go"
	"github.com/zerok/clocked/internal/backup"
)

type snapshotView struct {
	backup *backup.Backup
}

func (v *snapshotView) Render() error {
	return nil
}

func (v *snapshotView) HandleKeyEvent(evt termbox.Event) error {
	return nil
}
