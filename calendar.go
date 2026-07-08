package main

import (
	"hash/fnv"
	"sort"
	"strings"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
)

func calendarTitleCounts(calendars []ical.Calendar) map[string]int {
	counts := make(map[string]int, len(calendars))
	for _, calendar := range calendars {
		counts[calendar.Title]++
	}
	return counts
}

func sortCalendars(calendars []ical.Calendar) {
	titleCounts := calendarTitleCounts(calendars)
	sort.SliceStable(calendars, func(i, j int) bool {
		left := calendarDisplayLabel(calendars[i], titleCounts)
		right := calendarDisplayLabel(calendars[j], titleCounts)
		if left != right {
			return left < right
		}
		return calendars[i].ID < calendars[j].ID
	})
}
func loadCalendar(viewDay, now time.Time) (calendarData, error) {
	if now.IsZero() {
		now = time.Now()
	}
	if viewDay.IsZero() {
		viewDay = beginningOfDay(now)
	} else {
		viewDay = beginningOfDay(viewDay)
	}

	client, err := ical.New()
	if err != nil {
		return calendarData{}, err
	}

	calendars, err := client.Calendars()
	if err != nil {
		return calendarData{}, err
	}
	sortCalendars(calendars)

	windowStart := viewDay.AddDate(0, 0, -1)
	windowEnd := viewDay.AddDate(0, 0, 2)

	events, err := client.Events(windowStart, windowEnd)
	if err != nil {
		return calendarData{}, err
	}

	calendarColors, legend := buildCalendarDecorations(calendars, events)

	return calendarData{
		sections:       buildDaySections(viewDay, events),
		calendarColors: calendarColors,
		legend:         legend,
		currentTime:    now,
	}, nil
}
func buildDaySections(today time.Time, events []ical.Event) []daySection {
	sections := []daySection{
		{label: "Yesterday", date: beginningOfDay(today).AddDate(0, 0, -1)},
		{label: "Today", date: beginningOfDay(today)},
		{label: "Tomorrow", date: beginningOfDay(today).AddDate(0, 0, 1)},
	}

	for i := range sections {
		dayStart := sections[i].date
		dayEnd := dayStart.AddDate(0, 0, 1)

		for _, event := range events {
			// Include events on every day they overlap so overnight and all-day
			// entries appear in each relevant section.
			if eventOverlapsDay(event, dayStart, dayEnd) {
				sections[i].events = append(sections[i].events, event)
			}
		}

		sort.SliceStable(sections[i].events, func(a, b int) bool {
			left := sections[i].events[a]
			right := sections[i].events[b]

			if !left.StartDate.Equal(right.StartDate) {
				return left.StartDate.Before(right.StartDate)
			}

			if !left.EndDate.Equal(right.EndDate) {
				return left.EndDate.Before(right.EndDate)
			}

			return left.Title < right.Title
		})
	}

	return sections
}

