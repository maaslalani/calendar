package main

import (
	"os"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
)

const (
	calendarDemoEnv         = "CALENDAR_DEMO"
	demoWorkCalendarID      = "demo-work"
	demoWorkCalendarTitle   = "Work"
	demoSocialCalendarID    = "demo-social"
	demoSocialCalendarTitle = "Social"
)

var demoCalendarDefinitions = [...]ical.Calendar{
	{
		ID:     demoWorkCalendarID,
		Title:  demoWorkCalendarTitle,
		Type:   ical.CalendarTypeLocal,
		Source: "Demo",
		Color:  "#7C3AED",
	},
	{
		ID:     demoSocialCalendarID,
		Title:  demoSocialCalendarTitle,
		Type:   ical.CalendarTypeLocal,
		Source: "Demo",
		Color:  "#F97316",
	},
}

func calendarDemoEnabled() bool {
	return os.Getenv(calendarDemoEnv) != ""
}

func loadDemoCalendar(viewDay, now time.Time) calendarData {
	calendars := append([]ical.Calendar(nil), demoCalendarDefinitions[:]...)
	return newCalendarData(viewDay, now, calendars, demoEvents(viewDay))
}
