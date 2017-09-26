package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerok/clocked"
	"github.com/zerok/clocked/internal/database"
)

func TestFiltering(t *testing.T) {
	app := newApplication()
	app.db = database.NewInMemory()
	app.db.AddTask(clocked.Task{Code: "a"})
	app.db.AddTask(clocked.Task{Code: "b"})
	app.taskListView = NewScrollableList(Area{})
	app.updateTaskList()
	// Once the filter is changed, the first item in the list should be
	// selected.
	app.selectNextRow()
	app.selectNextRow()
	selected, ok := app.taskListView.SelectedItem()
	require.True(t, ok, "An item should have been selected")
	require.Equal(t, "b ", selected.Label(), "b should have been the selected item")
	app.pushFilter('b')
	selected, ok = app.taskListView.SelectedItem()
	require.True(t, ok, "An item should have been selected after changing the filter")
	require.Equal(t, "b ", selected.Label(), "The single matching item should have been selected")
	app.pushFilter('c')
	selected, ok = app.taskListView.SelectedItem()
	require.False(t, ok, "Since there is no matching item, nothing should be selected")
}

func TestJumpToActiveTask(t *testing.T) {
	app := newApplication()
	app.db = database.NewInMemory()
	app.db.AddTask(clocked.Task{Code: "a"})
	app.db.AddTask(clocked.Task{Code: "b"})
	app.taskListView = NewScrollableList(Area{})
	app.updateTaskList()

	require.NoError(t, app.db.ClockInto("b"), "Clocking into a should have worked")
	app.jumpToActiveTask()
	selected, ok := app.taskListView.SelectedItem()
	require.True(t, ok, "A task should be selected")
	require.Equal(t, "b ", selected.Label(), "b should be the selected item")
}
