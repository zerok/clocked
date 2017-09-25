package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockScrollViewItem string

func (s MockScrollViewItem) Label() string {
	return string(s)
}

func TestScrollviewUpdateOffset(t *testing.T) {
	sv := NewScrollableList(Area{
		Width:  5,
		Height: 5,
	})
	sv.windowSize = 3
	sv.selectedIndex = 3
	sv.recalculateOffset()
	assert.Equal(t, 1, sv.offset)
	sv.selectedIndex = 4
	sv.recalculateOffset()
	assert.Equal(t, 2, sv.offset)
}

func TestScrollViewSelectedItem(t *testing.T) {
	sv := NewScrollableList(Area{})
	sv.UpdateItems([]ScrollableListItem{
		MockScrollViewItem("a"),
		MockScrollViewItem("b"),
		MockScrollViewItem("c"),
	})

	// Let's make sure that jumping around with next and previous updates the
	// selected item
	sv.Next()     // a
	sv.Previous() // c
	selected, ok := sv.SelectedItem()
	assert.True(t, ok, "An item should have been selected")
	assert.Equal(t, "c", selected.Label(), "c should have been the selected item")

	// If the number of available items gets smaller and the previously
	// selected item is beyond that number, no item should end up being
	// selected.
	sv.UpdateItems([]ScrollableListItem{
		MockScrollViewItem("a"),
		MockScrollViewItem("b"),
	})
	_, ok = sv.SelectedItem()
	assert.False(t, ok, "No item should have been selected")
}
