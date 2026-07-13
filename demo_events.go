package main

import (
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
)

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
	{key: "design-sprint", title: "Design sprint", weekday: time.Monday, allDayDays: 3, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "monday-map-making", title: "Monday Map-Making", weekday: time.Monday, hour: 9, duration: time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "roadmap-workshop", title: "Roadmap workshop", weekday: time.Monday, hour: 10, minute: 30, duration: 90 * time.Minute, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "prototype-planning", title: "Prototype planning", weekday: time.Monday, hour: 14, duration: 2 * time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},

	{key: "espanol-espresso-tuesday", title: "Español & Espresso", weekday: time.Tuesday, hour: 7, minute: 30, duration: 45 * time.Minute, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},
	{key: "api-design-review", title: "API design review", weekday: time.Tuesday, hour: 9, minute: 30, duration: time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "mobile-sync", title: "Mobile sync", weekday: time.Tuesday, hour: 10, duration: 30 * time.Minute, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "taco-tuesday", title: "Taco Tuesday", weekday: time.Tuesday, hour: 18, minute: 30, duration: time.Hour, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},

	{key: "morning-run", title: "Morning run", weekday: time.Wednesday, hour: 7, minute: 30, duration: 45 * time.Minute, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},
	{key: "deep-work-onboarding", title: "Deep work: onboarding", weekday: time.Wednesday, hour: 9, minute: 30, duration: 3 * time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "lunch-priya", title: "Lunch with Priya", weekday: time.Wednesday, hour: 13, duration: time.Hour, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},
	{key: "write-docs", title: "Write the docs", weekday: time.Wednesday, hour: 14, minute: 30, duration: 2 * time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},

	{key: "team-offsite", title: "Team offsite", weekday: time.Thursday, allDayDays: 2, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "espanol-espresso-thursday", title: "Español & Espresso", weekday: time.Thursday, hour: 7, minute: 30, duration: 45 * time.Minute, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},
	{key: "customer-interview", title: "Customer interview", weekday: time.Thursday, hour: 9, minute: 30, duration: 45 * time.Minute, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "demo-rehearsal", title: "Demo rehearsal", weekday: time.Thursday, hour: 11, duration: time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "climbing", title: "Climbing", weekday: time.Thursday, hour: 18, duration: 90 * time.Minute, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},

	{key: "release-readiness", title: "Release readiness", weekday: time.Friday, hour: 9, duration: time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "bug-bash", title: "Bug bash", weekday: time.Friday, hour: 9, minute: 30, duration: 90 * time.Minute, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "retro", title: "Retro: keep / drop / try", weekday: time.Friday, hour: 14, duration: time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "golden-hour-walk", title: "Golden hour walk", weekday: time.Friday, hour: 17, minute: 30, duration: time.Hour, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},

	{key: "maya-birthday", title: "Maya's birthday", weekday: time.Saturday, allDayDays: 1, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},
	{key: "farmers-market", title: "Farmers market", weekday: time.Saturday, hour: 9, duration: 90 * time.Minute, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},
	{key: "brunch-maya", title: "Brunch with Maya", weekday: time.Saturday, hour: 11, duration: 90 * time.Minute, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},
	{key: "side-project", title: "Side project hour", weekday: time.Saturday, hour: 14, minute: 30, duration: 2 * time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "late-noodles", title: "Late-night noodles", weekday: time.Saturday, hour: 20, duration: time.Hour, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},

	{key: "slow-coffee", title: "Slow coffee", weekday: time.Sunday, hour: 9, minute: 30, duration: time.Hour, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},
	{key: "plan-week", title: "Plan the week", weekday: time.Sunday, hour: 16, duration: time.Hour, calendarID: demoWorkCalendarID, calendarName: demoWorkCalendarTitle},
	{key: "book-tea", title: "Book & tea", weekday: time.Sunday, hour: 19, minute: 30, duration: time.Hour, calendarID: demoSocialCalendarID, calendarName: demoSocialCalendarTitle},
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
