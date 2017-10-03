package main

import (
	"fmt"

	"github.com/nsf/termbox-go"
)

type View interface {
	HandleKeyEvent(termbox.Event) error
	Render(Area) error
}

// Focusable is a view that offers a special method being called before
// it is being rendered. This way you can initialize things like scroll
// views or resize then.
type Focusable interface {
	BeforeFocus() error
}

// ErrCloseView can be returned by either Render or HandleKeyEvent in order
// to tell the dispatcher that the previous view should be rendered instead
// again.
var ErrCloseView = fmt.Errorf("close view")
