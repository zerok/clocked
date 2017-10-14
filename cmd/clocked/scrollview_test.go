package main

import (
	"testing"

	"github.com/stretchr/testify/require"
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
	require.Equal(t, 4, sv.windowSize)

	sv.selectedIndex = 3
	sv.recalculateOffset()
	require.Equal(t, 0, sv.offset, "No offset should be required to show the last visible item")

	sv.selectedIndex = 4
	sv.recalculateOffset()
	require.Equal(t, 1, sv.offset, "1 as offset should be set for focusing the first not-visible item")
}

func TestScrollViewSelectedItem(t *testing.T) {
	sv := NewScrollableList(Area{})
	sv.UpdateItems([]ScrollableListItem{
		MockScrollViewItem("a"),
		MockScrollViewItem("b"),
		MockScrollViewItem("c"),
		MockScrollViewItem("d"),
		MockScrollViewItem("e"),
	})

	// Let's make sure that jumping around with next and previous updates the
	// selected item
	sv.Next()     // a
	sv.Previous() // e
	selected, ok := sv.SelectedItem()
	require.True(t, ok, "An item should have been selected")
	require.Equal(t, "e", selected.Label(), "e should have been the selected item")

	// If the number of available items gets smaller and the previously
	// selected item is beyond that number, no item should end up being
	// selected.
	sv.UpdateItems([]ScrollableListItem{
		MockScrollViewItem("a"),
		MockScrollViewItem("b"),
	})
	_, ok = sv.SelectedItem()
	require.False(t, ok, "No item should have been selected")
}
