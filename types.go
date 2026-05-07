package main

import (
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
)

type calendarData struct {
	sections       []daySection
	calendars      []ical.Calendar
	calendarColors map[string]string
	legend         []calendarLegendItem
	currentTime    time.Time
}

type daySection struct {
	label  string
	date   time.Time
	events []ical.Event
}

type calendarLegendItem struct {
	key   string
	label string
	color string
}

type timedEventBlock struct {
	event     ical.Event
	startSlot int
	endSlot   int
	accent    string
	layer     int
	cluster   int
}

type eventAccentSegment struct {
	accent string
	color  string
}

type timedEventLayout struct {
	columnWidths []int
	separator    string
}

type calendarLayout struct {
	sectionWidths []int
	separator     string
}

type calendarSlotWindow struct {
	start int
	end   int
}
