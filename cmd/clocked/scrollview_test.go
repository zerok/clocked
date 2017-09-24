package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
