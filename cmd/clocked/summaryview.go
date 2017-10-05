package main

import (
	"fmt"
	"time"

	"github.com/nsf/termbox-go"
	"github.com/zerok/clocked/internal/backup"
	"github.com/zerok/clocked/internal/database"
)

type summaryView struct {
	backup  *backup.Backup
	app     *application
	date    *time.Time
	summary database.Summary
	area    Area
}

func (v *summaryView) Render(area Area) error {
	v.area = area
	v.summary = v.app.db.GenerateDailySummary(*v.date)
	v.renderSummary()
	return nil
}

func (v *summaryView) BeforeFocus() error {
	now := time.Now()
	v.date = &now
	return nil
}

func (v *summaryView) renderSummary() {
	area := v.area
	v.app.drawHeadline(area.XMin(), area.YMin(), fmt.Sprintf("Summary for %s", v.date.Format("Mon, 2 Jan 2006")))
	for idx, b := range v.summary.Bookings {
		var color termbox.Attribute
		switch b.SubmissionStatus {
		case database.SubmissionStatusOK:
			color = termbox.ColorGreen
		case database.SubmissionStatusSkipped:
			color = termbox.ColorYellow
		default:
			color = termbox.ColorDefault
		}
		v.app.drawText(area.XMin(), area.YMin()+1+idx, fmt.Sprintf("%s - %s (%s)", formatTime(b.Start), formatTime(b.Stop), b.Code), color, termbox.ColorDefault)
	}

	idx := 0
	for key, dur := range v.summary.Totals {
		v.app.drawText(area.XMin()+area.Width/2, area.YMin()+1+idx, fmt.Sprintf("%s: %s", key, dur), termbox.ColorDefault, termbox.ColorDefault)
		idx++
	}
	idx++
	v.app.drawText(area.XMin()+area.Width/2, area.YMin()+1+idx, "Total: ", termbox.ColorDefault|termbox.AttrBold, termbox.ColorDefault)
	v.app.drawText(area.XMin()+area.Width/2+7, area.YMin()+1+idx, v.summary.Total.String(), termbox.ColorDefault, termbox.ColorDefault)
}

func (v *summaryView) HandleKeyEvent(evt termbox.Event) error {
	switch {
	case evt.Key == termbox.KeyEsc || evt.Ch == 'q':
		v.app.switchMode(selectionMode)
		v.date = nil
	case evt.Key == termbox.KeyCtrlJ:
		if v.app.jiraClient != nil {
			if view, ok := v.app.views[syncMode].(*syncView); ok {
				view.SetSummary(*v.date, v.summary)
			}
			v.app.switchMode(syncMode)
		} else {
			v.app.err = fmt.Errorf("JIRA not configured")
		}
	default:
		dateDelta := 0

		switch evt.Ch {
		case 'j':
			dateDelta = 1
		case 'k':
			dateDelta = -1
		}

		if dateDelta != 0 {
			nextDate := time.Now()
			if v.date == nil {
				nextDate = nextDate.AddDate(0, 0, dateDelta)
			} else {
				nextDate = v.date.AddDate(0, 0, dateDelta)
			}
			v.date = &nextDate
		}
	}
	return nil
}
