package main

import (
	"os"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
)

const (
	calendarDemoEnv     = "CALENDAR_DEMO"
	calendarWork        = "demo-work"
	calendarWorkTitle   = "Work"
	calendarSocial      = "demo-social"
	calendarSocialTitle = "Social"
)

func calendarDemoEnabled() bool {
	return os.Getenv(calendarDemoEnv) != ""
}

func loadDemoCalendar(viewDay, now time.Time) calendarData {
	calendars := []ical.Calendar{
		{
			ID:     calendarWork,
			Title:  calendarWorkTitle,
			Type:   ical.CalendarTypeLocal,
			Source: "Demo",
			Color:  "#7C3AED",
		},
		{
			ID:     calendarSocial,
			Title:  calendarSocialTitle,
			Type:   ical.CalendarTypeLocal,
			Source: "Demo",
			Color:  "#F97316",
		},
	}
	return newCalendarData(viewDay, now, calendars, demoEvents(viewDay))
}

type demoEventSpec struct {
	key          string
	title        string
	weekday      time.Weekday
	hour         int
	minute       int
	duration     time.Duration
	allDayDays   int
	calendarID   string
	calendarName string
}

var demoEventSpecs = []demoEventSpec{
	{key: "roadmap-workshop", title: "Roadmap workshop", weekday: time.Monday, hour: 10, minute: 30, duration: 90 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "prototype", title: "Prototype", weekday: time.Monday, hour: 14, duration: 2 * time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},

	{key: "launch", title: "Launch", weekday: time.Tuesday, allDayDays: 1, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "espanol", title: "Learn Español", weekday: time.Tuesday, hour: 7, minute: 30, duration: 45 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "api-design-review", title: "API design review", weekday: time.Tuesday, hour: 9, minute: 30, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "mobile-sync", title: "Mobile sync", weekday: time.Tuesday, hour: 10, duration: 30 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "chai-night", title: "Chai Night", weekday: time.Tuesday, hour: 18, minute: 30, duration: 3 * time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},

	{key: "morning-run", title: "Morning run", weekday: time.Wednesday, hour: 7, minute: 30, duration: 45 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "onboarding", title: "Onboarding", weekday: time.Wednesday, hour: 9, minute: 30, duration: 3 * time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "lunch-priya", title: "Lunch with Priya", weekday: time.Wednesday, hour: 13, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "write-docs", title: "Write the docs", weekday: time.Wednesday, hour: 14, minute: 30, duration: 2 * time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},

	{key: "team-offsite", title: "Team offsite", weekday: time.Thursday, allDayDays: 2, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "espanol", title: "Learn Español", weekday: time.Thursday, hour: 7, minute: 30, duration: 45 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "customer-interview", title: "Customer interview", weekday: time.Thursday, hour: 9, minute: 30, duration: 45 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "demo-rehearsal", title: "Demo rehearsal", weekday: time.Thursday, hour: 11, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "go-kart", title: "Go-karting", weekday: time.Thursday, hour: 18, duration: 90 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},

	{key: "release-readiness", title: "Release", weekday: time.Friday, hour: 9, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "bug-bash", title: "Bug bash", weekday: time.Friday, hour: 9, minute: 30, duration: 90 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "retro", title: "Retro: terminal calendar", weekday: time.Friday, hour: 14, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "walk", title: "Walk", weekday: time.Friday, hour: 17, minute: 30, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},

	{key: "maya-birthday", title: "Maya's birthday", weekday: time.Saturday, allDayDays: 1, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "farmers-market", title: "Farmer's market", weekday: time.Saturday, hour: 9, duration: 90 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "brunch-maya", title: "Brunch with Maya", weekday: time.Saturday, hour: 11, duration: 90 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "side-project", title: "Work on Terminal Calendar", weekday: time.Saturday, hour: 14, minute: 30, duration: 2 * time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "dinner-ramen", title: "Dinner: Ramen", weekday: time.Saturday, hour: 20, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},

	{key: "coffee", title: "Coffee", weekday: time.Sunday, hour: 9, minute: 30, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "plan", title: "Plan", weekday: time.Sunday, hour: 16, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "chai-night", title: "Chai Night", weekday: time.Sunday, hour: 19, minute: 30, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},
}

func demoEvents(viewDay time.Time) []ical.Event {
	start := beginningOfDay(viewDay).AddDate(0, 0, -7)
	end := beginningOfDay(viewDay).AddDate(0, 0, 8)
	var events []ical.Event

	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		for _, spec := range demoEventSpecs {
			if spec.weekday == day.Weekday() {
				events = append(events, newDemoEvent(spec, day))
			}
		}
	}
	return events
}

func newDemoEvent(spec demoEventSpec, day time.Time) ical.Event {
	start := beginningOfDay(day)
	end := start.AddDate(0, 0, spec.allDayDays)
	allDay := spec.allDayDays > 0
	if !allDay {
		start = time.Date(day.Year(), day.Month(), day.Day(), spec.hour, spec.minute, 0, 0, day.Location())
		end = start.Add(spec.duration)
	}

	return ical.Event{
		ID:         spec.key + ":" + day.Format("2006-01-02"),
		Title:      spec.title,
		StartDate:  start,
		EndDate:    end,
		AllDay:     allDay,
		Calendar:   spec.calendarName,
		CalendarID: spec.calendarID,
	}
}
