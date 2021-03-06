package main

import (
	"fmt"

	"github.com/nsf/termbox-go"
)

type ItemMatcherFunc func(ScrollableListItem) bool

type ScrollableListItem interface {
	Label() string
}

type ScrollableList struct {
	selectedIndex int
	items         []ScrollableListItem
	windowSize    int
	offset        int
	area          Area
}

func NewScrollableList(area Area) *ScrollableList {
	sv := &ScrollableList{
		items:         make([]ScrollableListItem, 0, 0),
		selectedIndex: -1,
	}
	sv.UpdateArea(area)
	return sv
}

func (s *ScrollableList) SelectItemByLabel(l string) (int, bool) {
	for idx, i := range s.items {
		if i.Label() == l {
			_, ok := s.SelectItemByIndex(idx)
			return idx, ok
		}
	}
	return -1, false
}

func (s *ScrollableList) SelectItemByIndex(i int) (ScrollableListItem, bool) {
	if i >= len(s.items) {
		return nil, false
	}
	s.selectedIndex = i
	s.recalculateOffset()
	return s.items[i], true
}

func (s *ScrollableList) SelectMatchingItem(matcher ItemMatcherFunc) (int, bool) {
	for idx, item := range s.items {
		if matcher(item) {
			_, selected := s.SelectItemByIndex(idx)
			return idx, selected
		}
	}
	return -1, false
}

func (s *ScrollableList) drawWindow() {
	line := 0
	for i := s.offset; i < s.offset+s.windowSize; i++ {
		if i >= len(s.items) {
			break
		}
		s.renderItem(i, line)
		line++
	}
}

func (s *ScrollableList) renderItem(idx int, line int) {
	item := s.items[idx]
	if idx == s.selectedIndex {
		termbox.SetCell(s.area.XMin(), s.area.YMin()+line, '>', termbox.ColorDefault|termbox.AttrBold, termbox.ColorDefault)
	}
	for chIdx, c := range item.Label() {
		termbox.SetCell(s.area.XMin()+3+chIdx, s.area.YMin()+line, c, termbox.ColorDefault, termbox.ColorDefault)
	}
}

func (s *ScrollableList) Next() {
	cur := s.selectedIndex
	cur++
	if cur > len(s.items)-1 {
		cur = 0
	}
	s.selectedIndex = cur
	s.recalculateOffset()
}

func (s *ScrollableList) Previous() {
	cur := s.selectedIndex
	cur--
	if cur < 0 {
		cur = len(s.items) - 1
	}
	s.selectedIndex = cur
	s.recalculateOffset()
}

func (s *ScrollableList) SelectedItem() (ScrollableListItem, bool) {
	if s.selectedIndex < 0 {
		return nil, false
	}
	if s.selectedIndex >= len(s.items) {
		return nil, false
	}
	return s.items[s.selectedIndex], true
}

func (s *ScrollableList) recalculateOffset() {
	if s.selectedIndex < s.offset {
		s.offset = s.selectedIndex
		return
	}
	if s.selectedIndex >= s.offset+s.windowSize {
		s.offset = s.selectedIndex - s.windowSize + 1
	}
}

func (s *ScrollableList) Render() {
	s.drawWindow()
	s.renderPager()
}

func (s *ScrollableList) UpdateArea(a Area) {
	s.area = a
	s.windowSize = a.Height - 1 // -1 for the pager
}

func (s *ScrollableList) UpdateItems(i []ScrollableListItem) {
	if s.selectedIndex >= len(i) {
		s.selectedIndex = -1
	}
	s.items = i
}

func (s *ScrollableList) renderPager() {
	text := fmt.Sprintf("[%d/%d]", s.selectedIndex+1, len(s.items))
	xOffset := s.area.XMax() - len(text)
	yOffset := s.area.YMax()
	for idx, c := range text {
		termbox.SetCell(xOffset+idx, yOffset, c, termbox.ColorDefault, termbox.ColorDefault)
	}
}
