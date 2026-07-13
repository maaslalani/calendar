package main

import (
	"strings"
	"testing"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
)

func TestLoadCalendarDemoUsesCodeDefinedCalendars(t *testing.T) {
	t.Setenv(calendarDemoEnv, "1")

	now := time.Date(2026, time.July, 13, 10, 16, 0, 0, time.Local)
	data, err := loadCalendar(beginningOfDay(now), now)
	if err != nil {
		t.Fatalf("loadCalendar() error = %v", err)
	}

	if len(data.legend) != len(demoCalendarDefinitions) {
		t.Fatalf("len(data.legend) = %d, want %d", len(data.legend), len(demoCalendarDefinitions))
	}
	for i, section := range data.sections {
		if len(section.events) == 0 {
			t.Errorf("data.sections[%d] has no events", i)
		}
	}

	rendered := renderCalendarLayout(data, 110)
	for _, text := range []string{
		"Slow coffee",
		"Prototype planning",
		"Taco Tuesday",
		demoSocialCalendarTitle,
		demoWorkCalendarTitle,
	} {
		if !strings.Contains(rendered, text) {
			t.Errorf("rendered calendar does not contain %q", text)
		}
	}
}

func TestDemoCalendarDefinitions(t *testing.T) {
	want := []ical.Calendar{
		{ID: demoWorkCalendarID, Title: demoWorkCalendarTitle, Color: "#7C3AED"},
		{ID: demoSocialCalendarID, Title: demoSocialCalendarTitle, Color: "#F97316"},
	}

	for i, expected := range want {
		calendar := demoCalendarDefinitions[i]
		if calendar.ID != expected.ID || calendar.Title != expected.Title || calendar.Color != expected.Color {
			t.Errorf("demoCalendarDefinitions[%d] = %+v, want ID %q, title %q, color %q", i, calendar, expected.ID, expected.Title, expected.Color)
		}
	}
}

func TestDemoEventsFollowViewDay(t *testing.T) {
	viewDay := time.Date(2032, time.January, 20, 0, 0, 0, 0, time.Local)
	data := loadDemoCalendar(viewDay, viewDay.Add(10*time.Hour))

	for i, section := range data.sections {
		if len(section.events) == 0 {
			t.Errorf("data.sections[%d] has no events for %s", i, section.date.Format("2006-01-02"))
		}
	}
}
