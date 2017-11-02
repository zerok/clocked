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

func (v *syncView) KeyMapping() []KeyMap {
	return []KeyMap{
		{Label: "Quit", Key: "^c"},
		{Label: "Start", Key: "s"},
		{Label: "Cancel", Key: "q/ESC"},
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
	offlineBookings, onlineBookings := v.filterOfflineBookings(v.summary.Bookings)

	v.app.drawHeadline(v.area.XMin(), v.area.YMin(), fmt.Sprintf("Sychronizing tasks for %s with JIRA", v.date.Format("Mon, 2 Jan 2006")))
	var maxStatusLength int
	for _, s := range v.syncStatus {
		if len(s) > maxStatusLength {
			maxStatusLength = len(s)
		}
	}
	yOffset := 0
	for idx, booking := range onlineBookings {
		v.app.drawText(v.area.XMin(), v.area.YMin()+idx+1, fmt.Sprintf("[%s] %s - %s: %s", v.renderStatus(v.syncStatus[idx], maxStatusLength), formatTime(booking.Start), formatTime(booking.Stop), booking.Code), termbox.ColorDefault, termbox.ColorDefault)
		yOffset = v.area.YMin() + idx + 1
	}

	if len(offlineBookings) > 0 {
		yOffset += 2
		v.app.drawText(v.area.XMin(), yOffset, "Offline bookings:", termbox.ColorDefault|termbox.AttrBold, termbox.ColorDefault)
		yOffset++
		for _, b := range offlineBookings {
			v.app.drawText(v.area.XMin(), yOffset, fmt.Sprintf("%s - %s: %s", formatTime(b.Start), formatTime(b.Stop), b.Code), termbox.ColorDefault, termbox.ColorDefault)
		}
	}
}

func (v *syncView) filterOfflineBookings(all []database.TaskBooking) ([]database.TaskBooking, []database.TaskBooking) {
	offlineBookings := make([]database.TaskBooking, 0, 5)
	onlineBookings := make([]database.TaskBooking, 0, len(all))
	for _, b := range all {
		if t, found := v.app.db.TaskByCode(b.Code); found {
			if t.HasTag("offline") {
				offlineBookings = append(offlineBookings, b)
			} else {
				onlineBookings = append(onlineBookings, b)
			}
		}
	}
	return offlineBookings, onlineBookings
}

func (v *syncView) HandleKeyEvent(evt termbox.Event) error {
	switch {
	case evt.Ch == 'q' || evt.Key == termbox.KeyEsc:
		v.app.switchMode(summaryMode)
	case evt.Ch == 's':
		termbox.Close()
		if err := v.app.jiraClient.RemoveDatedWorklogs(context.Background(), v.date); err != nil {
			v.app.err = err
			return err
		}
		_, onlineBookings := v.filterOfflineBookings(v.summary.Bookings)
		for idx, b := range onlineBookings {
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
