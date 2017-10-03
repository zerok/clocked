package main

import (
	"context"
	"fmt"
	"strings"
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
	var now time.Time
	if v.date != nil {
		now = *v.date
	} else {
		now = time.Now()
	}
	v.date = &now
	v.area = area
	v.summary = v.app.db.GenerateDailySummary(now)
	v.renderSummary()
	return nil
}

func (v *summaryView) renderSummary() {
	area := v.area
	v.app.drawText(area.XMin(), area.YMin(), fmt.Sprintf("Summary for %s", v.date.Format("Mon, 2 Jan 2006")), termbox.ColorBlue|termbox.AttrBold, termbox.ColorDefault)
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
	switch evt.Key {
	case termbox.KeyEsc:
		v.app.switchMode(selectionMode)
		v.date = nil
	case termbox.KeyCtrlJ:
		if v.app.jiraClient != nil {
			for idx, b := range v.summary.Bookings {
				if b.Stop != nil {
					if strings.HasPrefix(b.Code, "NTI-") {
						v.app.log.Infof("Skipping %s", b.Code)
						(&v.summary.Bookings[idx]).SubmissionStatus = database.SubmissionStatusSkipped
						continue
					}
					if err := v.app.jiraClient.AddWorklog(context.Background(), b.Code, *b.Start, b.Duration()); err != nil {
						v.app.err = err
						return err
					}
					(&v.summary.Bookings[idx]).SubmissionStatus = database.SubmissionStatusOK
					v.renderSummary()
					termbox.Flush()
				}
			}
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