func eventOverlapsDay(event ical.Event, dayStart, dayEnd time.Time) bool {
	return event.StartDate.Before(dayEnd) && event.EndDate.After(dayStart)
}
func buildCalendarDecorations(calendars []ical.Calendar, events []ical.Event) (map[string]string, []calendarLegendItem) {
	calendarByID := make(map[string]ical.Calendar, len(calendars))
	calendarsByTitle := make(map[string][]ical.Calendar, len(calendars))
	titleCounts := make(map[string]int, len(calendars))

	for _, calendar := range calendars {
		calendarByID[calendar.ID] = calendar
		calendarsByTitle[calendar.Title] = append(calendarsByTitle[calendar.Title], calendar)
		titleCounts[calendar.Title]++
	}

	calendarColors := make(map[string]string, len(events)*2)
	legendByKey := make(map[string]calendarLegendItem, len(calendars)+len(events))

	// Seed the legend with every calendar so they always appear, even when
	// they have no events in the current view.
	for _, calendar := range calendars {
		key := calendarKey(calendar)
		legendByKey[key] = calendarLegendItem{
			key:   key,
			label: calendarDisplayLabel(calendar, titleCounts),
			color: normalizeCalendarColor(calendar.Color, key),
		}
	}

	for _, event := range events {
		key := legendKeyForEvent(event)
		label := fallbackCalendarLabel(event)
		color := fallbackCalendarColor(key)

		if calendar, ok := resolveEventCalendar(event, calendarByID, calendarsByTitle); ok {
			key = calendarKey(calendar)
			label = calendarDisplayLabel(calendar, titleCounts)
			color = normalizeCalendarColor(calendar.Color, key)
		}

		if event.CalendarID != "" {
			calendarColors[event.CalendarID] = color
		}
		if event.Calendar != "" {
			calendarColors[event.Calendar] = color
		}

		legendByKey[key] = calendarLegendItem{
			key:   key,
			label: label,
			color: color,
		}
	}

	legend := make([]calendarLegendItem, 0, len(legendByKey))
	for _, item := range legendByKey {
		legend = append(legend, item)
	}

	sort.Slice(legend, func(i, j int) bool {
		if legend[i].label != legend[j].label {
			return legend[i].label < legend[j].label
		}
		return legend[i].key < legend[j].key
	})

	return calendarColors, legend
}

func resolveEventCalendar(event ical.Event, calendarByID map[string]ical.Calendar, calendarsByTitle map[string][]ical.Calendar) (ical.Calendar, bool) {
	if event.CalendarID != "" {
		if calendar, ok := calendarByID[event.CalendarID]; ok {
			return calendar, true
		}
	}

	if event.Calendar == "" {
		return ical.Calendar{}, false
	}

	calendars := calendarsByTitle[event.Calendar]
	if len(calendars) == 0 {
		return ical.Calendar{}, false
	}

	return calendars[0], true
}

func calendarKey(calendar ical.Calendar) string {
	if calendar.ID != "" {
		return "id:" + calendar.ID
	}
	return "title:" + calendar.Title
}

func legendKeyForEvent(event ical.Event) string {
	if event.CalendarID != "" {
		return "id:" + event.CalendarID
	}
	if strings.TrimSpace(event.Calendar) != "" {
		return "title:" + strings.TrimSpace(event.Calendar)
	}
	return "title:unknown"
}

func fallbackCalendarLabel(event ical.Event) string {
	if strings.TrimSpace(event.Calendar) != "" {
		return strings.TrimSpace(event.Calendar)
	}
	return "Unknown calendar"
}

func calendarDisplayLabel(calendar ical.Calendar, titleCounts map[string]int) string {
	label := strings.TrimSpace(calendar.Title)
	if label == "" {
		label = "Unknown calendar"
	}

	if titleCounts[calendar.Title] > 1 && strings.TrimSpace(calendar.Source) != "" {
		label += " (" + strings.TrimSpace(calendar.Source) + ")"
	}

	return label
}

func normalizeCalendarColor(color, key string) string {
	color = strings.ToUpper(strings.TrimSpace(color))
	if strings.HasPrefix(color, "#") && (len(color) == 4 || len(color) == 7) {
		return color
	}
	return fallbackCalendarColor(key)
}

func fallbackCalendarColor(key string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))
	return fallbackCalendarColors[int(hasher.Sum32())%len(fallbackCalendarColors)]
}

func eventCalendarColor(event ical.Event, calendarColors map[string]string) string {
	if event.CalendarID != "" {
		if color, ok := calendarColors[event.CalendarID]; ok {
			return color
		}
	}

	if event.Calendar != "" {
		if color, ok := calendarColors[event.Calendar]; ok {
			return color
		}
	}

	return fallbackCalendarColor(legendKeyForEvent(event))
}

func renderLegend(items []calendarLegendItem) string {
	if len(items) == 0 {
		return ""
	}

	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, renderLegendSwatch(item.color)+" "+item.label)
	}

	return strings.Join(parts, "  ")
}
