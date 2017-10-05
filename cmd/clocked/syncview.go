package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
	"github.com/zerok/clocked/internal/database"
)

type syncView struct {
	app        *application
	summary    database.Summary
	date       time.Time
	syncStatus []string
	area       Area
}

func newSyncView(app *application) *syncView {
	return &syncView{
		app: app,
	}
}

func (v *syncView) SetSummary(date time.Time, summary database.Summary) {
	v.summary = summary
	v.date = date
	v.syncStatus = make([]string, len(summary.Bookings), len(summary.Bookings))
}

func (v *syncView) renderStatus(s string, length int) string {
	return fmt.Sprintf("%s%s", strings.Repeat(" ", length-len(s)), s)
}

func (v *syncView) Render(area Area) error {
	v.area = area
	v.renderListing()
	return nil
}

func (v *syncView) renderListing() {
	v.app.drawHeadline(v.area.XMin(), v.area.YMin(), "Sychronizing with JIRA")
	var maxStatusLength int
	for _, s := range v.syncStatus {
		if len(s) > maxStatusLength {
			maxStatusLength = len(s)
		}
	}
	for idx, booking := range v.summary.Bookings {
		v.app.drawText(v.area.XMin(), v.area.YMin()+idx+1, fmt.Sprintf("[%s] %s - %s: %s", v.renderStatus(v.syncStatus[idx], maxStatusLength), formatTime(booking.Start), formatTime(booking.Stop), booking.Code), termbox.ColorDefault, termbox.ColorDefault)
	}
}

func (v *syncView) HandleKeyEvent(evt termbox.Event) error {
	switch {
	case evt.Ch == 'q':
		v.app.switchMode(summaryMode)
	case evt.Ch == 's':
		termbox.Close()
		if err := v.app.jiraClient.RemoveDatedWorklogs(context.Background(), v.date); err != nil {
			fmt.Println(err)
		}
		for idx, b := range v.summary.Bookings {
			if err := v.app.jiraClient.AddWorklog(context.Background(), b.Code, *b.Start, b.Duration()); err != nil {
				v.app.err = err
				v.syncStatus[idx] = "error"
				break
			}
			v.syncStatus[idx] = "done"
		}
		termbox.Init()
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	}
	return nil
}
