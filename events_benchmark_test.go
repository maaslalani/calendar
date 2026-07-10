package main

import (
	"fmt"
	"testing"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
)

var benchmarkTimedLines [][]string

func BenchmarkRenderTimedLines(b *testing.B) {
	day := time.Date(2026, time.July, 9, 0, 0, 0, 0, time.Local)
	colors := map[string]string{
		"calendar-0": "#60A5FA",
		"calendar-1": "#F97316",
		"calendar-2": "#A78BFA",
		"calendar-3": "#34D399",
	}
	events := make([]ical.Event, 0, 96)
	for i := range 96 {
		startSlot := (i * 7) % (slotsPerDay - 8)
		durationSlots := 1 + i%8
		events = append(events, ical.Event{
			Title:      fmt.Sprintf("Event %d with a title to render", i),
			CalendarID: fmt.Sprintf("calendar-%d", i%4),
			StartDate:  day.Add(time.Duration(startSlot) * slotDuration),
			EndDate:    day.Add(time.Duration(startSlot+durationSlots) * slotDuration),
			Recurring:  i%5 == 0,
		})
	}

	b.ReportAllocs()
	for b.Loop() {
		benchmarkTimedLines = renderTimedLines(day, events, colors, 0, slotsPerDay, 120)
	}
}
