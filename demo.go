package main

import (
	"os"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
)

const (
	calendarDemoEnv        = "CALENDAR_DEMO"
	calendarWork           = "demo-work"
	calendarWorkTitle      = "Work"
	calendarSocial         = "demo-social"
	calendarSocialTitle    = "Social"
	calendarReminders      = "demo-reminders"
	calendarRemindersTitle = "Reminders"
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
			Color:  "#C4A7E7",
		},
		{
			ID:     calendarSocial,
			Title:  calendarSocialTitle,
			Type:   ical.CalendarTypeLocal,
			Source: "Demo",
			Color:  "#F6C177",
		},
		{
			ID:     calendarReminders,
			Title:  calendarRemindersTitle,
			Type:   ical.CalendarTypeLocal,
			Source: "Demo",
			Color:  "#9CCFD8",
		},
	}
	return newCalendarData(viewDay, now, calendars, demoEvents(now))
}

type demoEventSpec struct {
	key          string
	title        string
	weekday      time.Weekday
	hour         int
	minute       int
	duration     time.Duration
	allDayDays   int
	recurring    bool
	calendarID   string
	calendarName string
}

var demoEventSpecs = []demoEventSpec{
	{key: "coffee-will", title: "Coffee with Will", weekday: time.Monday, hour: 8, duration: 45 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "calendar-themes", title: "Design calendar themes", weekday: time.Monday, hour: 9, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "roadmap-workshop", title: "Plan calendar roadmap", weekday: time.Monday, hour: 10, minute: 30, duration: 90 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "prototype", title: "Prototype event creation", weekday: time.Monday, hour: 14, duration: 2 * time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "call-mom", title: "Call Mom", weekday: time.Monday, hour: 17, duration: 30 * time.Minute, recurring: true, calendarID: calendarReminders, calendarName: calendarRemindersTitle},

	{key: "launch", title: "Launch", weekday: time.Tuesday, allDayDays: 1, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "espanol", title: "Coffee", weekday: time.Tuesday, hour: 7, minute: 30, duration: 45 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "api-design-review", title: "Review EventKit integration", weekday: time.Tuesday, hour: 9, minute: 30, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "mobile-sync", title: "Implement calendar sync", weekday: time.Tuesday, hour: 10, duration: 30 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "recurring-events", title: "Handle recurring events", weekday: time.Tuesday, hour: 11, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "call-mom", title: "Call Mom", weekday: time.Tuesday, hour: 17, duration: 30 * time.Minute, recurring: true, calendarID: calendarReminders, calendarName: calendarRemindersTitle},
	{key: "chai-night", title: "Chai Night", weekday: time.Tuesday, hour: 18, minute: 30, duration: 3 * time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},

	{key: "morning-run", title: "Morning run", weekday: time.Wednesday, hour: 7, minute: 30, duration: 45 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "onboarding", title: "Build event creation", weekday: time.Wednesday, hour: 9, minute: 30, duration: 3 * time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "lunch", title: "Lunch with Will", weekday: time.Wednesday, hour: 13, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "write-docs", title: "Document calendar shortcuts", weekday: time.Wednesday, hour: 14, minute: 30, duration: 2 * time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "call-mom", title: "Call Mom", weekday: time.Wednesday, hour: 17, duration: 30 * time.Minute, recurring: true, calendarID: calendarReminders, calendarName: calendarRemindersTitle},
	{key: "dinner", title: "Dinner", weekday: time.Wednesday, hour: 18, minute: 30, duration: 90 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},

	{key: "team-offsite", title: "Build multi-day events", weekday: time.Thursday, allDayDays: 2, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "espanol", title: "Coffee", weekday: time.Thursday, hour: 7, minute: 30, duration: 45 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "customer-interview", title: "Test keyboard navigation", weekday: time.Thursday, hour: 9, minute: 30, duration: 45 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "demo", title: "Demo", weekday: time.Thursday, hour: 11, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "event-editor", title: "Polish event editor", weekday: time.Thursday, hour: 13, duration: 90 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "call-mom", title: "Call Mom", weekday: time.Thursday, hour: 17, duration: 30 * time.Minute, recurring: true, calendarID: calendarReminders, calendarName: calendarRemindersTitle},
	{key: "go-kart", title: "Go-karting", weekday: time.Thursday, hour: 18, duration: 90 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},

	{key: "release-readiness", title: "Prepare calendar release", weekday: time.Friday, hour: 9, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "bug-bash", title: "Test overlapping events", weekday: time.Friday, hour: 9, minute: 30, duration: 90 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "timezone-changes", title: "Test timezone changes", weekday: time.Friday, hour: 11, minute: 30, duration: 45 * time.Minute, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "retro", title: "Retro: terminal calendar", weekday: time.Friday, hour: 14, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "call-mom", title: "Call Mom", weekday: time.Friday, hour: 17, duration: 30 * time.Minute, recurring: true, calendarID: calendarReminders, calendarName: calendarRemindersTitle},
	{key: "walk", title: "Walk", weekday: time.Friday, hour: 17, minute: 30, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "movie-night", title: "Movie night", weekday: time.Friday, hour: 20, duration: 2 * time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},

	{key: "maya-birthday", title: "Maya's birthday", weekday: time.Saturday, allDayDays: 1, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "farmers-market", title: "Farmer's market", weekday: time.Saturday, hour: 9, duration: 90 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "brunch-maya", title: "Brunch with Maya", weekday: time.Saturday, hour: 11, duration: 90 * time.Minute, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "side-project", title: "Work on terminal calendar", weekday: time.Saturday, hour: 14, minute: 30, duration: 2 * time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "call-mom", title: "Call Mom", weekday: time.Saturday, hour: 17, duration: 30 * time.Minute, recurring: true, calendarID: calendarReminders, calendarName: calendarRemindersTitle},
	{key: "dinner-ramen", title: "Dinner: Ramen", weekday: time.Saturday, hour: 20, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},

	{key: "coffee", title: "Coffee", weekday: time.Sunday, hour: 9, minute: 30, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "long-run", title: "Long run", weekday: time.Sunday, hour: 11, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},
	{key: "plan", title: "Plan next calendar features", weekday: time.Sunday, hour: 16, duration: time.Hour, calendarID: calendarWork, calendarName: calendarWorkTitle},
	{key: "call-mom", title: "Call Mom", weekday: time.Sunday, hour: 17, duration: 30 * time.Minute, recurring: true, calendarID: calendarReminders, calendarName: calendarRemindersTitle},
	{key: "chai-night", title: "Chai Night", weekday: time.Sunday, hour: 19, minute: 30, duration: time.Hour, calendarID: calendarSocial, calendarName: calendarSocialTitle},
}

func demoEvents(now time.Time) []ical.Event {
	firstDay := beginningOfDay(now).AddDate(0, 0, -1)
	events := make([]ical.Event, 0, len(demoEventSpecs))
	for _, spec := range demoEventSpecs {
		offset := (int(spec.weekday) - int(firstDay.Weekday()) + 7) % 7
		events = append(events, newDemoEvent(spec, firstDay.AddDate(0, 0, offset)))
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
		Recurring:  spec.recurring,
		Calendar:   spec.calendarName,
		CalendarID: spec.calendarID,
	}
}
