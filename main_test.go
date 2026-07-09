package main

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestBuildDaySections(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	today := time.Date(2026, 3, 7, 15, 0, 0, 0, loc)

	events := []ical.Event{
		{
			Title:     "Overnight deploy",
			StartDate: time.Date(2026, 3, 7, 23, 30, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 8, 1, 0, 0, 0, loc),
			Calendar:  "Work",
		},
		{
			Title:     "Lunch",
			StartDate: time.Date(2026, 3, 6, 12, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 6, 13, 0, 0, 0, loc),
			Calendar:  "Personal",
		},
		{
			Title:     "Standup",
			StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 9, 30, 0, 0, loc),
			Calendar:  "Work",
		},
	}

	sections := buildDaySections(today, events)

	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(sections))
	}

	if sections[0].label != "Yesterday" || sections[1].label != "Today" || sections[2].label != "Tomorrow" {
		t.Fatalf("unexpected labels: %#v", []string{sections[0].label, sections[1].label, sections[2].label})
	}

	assertTitles(t, sections[0].events, []string{"Lunch"})
	assertTitles(t, sections[1].events, []string{"Standup", "Overnight deploy"})
	assertTitles(t, sections[2].events, []string{"Overnight deploy"})
}

func TestSectionLabelsAdaptRelativeToToday(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	today := time.Date(2026, 3, 7, 15, 0, 0, 0, loc)

	labels := sectionLabels(buildDaySections(today.AddDate(0, 0, 1), nil), today)
	want := []string{"Today", "Tomorrow", "In 2 days"}
	if !reflect.DeepEqual(labels, want) {
		t.Fatalf("expected labels %v, got %v", want, labels)
	}
}

func TestRenderCalendarLayoutPlacesDatesAboveRelativeLabels(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)

	layout := ansi.Strip(renderCalendarLayout(calendarData{
		sections:    buildDaySections(now, nil),
		currentTime: now,
	}, 120))
	lines := strings.Split(layout, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least two header rows, got %d\n%s", len(lines), layout)
	}

	if got := lines[0]; !strings.Contains(got, "Fri Mar 6") || !strings.Contains(got, "Sat Mar 7") || !strings.Contains(got, "Sun Mar 8") {
		t.Fatalf("expected first header row to contain absolute dates, got %q", got)
	}
	if strings.Contains(lines[0], "Yesterday") || strings.Contains(lines[0], "Today") || strings.Contains(lines[0], "Tomorrow") {
		t.Fatalf("expected first header row to omit relative labels, got %q", lines[0])
	}

	if got := lines[1]; !strings.Contains(got, "Yesterday") || !strings.Contains(got, "Today") || !strings.Contains(got, "Tomorrow") {
		t.Fatalf("expected second header row to contain relative labels, got %q", got)
	}
	if strings.Contains(lines[1], "Fri Mar 6") || strings.Contains(lines[1], "Sat Mar 7") || strings.Contains(lines[1], "Sun Mar 8") {
		t.Fatalf("expected second header row to omit absolute dates, got %q", lines[1])
	}
}

func TestRenderLoadingCalendarLayoutPlacesDatesAboveRelativeLabels(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)

	layout := ansi.Strip(renderLoadingCalendarLayout(now, 120))
	lines := strings.Split(layout, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least two header rows, got %d\n%s", len(lines), layout)
	}

	if got := lines[0]; !strings.Contains(got, "Fri Mar 6") || !strings.Contains(got, "Sat Mar 7") || !strings.Contains(got, "Sun Mar 8") {
		t.Fatalf("expected first loading header row to contain absolute dates, got %q", got)
	}
	if got := lines[1]; !strings.Contains(got, "Yesterday") || !strings.Contains(got, "Today") || !strings.Contains(got, "Tomorrow") {
		t.Fatalf("expected second loading header row to contain relative labels, got %q", got)
	}
}

func TestSectionHeaderCellStyleHighlightsTodayInRed(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	todayDate := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	today := daySection{label: "Today", date: todayDate}
	tomorrow := daySection{label: "Tomorrow", date: todayDate.AddDate(0, 0, 1)}

	if got := sectionHeaderCellStyle(today, time.Time{}, headerCellStyle).GetForeground(); got != currentTimeColor {
		t.Fatalf("expected fallback today header to use %v, got %v", currentTimeColor, got)
	}

	highlightedLabel := sectionHeaderCellStyle(today, todayDate.Add(15*time.Hour), dateCellStyle)
	if got := highlightedLabel.GetForeground(); got != currentTimeColor {
		t.Fatalf("expected current-day relative label to use %v, got %v", currentTimeColor, got)
	}
	if !highlightedLabel.GetFaint() {
		t.Fatal("expected highlighted relative label to preserve faint styling")
	}

	if got := sectionHeaderCellStyle(tomorrow, todayDate.Add(15*time.Hour), dateCellStyle).GetForeground(); got != dateCellStyle.GetForeground() {
		t.Fatalf("expected non-current header color to remain %v, got %v", dateCellStyle.GetForeground(), got)
	}
}

func TestNewProgramStartsInAltScreen(t *testing.T) {
	got := startupOptionsForProgram(newProgram())
	want := startupOptionsForOptions(tea.WithAltScreen())
	if want == 0 {
		t.Fatal("expected bubble tea alt-screen option to set startup options")
	}
	if got&want != want {
		t.Fatalf("expected newProgram to enable alt screen, got startup options %d", got)
	}
}

func startupOptionsForOptions(opts ...tea.ProgramOption) int64 {
	var p tea.Program
	for _, opt := range opts {
		opt(&p)
	}
	return startupOptionsForProgram(&p)
}

func startupOptionsForProgram(p *tea.Program) int64 {
	return reflect.ValueOf(p).Elem().FieldByName("startupOptions").Int()
}

func TestEventSlotRangeClampsToDay(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	event := ical.Event{
		Title:     "Overnight deploy",
		StartDate: time.Date(2026, 3, 7, 23, 30, 0, 0, loc),
		EndDate:   time.Date(2026, 3, 8, 1, 0, 0, 0, loc),
	}

	startSlot, endSlot, ok := eventSlotRange(day, event)
	if !ok {
		t.Fatal("expected event to overlap the day")
	}

	if startSlot != 47 || endSlot != 48 {
		t.Fatalf("expected slots 47-48, got %d-%d", startSlot, endSlot)
	}
}

func TestEventSlotRangeUsesWallClockTimeAcrossDST(t *testing.T) {
	loc := mustLoadLocation(t, "America/Los_Angeles")
	day := time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	event := ical.Event{
		Title:     "Standup",
		StartDate: time.Date(2026, 3, 8, 9, 0, 0, 0, loc),
		EndDate:   time.Date(2026, 3, 8, 10, 0, 0, 0, loc),
	}

	startSlot, endSlot, ok := eventSlotRange(day, event)
	if !ok {
		t.Fatal("expected event to overlap the day")
	}

	if startSlot != 18 || endSlot != 20 {
		t.Fatalf("expected 9:00-10:00 AM to occupy slots 18-20 on DST day, got %d-%d", startSlot, endSlot)
	}
}

func TestVisibleSlotWindowAddsPadding(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	today := time.Date(2026, 3, 7, 15, 0, 0, 0, loc)
	sections := buildDaySections(today, []ical.Event{
		{
			Title:     "Standup",
			StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 9, 30, 0, 0, loc),
		},
		{
			Title:     "Workshop",
			StartDate: time.Date(2026, 3, 7, 13, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 15, 0, 0, 0, loc),
		},
	})

	startSlot, endSlot := visibleSlotWindow(sections)
	if startSlot != 16 || endSlot != 32 {
		t.Fatalf("expected window 16-32, got %d-%d", startSlot, endSlot)
	}
}

func TestContentWidthForTerminalSubtractsScreenPadding(t *testing.T) {
	if got := contentWidthForTerminal(120); got != 116 {
		t.Fatalf("expected content width 116, got %d", got)
	}
}

func TestBuildCalendarDecorationsUsesCalendarColorsAndFallbacks(t *testing.T) {
	events := []ical.Event{
		{Title: "Standup", Calendar: "Work", CalendarID: "work"},
		{Title: "Gym", Calendar: "Personal"},
	}

	calendarColors, legend := buildCalendarDecorations([]ical.Calendar{
		{ID: "work", Title: "Work", Color: "#112233"},
	}, events)

	if color := calendarColors["work"]; color != "#112233" {
		t.Fatalf("expected work color #112233, got %q", color)
	}

	personalColor := fallbackCalendarColor("title:Personal")
	if color := calendarColors["Personal"]; color != personalColor {
		t.Fatalf("expected fallback color %q for Personal, got %q", personalColor, color)
	}

	if len(legend) != 2 {
		t.Fatalf("expected 2 legend items, got %d", len(legend))
	}
}

func TestBuildCalendarDecorationsIncludesCalendarsWithoutEvents(t *testing.T) {
	events := []ical.Event{
		{Title: "Standup", Calendar: "Work", CalendarID: "work"},
	}

	_, legend := buildCalendarDecorations([]ical.Calendar{
		{ID: "work", Title: "Work", Color: "#112233"},
		{ID: "holidays", Title: "Holidays", Color: "#445566"},
	}, events)

	labels := make(map[string]bool, len(legend))
	for _, item := range legend {
		labels[item.label] = true
	}

	if !labels["Work"] {
		t.Fatalf("expected legend to include the Work calendar, got %+v", legend)
	}
	if !labels["Holidays"] {
		t.Fatalf("expected legend to include the Holidays calendar even without events, got %+v", legend)
	}
}

func TestEventBackgroundColorDimsCalendarColor(t *testing.T) {
	if got := eventBackgroundColor("#60A5FA"); got != "#1A2E46" {
		t.Fatalf("expected dimmed background color #1A2E46, got %q", got)
	}
}

func TestRecurringMarkerColorScalesCalendarColor(t *testing.T) {
	if got := recurringMarkerColor("#60A5FA"); got != "#487BBB" {
		t.Fatalf("expected scaled recurring marker color #487BBB, got %q", got)
	}
}

func TestEventBackgroundStyleUsesDimmedColor(t *testing.T) {
	if got := eventBackgroundStyle(eventBackgroundColor("#60A5FA")).GetBackground(); got != lipgloss.Color("#1A2E46") {
		t.Fatalf("expected event background style to use #1A2E46, got %v", got)
	}
}

func TestEventForegroundStyleUsesCalendarColor(t *testing.T) {
	if got := eventForegroundStyle("#60A5FA").GetForeground(); got != lipgloss.Color("#60A5FA") {
		t.Fatalf("expected event foreground style to use #60A5FA, got %v", got)
	}
}

func TestRecurringMarkerStyleUsesScaledColor(t *testing.T) {
	if got := recurringMarkerStyle("#60A5FA", "").GetForeground(); got != lipgloss.Color("#487BBB") {
		t.Fatalf("expected recurring marker style to use #487BBB, got %v", got)
	}
}

func TestRecurringMarkerStylePreservesEventBackground(t *testing.T) {
	background := eventBackgroundColor("#60A5FA")
	if got := recurringMarkerStyle("#60A5FA", background).GetBackground(); got != lipgloss.Color(background) {
		t.Fatalf("expected recurring marker style to use %v background, got %v", lipgloss.Color(background), got)
	}
}

func TestDialogChromeUsesSingleSurfaceBackground(t *testing.T) {
	tests := []struct {
		name string
		got  lipgloss.TerminalColor
	}{
		{name: "dialog card", got: dialogCardStyle.GetBackground()},
		{name: "section", got: dialogSectionStyle.GetBackground()},
		{name: "accessory chip", got: dialogAccessoryChipStyle.GetBackground()},
		{name: "input", got: dialogInputStyle.GetBackground()},
		{name: "focused input", got: dialogFocusedInputStyle.GetBackground()},
	}

	for _, tt := range tests {
		if tt.got != dialogSurfaceColor {
			t.Fatalf("%s: expected dialog background %v, got %v", tt.name, dialogSurfaceColor, tt.got)
		}
	}
}

func TestDialogButtonsUseDarkGrayBaseAndRedFocus(t *testing.T) {
	if got := dialogPrimaryButtonStyle.GetBackground(); got != dialogButtonColor {
		t.Fatalf("expected primary button background %v, got %v", dialogButtonColor, got)
	}
	if got := dialogSecondaryButtonStyle.GetBackground(); got != dialogButtonColor {
		t.Fatalf("expected secondary button background %v, got %v", dialogButtonColor, got)
	}
	if got := dialogFocusedPrimaryButtonStyle.GetBackground(); got != currentTimeColor {
		t.Fatalf("expected focused primary button background %v, got %v", currentTimeColor, got)
	}
	if got := dialogFocusedSecondaryButtonStyle.GetBackground(); got != currentTimeColor {
		t.Fatalf("expected focused secondary button background %v, got %v", currentTimeColor, got)
	}
	if dialogFocusedPrimaryButtonStyle.GetBackground() == dialogPrimaryButtonStyle.GetBackground() {
		t.Fatal("expected focused primary button background to change when selected")
	}
	if dialogFocusedSecondaryButtonStyle.GetBackground() == dialogSecondaryButtonStyle.GetBackground() {
		t.Fatal("expected focused secondary button background to change when selected")
	}
}

func TestRenderEventSummaryLineUsesForegroundWithoutAccent(t *testing.T) {
	line := ansi.Strip(renderEventSummaryLine("Standup", "", "#60A5FA", 12))

	if strings.Contains(line, "┃") {
		t.Fatalf("expected all-day summary line to omit the left accent, got %q", line)
	}
	if got := line; got != "  Standup   " {
		t.Fatalf("expected centered all-day summary line, got %q", got)
	}
}

func TestRenderEventSummaryLineRightAlignsRecurringMarker(t *testing.T) {
	line := ansi.Strip(renderEventSummaryLine("Standup", recurringMarker, "#60A5FA", 12))

	if got := line; got != " Standup ⟳  " {
		t.Fatalf("expected recurring marker to be right-aligned, got %q", got)
	}
}

func TestRenderAllDayLinesOmitsRecurringMarker(t *testing.T) {
	lines := renderAllDayLines([]ical.Event{{
		Title:     "Standup",
		Recurring: true,
		AllDay:    true,
	}}, map[string]string{}, 12)

	if len(lines) != 1 {
		t.Fatalf("expected 1 all-day line, got %d", len(lines))
	}
	if strings.Contains(ansi.Strip(lines[0]), recurringMarker) {
		t.Fatalf("expected recurring marker to be omitted from all-day line, got %q", ansi.Strip(lines[0]))
	}
	if got := ansi.Strip(lines[0]); got != "  Standup   " {
		t.Fatalf("expected centered all-day summary line without marker, got %q", got)
	}
}

func TestRenderAllDayLinesCapsAtMaxAllDayEvents(t *testing.T) {
	events := make([]ical.Event, maxAllDayEvents+2)
	for i := range events {
		events[i] = ical.Event{Title: "Event", AllDay: true}
	}

	lines := renderAllDayLines(events, map[string]string{}, 12)

	if len(lines) != maxAllDayEvents {
		t.Fatalf("expected all-day lines to be capped at %d, got %d", maxAllDayEvents, len(lines))
	}
}

func TestRenderAllDayLinesShowsOverflowIndicator(t *testing.T) {
	events := make([]ical.Event, maxAllDayEvents+2)
	for i := range events {
		events[i] = ical.Event{Title: "Event", AllDay: true}
	}

	lines := renderAllDayLines(events, map[string]string{}, 20)

	if len(lines) != maxAllDayEvents {
		t.Fatalf("expected %d all-day lines, got %d", maxAllDayEvents, len(lines))
	}

	hidden := len(events) - (maxAllDayEvents - 1)
	last := ansi.Strip(lines[len(lines)-1])
	if want := "+" + strconv.Itoa(hidden) + " more"; !strings.Contains(last, want) {
		t.Fatalf("expected overflow indicator %q, got %q", want, last)
	}
}

func TestRenderAllDayLinesWithoutOverflowOmitsIndicator(t *testing.T) {
	events := make([]ical.Event, maxAllDayEvents)
	for i := range events {
		events[i] = ical.Event{Title: "Event", AllDay: true}
	}

	lines := renderAllDayLines(events, map[string]string{}, 20)

	if len(lines) != maxAllDayEvents {
		t.Fatalf("expected %d all-day lines, got %d", maxAllDayEvents, len(lines))
	}
	for _, line := range lines {
		if strings.Contains(ansi.Strip(line), "more") {
			t.Fatalf("did not expect overflow indicator, got %q", ansi.Strip(line))
		}
	}
}

func TestRenderCalendarLayoutKeepsTimelineFixedWithAllDayEvents(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)
	day := beginningOfDay(now)

	timelineRow := func(eventCount int) int {
		events := make([]ical.Event, eventCount)
		for i := range events {
			events[i] = ical.Event{Title: "Holiday", AllDay: true, StartDate: day, EndDate: day.AddDate(0, 0, 1)}
		}

		layout := ansi.Strip(renderCalendarLayout(calendarData{
			sections:    buildDaySections(now, events),
			currentTime: now,
		}, 120))

		for i, line := range strings.Split(layout, "\n") {
			if strings.Contains(line, "12:00 AM") {
				return i
			}
		}
		return -1
	}

	baseline := timelineRow(0)
	if baseline == -1 {
		t.Fatal("expected to find the timeline start row")
	}
	for _, count := range []int{1, 2, 3, 4, 6} {
		if got := timelineRow(count); got != baseline {
			t.Fatalf("expected timeline to stay at row %d with %d all-day events, got %d", baseline, count, got)
		}
	}
}

func TestRenderEventMarkerAddsTwoSpacesOfPadding(t *testing.T) {
	marker := ansi.Strip(renderEventMarker(recurringMarker, "#60A5FA", "", 4))

	if got := marker; got != "⟳  " {
		t.Fatalf("expected recurring marker padding to be two spaces, got %q", got)
	}
}

func TestRenderEventLineRightAlignsRecurringMarker(t *testing.T) {
	line := ansi.Strip(renderEventLine("Standup", recurringMarker, "#60A5FA", []eventAccentSegment{{
		accent: eventAccent,
		color:  "#60A5FA",
	}}, 12, eventBackgroundColor("#60A5FA")))

	if got := line; got != "┃ Standup⟳  " {
		t.Fatalf("expected recurring marker to be right-aligned, got %q", got)
	}
}

func TestTimelineStylesUseSubduedColors(t *testing.T) {
	if got := timeAxisStyle.GetForeground(); got != timelineMutedColor {
		t.Fatalf("expected muted time axis color %v, got %v", timelineMutedColor, got)
	}
	if timeAxisStyle.GetFaint() {
		t.Fatal("expected muted time axis labels to avoid the faint style")
	}
	if got := activeTimeAxisStyle.GetForeground(); got != timelineActiveColor {
		t.Fatalf("expected active time axis color %v, got %v", timelineActiveColor, got)
	}
}

func TestCurrentTimeEdgeColorIsDimmedForAdjacentDays(t *testing.T) {
	if currentTimeEdgeColor != lipgloss.Color("#662626") {
		t.Fatalf("expected adjacent-day current time line color #662626, got %v", currentTimeEdgeColor)
	}
	if got := currentTimeEdgeStyle.GetForeground(); got != currentTimeEdgeColor {
		t.Fatalf("expected adjacent-day current time style to use %v, got %v", currentTimeEdgeColor, got)
	}
}

func TestRenderTimedLinesScalesWithDuration(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	lines := renderTimedLines(day, []ical.Event{
		{
			Title:     "Quick sync",
			StartDate: time.Date(2026, 3, 7, 8, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 8, 15, 0, 0, loc),
			Calendar:  "Work",
		},
		{
			Title:     "Standup",
			StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 9, 30, 0, 0, loc),
			Calendar:  "Work",
		},
		{
			Title:     "Workshop",
			StartDate: time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 12, 0, 0, 0, loc),
			Calendar:  "Work",
		},
		{
			Title:     "Design review",
			StartDate: time.Date(2026, 3, 7, 10, 30, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 10, 45, 0, 0, loc),
			Calendar:  "Personal",
		},
		{
			Title:     "QA",
			StartDate: time.Date(2026, 3, 7, 10, 30, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 11, 0, 0, 0, loc),
			Calendar:  "Work",
		},
	}, map[string]string{"Work": "#112233", "Personal": "#445566"}, 16, 24, dayColumnWidth)

	if len(lines[0]) != 1 || !strings.Contains(lines[0][0], "Quick sync") {
		t.Fatalf("expected quick sync label in first slot line, got %#v", lines[0])
	}
	if !strings.Contains(lines[0][0], "│") {
		t.Fatalf("expected short event to include a thin accent, got %#v", lines[0])
	}

	if len(lines[2]) != 1 || !strings.Contains(lines[2][0], "Standup") {
		t.Fatalf("expected standup label in first slot line, got %#v", lines[2])
	}
	if strings.Contains(lines[2][0], "9:00 AM") {
		t.Fatalf("expected standup line to omit event time, got %#v", lines[2])
	}
	if !strings.Contains(lines[2][0], "┃") {
		t.Fatalf("expected timed line to include a thick vertical accent, got %#v", lines[2])
	}

	for _, idx := range []int{4, 5, 6, 7} {
		if len(lines[idx]) == 0 || lines[idx][0] == "" {
			t.Fatalf("expected workshop to occupy slot %d", idx)
		}
		if !strings.Contains(lines[idx][0], "┃") {
			t.Fatalf("expected workshop to include a vertical accent on slot %d, got %#v", idx, lines[idx])
		}
	}

	if len(lines[5]) != 1 {
		t.Fatalf("expected overlapping slot to stay within one row, got %#v", lines[5])
	}
	if strings.Contains(lines[5][0], "Workshop") {
		t.Fatalf("expected continuing event to omit its title in the overlap slot, got %#v", lines[5])
	}
	if !strings.Contains(lines[5][0], "Design") || !strings.Contains(lines[5][0], "QA") {
		t.Fatalf("expected overlapping events to share the same slot row side by side, got %#v", lines[5])
	}
	if strings.Count(lines[5][0], "┃")+strings.Count(lines[5][0], "│") < 3 {
		t.Fatalf("expected overlapping slot to keep three visible accent markers, got %#v", lines[5])
	}

	if strings.Contains(flattenLines(lines), "━") {
		t.Fatalf("expected timed lines to avoid horizontal fill characters: %#v", lines)
	}
}

func TestRenderTimedLinesKeepsEventAccentAlignedAcrossSlots(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	lines := renderTimedLines(day, []ical.Event{
		{
			Title:     "Planning",
			StartDate: time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 11, 0, 0, 0, loc),
			Calendar:  "Work",
		},
		{
			Title:     "Focus time",
			StartDate: time.Date(2026, 3, 7, 10, 30, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 12, 0, 0, 0, loc),
			Calendar:  "Personal",
		},
	}, map[string]string{"Work": "#112233", "Personal": "#445566"}, 20, 24, dayColumnWidth)

	overlapLine := ansi.Strip(lines[1][0])
	continuationLine := ansi.Strip(lines[2][0])

	if got, want := runeIndex(overlapLine, "┃", 2), runeIndex(continuationLine, "┃", 1); got != want {
		t.Fatalf("expected continuing event accent to stay aligned, got overlap %q and continuation %q", overlapLine, continuationLine)
	}
}

func TestRenderTimedLinesShowsRecurringMarkerOnlyOnTopSlot(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	lines := renderTimedLines(day, []ical.Event{{
		Title:     "Workshop",
		StartDate: time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
		EndDate:   time.Date(2026, 3, 7, 12, 0, 0, 0, loc),
		Calendar:  "Work",
		Recurring: true,
	}}, map[string]string{"Work": "#112233"}, 20, 24, dayColumnWidth)

	if got := strings.Count(ansi.Strip(flattenLines(lines)), recurringMarker); got != 1 {
		t.Fatalf("expected exactly one recurring marker across the rendered block, got %d: %#v", got, lines)
	}
	if len(lines[0]) != 1 || !strings.Contains(ansi.Strip(lines[0][0]), recurringMarker) {
		t.Fatalf("expected recurring marker on the first slot line, got %#v", lines[0])
	}
	for i := 1; i < len(lines); i++ {
		if strings.Contains(ansi.Strip(strings.Join(lines[i], "")), recurringMarker) {
			t.Fatalf("expected recurring marker to be omitted from continuation slot %d, got %#v", i, lines[i])
		}
	}
}

func TestRenderTimedLinesShowsRecurringMarkerOnTopVisibleSlotWhenClipped(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	lines := renderTimedLines(day, []ical.Event{{
		Title:     "Workshop",
		StartDate: time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
		EndDate:   time.Date(2026, 3, 7, 12, 0, 0, 0, loc),
		Calendar:  "Work",
		Recurring: true,
	}}, map[string]string{"Work": "#112233"}, 21, 24, dayColumnWidth)

	if got := strings.Count(ansi.Strip(flattenLines(lines)), recurringMarker); got != 1 {
		t.Fatalf("expected exactly one recurring marker across the clipped rendered block, got %d: %#v", got, lines)
	}
	if len(lines[0]) != 1 || !strings.Contains(ansi.Strip(lines[0][0]), recurringMarker) {
		t.Fatalf("expected recurring marker on the top visible slot line, got %#v", lines[0])
	}
	for i := 1; i < len(lines); i++ {
		if strings.Contains(ansi.Strip(strings.Join(lines[i], "")), recurringMarker) {
			t.Fatalf("expected recurring marker to be omitted from later visible slot %d, got %#v", i, lines[i])
		}
	}
}

func TestSlotHasVisibleEventBoundary(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	sections := []daySection{
		{
			label: "Today",
			date:  time.Date(2026, 3, 7, 0, 0, 0, 0, loc),
			events: []ical.Event{
				{
					Title:     "Standup",
					StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
					EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
				},
				{
					Title:     "Offset",
					StartDate: time.Date(2026, 3, 7, 10, 15, 0, 0, loc),
					EndDate:   time.Date(2026, 3, 7, 10, 45, 0, 0, loc),
				},
			},
		},
	}

	if !slotHasVisibleEventBoundary(sections, 18) {
		t.Fatal("expected 9:00 AM slot to match an event boundary")
	}
	if !slotHasVisibleEventBoundary(sections, 20) {
		t.Fatal("expected 10:00 AM slot to match an event boundary")
	}
	if slotHasVisibleEventBoundary(sections, 21) {
		t.Fatal("expected 10:30 AM slot to stay dim when no boundary lands on that label")
	}
}

func TestSlotHasVisibleEventBoundaryUsesWallClockTimeAcrossDST(t *testing.T) {
	loc := mustLoadLocation(t, "America/Los_Angeles")
	sections := []daySection{
		{
			label: "Today",
			date:  time.Date(2026, 3, 8, 0, 0, 0, 0, loc),
			events: []ical.Event{
				{
					Title:     "Standup",
					StartDate: time.Date(2026, 3, 8, 9, 0, 0, 0, loc),
					EndDate:   time.Date(2026, 3, 8, 10, 0, 0, 0, loc),
				},
			},
		},
	}

	if !slotHasVisibleEventBoundary(sections, 18) {
		t.Fatal("expected 9:00 AM slot to match an event boundary on DST day")
	}
	if !slotHasVisibleEventBoundary(sections, 20) {
		t.Fatal("expected 10:00 AM slot to match an event boundary on DST day")
	}
}

func TestCurrentTimeMarkerSlot(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)

	tests := []struct {
		name       string
		now        time.Time
		windowFrom int
		windowTo   int
		wantSlot   int
		wantOK     bool
	}{
		{
			name:       "between slots",
			now:        time.Date(2026, 3, 7, 9, 15, 0, 0, loc),
			windowFrom: 16,
			windowTo:   24,
			wantSlot:   19,
			wantOK:     true,
		},
		{
			name:       "exact boundary",
			now:        time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
			windowFrom: 16,
			windowTo:   24,
			wantSlot:   18,
			wantOK:     true,
		},
		{
			name:       "outside window",
			now:        time.Date(2026, 3, 7, 7, 45, 0, 0, loc),
			windowFrom: 16,
			windowTo:   24,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSlot, gotOK := currentTimeMarkerSlot(tt.now, tt.windowFrom, tt.windowTo)
			if gotSlot != tt.wantSlot || gotOK != tt.wantOK {
				t.Fatalf("expected (%d, %t), got (%d, %t)", tt.wantSlot, tt.wantOK, gotSlot, gotOK)
			}
		})
	}
}

func TestCurrentTimeMarkerSlotUsesWallClockTimeAcrossDST(t *testing.T) {
	loc := mustLoadLocation(t, "America/Los_Angeles")
	now := time.Date(2026, 3, 8, 9, 15, 0, 0, loc)

	slot, ok := currentTimeMarkerSlot(now, 16, 24)
	if !ok {
		t.Fatal("expected current time to be visible on DST day")
	}
	if slot != 19 {
		t.Fatalf("expected 9:15 AM to map to slot 19 on DST day, got %d", slot)
	}
}

func TestRenderCalendarLayoutIncludesCurrentTimeLine(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	today := time.Date(2026, 3, 7, 15, 0, 0, 0, loc)
	layoutWidth := 120
	layout := ansi.Strip(renderCalendarLayout(calendarData{
		sections: buildDaySections(today, []ical.Event{
			{
				Title:     "Standup",
				StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
				EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
				Calendar:  "Work",
			},
		}),
		calendarColors: map[string]string{"Work": "#112233"},
		currentTime:    time.Date(2026, 3, 7, 9, 15, 0, 0, loc),
	}, layoutWidth))

	nine := strings.Index(layout, "9:00 AM")
	current := strings.Index(layout, "9:15 AM")
	nineThirty := strings.Index(layout, "9:30 AM")
	if nine == -1 || current == -1 || nineThirty == -1 {
		t.Fatalf("expected layout to include surrounding times and current time line\n%s", layout)
	}
	if !(nine < current && current < nineThirty) {
		t.Fatalf("expected current time line between 9:00 AM and 9:30 AM\n%s", layout)
	}
	if !strings.Contains(layout, strings.Repeat("─", newCalendarLayout(3, layoutWidth).sectionWidths[0])) {
		t.Fatalf("expected layout to include a horizontal current time line\n%s", layout)
	}
}

func TestRenderCurrentTimeSectionLineUsesActiveEventBackground(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	section := daySection{
		label: "Today",
		date:  day,
		events: []ical.Event{
			{
				Title:     "Standup",
				StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
				EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
				Calendar:  "Work",
			},
		},
	}

	got := renderCurrentTimeSectionLine(
		section,
		time.Date(2026, 3, 7, 9, 15, 0, 0, loc),
		map[string]string{"Work": "#112233"},
		dayColumnWidth,
	)
	want := currentTimeLineStyle.
		Background(lipgloss.Color(eventBackgroundColor("#112233"))).
		Render(strings.Repeat("─", dayColumnWidth))

	if got != want {
		t.Fatalf("expected current time line to use the active event background")
	}
}

func TestRenderCurrentTimeSectionLineOnlyTintsActiveColumns(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	calendarColors := map[string]string{"Work": "#112233", "Personal": "#445566"}
	section := daySection{
		label: "Today",
		date:  day,
		events: []ical.Event{
			{
				Title:     "Long planning",
				StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
				EndDate:   time.Date(2026, 3, 7, 11, 0, 0, 0, loc),
				Calendar:  "Work",
			},
			{
				Title:     "Standup",
				StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
				EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
				Calendar:  "Personal",
			},
		},
	}

	blocks := buildTimedEventBlocks(day, section.events)
	active := activeBlocksAtTime(day, blocks, time.Date(2026, 3, 7, 10, 15, 0, 0, loc))
	if len(active) != 1 {
		t.Fatalf("expected one active event at 10:15, got %d", len(active))
	}

	layout := buildTimedEventLayouts(blocks, calendarColors, dayColumnWidth)[active[0].cluster]
	expectedParts := make([]string, len(layout.columnWidths))
	for layer, width := range layout.columnWidths {
		expectedParts[layer] = currentTimeLineStyle.Render(strings.Repeat("─", width))
	}
	expectedParts[active[0].layer] = currentTimeLineStyle.
		Background(lipgloss.Color(eventBackgroundColor(calendarColors["Work"]))).
		Render(strings.Repeat("─", layout.columnWidths[active[0].layer]))

	got := renderCurrentTimeSectionLine(
		section,
		time.Date(2026, 3, 7, 10, 15, 0, 0, loc),
		calendarColors,
		dayColumnWidth,
	)
	want := strings.Join(expectedParts, currentTimeLineStyle.Render(strings.Repeat("─", lipgloss.Width(layout.separator))))

	if got != want {
		t.Fatalf("expected only the active event column to be tinted")
	}
}

func TestRenderCurrentTimeSectionLineUsesActiveEventBackgroundForAdjacentDays(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 10, 15, 0, 0, loc)

	tests := []struct {
		name     string
		label    string
		day      time.Time
		start    time.Time
		end      time.Time
		calendar string
		color    string
	}{
		{
			name:     "yesterday",
			label:    "Yesterday",
			day:      time.Date(2026, 3, 6, 0, 0, 0, 0, loc),
			start:    time.Date(2026, 3, 6, 10, 0, 0, 0, loc),
			end:      time.Date(2026, 3, 6, 11, 0, 0, 0, loc),
			calendar: "Work",
			color:    "#112233",
		},
		{
			name:     "tomorrow",
			label:    "Tomorrow",
			day:      time.Date(2026, 3, 8, 0, 0, 0, 0, loc),
			start:    time.Date(2026, 3, 8, 10, 0, 0, 0, loc),
			end:      time.Date(2026, 3, 8, 11, 0, 0, 0, loc),
			calendar: "Personal",
			color:    "#445566",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := daySection{
				label: tt.label,
				date:  tt.day,
				events: []ical.Event{{
					Title:     "Overlap",
					StartDate: tt.start,
					EndDate:   tt.end,
					Calendar:  tt.calendar,
				}},
			}

			got := renderCurrentTimeSectionLine(
				section,
				now,
				map[string]string{tt.calendar: tt.color},
				dayColumnWidth,
			)
			want := currentTimeEdgeStyle.
				Background(lipgloss.Color(eventBackgroundColor(tt.color))).
				Render(strings.Repeat("─", dayColumnWidth))

			if got != want {
				t.Fatalf("expected adjacent-day overlap to tint the current time line")
			}
		})
	}
}

func TestRenderCalendarLayoutIncludesTimelineAndLegend(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	today := time.Date(2026, 3, 7, 15, 0, 0, 0, loc)
	events := []ical.Event{
		{
			Title:     "Lunch",
			StartDate: time.Date(2026, 3, 6, 12, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 6, 13, 0, 0, 0, loc),
			Calendar:  "Personal",
		},
		{
			Title:     "Standup",
			StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
			Calendar:  "Work",
		},
		{
			Title:     "Overnight deploy",
			StartDate: time.Date(2026, 3, 7, 23, 30, 0, 0, loc),
			EndDate:   time.Date(2026, 3, 8, 1, 0, 0, 0, loc),
			Calendar:  "Work",
		},
	}

	layout := renderCalendarLayout(calendarData{
		sections:       buildDaySections(today, events),
		calendarColors: map[string]string{"Work": "#112233", "Personal": "#445566"},
		legend: []calendarLegendItem{
			{key: "personal", label: "Personal", color: "#445566"},
			{key: "work", label: "Work", color: "#112233"},
		},
	}, 120)

	for _, want := range []string{"Yesterday", "Today", "Tomorrow", "8:00 AM", "Work", "Personal"} {
		if !strings.Contains(layout, want) {
			t.Fatalf("expected layout to contain %q\n%s", want, layout)
		}
	}

	for _, unwanted := range []string{"Calendar index:", "Calendar", "Yesterday, today, and tomorrow", "━"} {
		if strings.Contains(layout, unwanted) {
			t.Fatalf("expected layout to omit %q\n%s", unwanted, layout)
		}
	}

	if !strings.Contains(layout, "┃") {
		t.Fatalf("expected layout to include vertical accent markers\n%s", layout)
	}
	if !strings.Contains(layout, "●") {
		t.Fatalf("expected layout legend to include circular color markers\n%s", layout)
	}

	for _, unwanted := range []string{"12:00 PM-1:00 PM", "9:00 AM-10:00 AM", "starts 11:30 PM"} {
		if strings.Contains(layout, unwanted) {
			t.Fatalf("expected layout to omit event time %q\n%s", unwanted, layout)
		}
	}
}

func TestRenderCalendarLayoutUsesFullTerminalWidth(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	width := 120
	layout := ansi.Strip(renderCalendarLayout(calendarData{
		sections: buildDaySections(time.Date(2026, 3, 7, 15, 0, 0, 0, loc), []ical.Event{
			{
				Title:     "Standup",
				StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
				EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
			},
		}),
	}, width))

	firstLine := strings.Split(layout, "\n")[0]
	if got := lipgloss.Width(firstLine); got != width {
		t.Fatalf("expected first row to fill terminal width %d, got %d\n%s", width, got, firstLine)
	}
}

func TestRenderCalendarLayoutExpandsToFillAvailableHeight(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)

	layout := ansi.Strip(renderCalendarLayoutWithHeight(calendarData{
		sections: buildDaySections(now, []ical.Event{
			{
				Title:     "Standup",
				StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
				EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
			},
		}),
		currentTime: now,
	}, 120, 20))

	if got := len(strings.Split(layout, "\n")); got != 20 {
		t.Fatalf("expected height-aware layout to use 20 rows, got %d\n%s", got, layout)
	}
}

func TestRenderCalendarLayoutWithHeightKeepsHeadersAndLegendVisibleWhenConstrained(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)

	layout := ansi.Strip(renderCalendarLayoutWithHeight(calendarData{
		sections: buildDaySections(now, []ical.Event{
			{
				Title:     "Standup",
				StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
				EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
				Calendar:  "Work",
			},
		}),
		legend: []calendarLegendItem{
			{key: "work", label: "Work", color: "#60A5FA"},
		},
		currentTime: now,
	}, 120, 8))

	lines := strings.Split(layout, "\n")
	if len(lines) != 8 {
		t.Fatalf("expected constrained layout to render 8 rows, got %d\n%s", len(lines), layout)
	}
	if !strings.Contains(lines[0], "Fri Mar 6") || !strings.Contains(lines[1], "Today") {
		t.Fatalf("expected day labels to remain pinned at the top\n%s", layout)
	}
	if got := lines[len(lines)-1]; !strings.Contains(got, "Work") {
		t.Fatalf("expected legend row to remain visible at the bottom, got %q\n%s", got, layout)
	}
}

func TestRenderCalendarLayoutWithHeightAndScrollSlicesTimedRows(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)

	data := calendarData{
		sections: buildDaySections(day, []ical.Event{
			{
				Title:     "Standup",
				StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
				EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
			},
		}),
	}

	top := ansi.Strip(renderCalendarLayoutWithHeightAndScroll(data, 120, 12, 0))
	if !strings.Contains(top, "12:00 AM") {
		t.Fatalf("expected offset 0 to render the start of the day\n%s", top)
	}

	scrolled := ansi.Strip(renderCalendarLayoutWithHeightAndScroll(data, 120, 12, 6))
	if strings.Contains(scrolled, "12:00 AM") {
		t.Fatalf("expected scrolling to move past the start of the day\n%s", scrolled)
	}
	if !strings.Contains(scrolled, "3:00 AM") {
		t.Fatalf("expected scrolled body to reveal later timed rows\n%s", scrolled)
	}
}

func TestRenderCalendarLayoutRendersFullDayWithoutEvents(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)

	layout := ansi.Strip(renderCalendarLayoutWithHeight(calendarData{
		sections: buildDaySections(day, nil),
	}, 120, 0))

	for _, want := range []string{"12:00 AM", "11:30 PM"} {
		if !strings.Contains(layout, want) {
			t.Fatalf("expected empty day to render the full timeline including %q\n%s", want, layout)
		}
	}
}

func TestRenderCalendarLayoutWithPinnedSlotWindowKeepsSameTimeRange(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	slotWindow := calendarSlotWindow{start: 16, end: 26}

	layout := ansi.Strip(renderCalendarLayoutWithHeightAndScrollUsingWindow(calendarData{
		sections: buildDaySections(day, []ical.Event{
			{
				Title:     "Afternoon workshop",
				StartDate: time.Date(2026, 3, 7, 14, 0, 0, 0, loc),
				EndDate:   time.Date(2026, 3, 7, 15, 0, 0, 0, loc),
			},
		}),
	}, 120, 15, 0, slotWindow))

	if !strings.Contains(layout, "8:00 AM") || !strings.Contains(layout, "12:30 PM") {
		t.Fatalf("expected pinned slot window to preserve the original time range\n%s", layout)
	}
	for _, line := range strings.Split(layout, "\n") {
		if strings.TrimSpace(line) == "2:00 PM" {
			t.Fatalf("expected pinned slot window to avoid recalculating around the new day events\n%s", layout)
		}
	}
}

func TestRenderLoadingCalendarLayoutWithPinnedSlotWindowKeepsSameTimeRange(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)
	slotWindow := calendarSlotWindow{start: 20, end: 30}

	layout := ansi.Strip(renderLoadingCalendarLayoutForDayWithHeightAndScrollUsingWindow(beginningOfDay(now), now, 120, 15, 0, slotWindow))

	if !strings.Contains(layout, "10:00 AM") || !strings.Contains(layout, "2:30 PM") {
		t.Fatalf("expected loading layout to reuse the pinned time range\n%s", layout)
	}
	if strings.Contains(layout, "8:00 AM") {
		t.Fatalf("expected loading layout to avoid falling back to the default time range\n%s", layout)
	}
}

func TestModelUpdateRefreshesCurrentTime(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)
	m := model{}

	updated, cmd := m.Update(currentTimeTickMsg(now))
	gotModel := updated.(model)
	if !gotModel.data.currentTime.Equal(now) {
		t.Fatalf("expected current time %v, got %v", now, gotModel.data.currentTime)
	}
	if cmd == nil {
		t.Fatal("expected current time update to schedule another tick")
	}
}

func TestModelUpdateSchedulesCalendarRefresh(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)
	m := model{}

	updated, cmd := m.Update(calendarRefreshTickMsg(now))
	gotModel := updated.(model)
	if !gotModel.data.currentTime.Equal(now) {
		t.Fatalf("expected refresh tick to update current time to %v, got %v", now, gotModel.data.currentTime)
	}
	if cmd == nil {
		t.Fatal("expected refresh tick to schedule a reload and another refresh tick")
	}
}

func TestModelUpdateStoresCalendarWatcher(t *testing.T) {
	changes := make(chan struct{}, 1)
	changes <- struct{}{}

	m := model{}
	updated, cmd := m.Update(watchCalendarReadyMsg{
		changes: changes,
		cancel:  func() {},
	})
	gotModel := updated.(model)

	if gotModel.watchChanges == nil {
		t.Fatal("expected watcher channel to be stored on the model")
	}
	if gotModel.watchCancel == nil {
		t.Fatal("expected watcher cancel function to be stored on the model")
	}
	if cmd == nil {
		t.Fatal("expected watcher setup to schedule a wait command")
	}
	if _, ok := cmd().(watchCalendarChangedMsg); !ok {
		t.Fatal("expected wait command to emit a calendar change message")
	}
}

func TestWaitForCalendarChangeCmdHandlesClosedChannel(t *testing.T) {
	changes := make(chan struct{})
	close(changes)

	msg := waitForCalendarChangeCmd(changes)()
	if _, ok := msg.(watchCalendarStoppedMsg); !ok {
		t.Fatalf("expected stopped watcher message, got %T", msg)
	}
}

func TestModelUpdateSetsWatchError(t *testing.T) {
	watchErr := errors.New("watch failed")

	updated, cmd := model{}.Update(watchCalendarReadyMsg{err: watchErr})
	gotModel := updated.(model)

	if !errors.Is(gotModel.err, watchErr) {
		t.Fatalf("expected watch error %v, got %v", watchErr, gotModel.err)
	}
	if cmd != nil {
		t.Fatal("expected no follow-up command on watch setup error")
	}
}

func TestOpenCalendarWatchRetriesWatcherAlreadyActive(t *testing.T) {
	attempts := 0
	var sleeps []time.Duration
	changes := make(chan struct{})

	gotChanges, cancel, err := openCalendarWatch(
		func(context.Context) (<-chan struct{}, error) {
			attempts++
			if attempts < 3 {
				return nil, errors.New("calendar: watcher already active")
			}
			return changes, nil
		},
		func(delay time.Duration) {
			sleeps = append(sleeps, delay)
		},
	)
	if err != nil {
		t.Fatalf("expected watch startup to recover, got %v", err)
	}
	if gotChanges != changes {
		t.Fatal("expected successful watcher channel to be returned")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if len(sleeps) != 2 || sleeps[0] != watchRetryDelay || sleeps[1] != watchRetryDelay*2 {
		t.Fatalf("unexpected retry delays: %#v", sleeps)
	}
	if cancel == nil {
		t.Fatal("expected cancel function for successful watcher")
	}
	cancel()
}

func TestOpenCalendarWatchStopsOnPermanentError(t *testing.T) {
	attempts := 0
	_, cancel, err := openCalendarWatch(
		func(context.Context) (<-chan struct{}, error) {
			attempts++
			return nil, errors.New("calendar: failed to start watcher")
		},
		func(time.Duration) {},
	)
	if err == nil {
		t.Fatal("expected startup error")
	}
	if attempts != 1 {
		t.Fatalf("expected one attempt for a permanent error, got %d", attempts)
	}
	if cancel != nil {
		t.Fatal("expected no cancel function on failed startup")
	}
}

func TestModelQuitCancelsCalendarWatcher(t *testing.T) {
	cancelled := false
	m := model{
		watchChanges: make(chan struct{}),
		watchCancel: func() {
			cancelled = true
		},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	gotModel := updated.(model)

	if !cancelled {
		t.Fatal("expected quit to cancel the active calendar watcher")
	}
	if gotModel.watchChanges != nil {
		t.Fatal("expected quit to clear the watcher channel")
	}
	if gotModel.watchCancel != nil {
		t.Fatal("expected quit to clear the watcher cancel function")
	}
	if cmd == nil {
		t.Fatal("expected quit command to be returned")
	}
}

func TestModelUpdateRefreshesCalendarOnTKey(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)
	m := model{
		watchChanges:  make(chan struct{}),
		watchCancel:   func() {},
		viewDayOffset: 3,
		data: calendarData{
			currentTime: now,
		},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	gotModel := updated.(model)

	if cmd == nil {
		t.Fatal("expected t key to trigger a refresh command")
	}
	if gotModel.watchChanges == nil {
		t.Fatal("expected manual refresh to keep the watcher active")
	}
	if gotModel.watchCancel == nil {
		t.Fatal("expected manual refresh to keep the watcher cancel function")
	}
	if gotModel.viewDayOffset != 0 {
		t.Fatalf("expected t key to reset the viewed day offset, got %d", gotModel.viewDayOffset)
	}
}

func TestModelUpdateMovesCalendarWindowOnHAndLKeys(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)

	tests := []struct {
		name       string
		key        rune
		start      int
		wantOffset int
	}{
		{name: "backward", key: 'h', start: 0, wantOffset: -1},
		{name: "forward", key: 'l', start: 0, wantOffset: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				viewDayOffset: tt.start,
				data: calendarData{
					currentTime: now,
				},
			}

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.key}})
			gotModel := updated.(model)

			if cmd == nil {
				t.Fatalf("expected %q key to trigger a reload command", string(tt.key))
			}
			if gotModel.viewDayOffset != tt.wantOffset {
				t.Fatalf("expected %q key to set viewed day offset to %d, got %d", string(tt.key), tt.wantOffset, gotModel.viewDayOffset)
			}
		})
	}
}

func TestModelUpdateHorizontalNavigationPreservesScrollOffset(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	m := model{
		width:             120,
		height:            10,
		timedScrollOffset: 2,
		data: calendarData{
			sections: buildDaySections(day, []ical.Event{
				{
					Title:     "Standup",
					StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
					EndDate:   time.Date(2026, 3, 7, 12, 0, 0, 0, loc),
				},
			}),
		},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	gotModel := updated.(model)

	if cmd == nil {
		t.Fatal("expected l key to trigger a reload command")
	}
	if gotModel.viewDayOffset != 1 {
		t.Fatalf("expected l key to advance the viewed day, got %d", gotModel.viewDayOffset)
	}
	if gotModel.timedScrollOffset != 2 {
		t.Fatalf("expected horizontal navigation to preserve the vertical scroll offset, got %d", gotModel.timedScrollOffset)
	}
}

func TestModelInitialFocusKeepsFullDayScrollable(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)
	m := model{
		width:        120,
		height:       20,
		focusPending: true,
		data: calendarData{
			currentTime: now,
			sections: buildDaySections(day, []ical.Event{
				{
					Title:     "Standup",
					StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
					EndDate:   time.Date(2026, 3, 7, 12, 0, 0, 0, loc),
				},
			}),
		},
	}

	m.applyPendingFocus()

	if m.focusPending {
		t.Fatal("expected pending focus to be consumed once the layout is known")
	}
	if m.timedScrollOffset <= 0 {
		t.Fatalf("expected the initial view to focus past the empty early hours, got offset %d", m.timedScrollOffset)
	}
	if maxOffset := m.maxTimedScrollOffset(); maxOffset <= m.timedScrollOffset {
		t.Fatalf("expected the rest of the day to stay scrollable beyond focus offset %d, got max %d", m.timedScrollOffset, maxOffset)
	}
}

func TestModelUpdateScrollsTimedViewportWithJKAndArrowKeys(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	m := model{
		width:  120,
		height: 10,
		data: calendarData{
			sections: buildDaySections(day, []ical.Event{
				{
					Title:     "Standup",
					StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
					EndDate:   time.Date(2026, 3, 7, 12, 0, 0, 0, loc),
				},
			}),
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	gotModel := updated.(model)
	if gotModel.timedScrollOffset != 1 {
		t.Fatalf("expected j to scroll down one row, got %d", gotModel.timedScrollOffset)
	}

	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyDown})
	gotModel = updated.(model)
	if gotModel.timedScrollOffset != 2 {
		t.Fatalf("expected down arrow to scroll down one row, got %d", gotModel.timedScrollOffset)
	}

	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyUp})
	gotModel = updated.(model)
	if gotModel.timedScrollOffset != 1 {
		t.Fatalf("expected up arrow to scroll up one row, got %d", gotModel.timedScrollOffset)
	}

	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	gotModel = updated.(model)
	if gotModel.timedScrollOffset != 0 {
		t.Fatalf("expected k to scroll back to the top, got %d", gotModel.timedScrollOffset)
	}
}

func TestModelUpdateClampsTimedViewportScroll(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	m := model{
		width:             120,
		height:            10,
		timedScrollOffset: 99,
		data: calendarData{
			sections: buildDaySections(day, []ical.Event{
				{
					Title:     "Workshop",
					StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
					EndDate:   time.Date(2026, 3, 7, 11, 0, 0, 0, loc),
				},
			}),
		},
	}

	m.clampTimedScrollOffset()
	if want := m.maxTimedScrollOffset(); m.timedScrollOffset != want {
		t.Fatalf("expected scroll offset to clamp to %d, got %d", want, m.timedScrollOffset)
	}
}

func TestModelUpdateOpensCreateDialogOnNKey(t *testing.T) {
	updated, _ := model{}.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	gotModel := updated.(model)

	if !gotModel.showCreateDialog {
		t.Fatal("expected n key to open the create-event dialog")
	}
	if gotModel.createDialog.focusIndex != dialogFocusTitle {
		t.Fatalf("expected title input to be focused, got %d", gotModel.createDialog.focusIndex)
	}
	if gotModel.createDialog.titleInput.Placeholder != "Title" {
		t.Fatalf("expected title placeholder Title, got %q", gotModel.createDialog.titleInput.Placeholder)
	}
	if gotModel.createDialog.durationInput.value != 30 {
		t.Fatalf("expected default duration 30 minutes, got %d", gotModel.createDialog.durationInput.value)
	}
	if gotModel.createDialog.hourInput.value < 0 || gotModel.createDialog.hourInput.value > 23 {
		t.Fatalf("expected default hour to be between 0 and 23, got %d", gotModel.createDialog.hourInput.value)
	}
	if gotModel.createDialog.minuteInput.value%dialogMinuteStep != 0 {
		t.Fatalf("expected default minutes to use %d-minute increments, got %d", dialogMinuteStep, gotModel.createDialog.minuteInput.value)
	}
	if gotModel.createDialog.startDate.IsZero() {
		t.Fatal("expected create dialog to prefill the start date")
	}
	if gotModel.createDialog.startDate != beginningOfDay(gotModel.createDialog.startDate) {
		t.Fatalf("expected dialog start date to be normalized to midnight, got %v", gotModel.createDialog.startDate)
	}
}

func TestModelUpdateUsesViewedDayForCreateDialog(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 10, 0, 0, loc)
	m := model{
		viewDayOffset: 2,
		data: calendarData{
			currentTime: now,
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	gotModel := updated.(model)

	wantDate := time.Date(2026, 3, 9, 0, 0, 0, 0, loc)
	if !gotModel.createDialog.startDate.Equal(wantDate) {
		t.Fatalf("expected create dialog to default to viewed day %v, got %v", wantDate, gotModel.createDialog.startDate)
	}
	if gotModel.createDialog.hourInput.value != 9 || gotModel.createDialog.minuteInput.value != 15 {
		t.Fatalf("expected create dialog to keep the current rounded time on the viewed day, got %02d:%02d", gotModel.createDialog.hourInput.value, gotModel.createDialog.minuteInput.value)
	}
}

func TestModelUpdateDialogCapturesTypedHInTitle(t *testing.T) {
	dialog, _ := newCreateEventDialog(time.Now())
	m := model{
		showCreateDialog: true,
		viewDayOffset:    2,
		createDialog:     dialog,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	gotModel := updated.(model)

	if gotModel.viewDayOffset != 2 {
		t.Fatalf("expected viewed day offset to stay unchanged while typing, got %d", gotModel.viewDayOffset)
	}
	if gotModel.createDialog.titleInput.Value() != "h" {
		t.Fatalf("expected title input to receive the typed h, got %q", gotModel.createDialog.titleInput.Value())
	}
}

func TestModelUpdateDialogTabMovesFocus(t *testing.T) {
	dialog, _ := newCreateEventDialog(time.Now())
	m := model{
		showCreateDialog: true,
		createDialog:     dialog,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	gotModel := updated.(model)
	if gotModel.createDialog.focusIndex != dialogFocusMonth {
		t.Fatalf("expected tab to move focus to month input, got %d", gotModel.createDialog.focusIndex)
	}

	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyTab})
	gotModel = updated.(model)
	if gotModel.createDialog.focusIndex != dialogFocusDay {
		t.Fatalf("expected second tab to move focus to day input, got %d", gotModel.createDialog.focusIndex)
	}

	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyTab})
	gotModel = updated.(model)
	if gotModel.createDialog.focusIndex != dialogFocusYear {
		t.Fatalf("expected third tab to move focus to year input, got %d", gotModel.createDialog.focusIndex)
	}

	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyTab})
	gotModel = updated.(model)
	if gotModel.createDialog.focusIndex != dialogFocusHour {
		t.Fatalf("expected fourth tab to move focus to hour input, got %d", gotModel.createDialog.focusIndex)
	}
}

func TestModelUpdateDialogUpDownAdjustsFocusedNumberInputs(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 7, 8, 0, 0, 0, loc))
	dialog.startDate = time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	dialog.hourInput.value = 23
	dialog.minuteInput.value = 45
	dialog.durationInput.value = 30
	_ = dialog.setFocus(dialogFocusMinute)

	m := model{
		showCreateDialog: true,
		createDialog:     dialog,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	gotModel := updated.(model)
	if gotModel.createDialog.hourInput.value != 0 || gotModel.createDialog.minuteInput.value != 0 {
		t.Fatalf("expected minute increment to roll over to 00:00, got %02d:%02d", gotModel.createDialog.hourInput.value, gotModel.createDialog.minuteInput.value)
	}
	if gotModel.createDialog.startDate.Day() != 8 {
		t.Fatalf("expected minute rollover to advance the day, got %v", gotModel.createDialog.startDate)
	}

	_ = gotModel.createDialog.setFocus(dialogFocusHour)
	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyDown})
	gotModel = updated.(model)
	if gotModel.createDialog.hourInput.value != 23 || gotModel.createDialog.minuteInput.value != 0 {
		t.Fatalf("expected hour decrement to roll back to 23:00, got %02d:%02d", gotModel.createDialog.hourInput.value, gotModel.createDialog.minuteInput.value)
	}
	if gotModel.createDialog.startDate.Day() != 7 {
		t.Fatalf("expected hour rollover to move back to the previous day, got %v", gotModel.createDialog.startDate)
	}

	_ = gotModel.createDialog.setFocus(dialogFocusDuration)
	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyDown})
	gotModel = updated.(model)
	if gotModel.createDialog.durationInput.value != 15 {
		t.Fatalf("expected duration decrement to keep %d-minute steps, got %d", dialogMinuteStep, gotModel.createDialog.durationInput.value)
	}

	_ = gotModel.createDialog.setFocus(dialogFocusDay)
	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyUp})
	gotModel = updated.(model)
	if gotModel.createDialog.startDate.Day() != 8 {
		t.Fatalf("expected day increment to advance the date, got %v", gotModel.createDialog.startDate)
	}

	_ = gotModel.createDialog.setFocus(dialogFocusMonth)
	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyDown})
	gotModel = updated.(model)
	if gotModel.createDialog.startDate.Month() != time.February {
		t.Fatalf("expected month decrement to update the month, got %v", gotModel.createDialog.startDate)
	}

	_ = gotModel.createDialog.setFocus(dialogFocusYear)
	updated, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyUp})
	gotModel = updated.(model)
	if gotModel.createDialog.startDate.Year() != 2027 {
		t.Fatalf("expected year increment to update the year, got %v", gotModel.createDialog.startDate)
	}
}

func TestModelUpdateDialogTypingDigitsPopulatesDateFields(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 7, 8, 0, 0, 0, loc))
	dialog.startDate = time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	_ = dialog.setFocus(dialogFocusMonth)

	m := model{
		showCreateDialog: true,
		createDialog:     dialog,
	}

	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'1'}},
		{Type: tea.KeyRunes, Runes: []rune{'2'}},
		{Type: tea.KeyTab},
		{Type: tea.KeyRunes, Runes: []rune{'2'}},
		{Type: tea.KeyRunes, Runes: []rune{'5'}},
		{Type: tea.KeyTab},
		{Type: tea.KeyRunes, Runes: []rune{'2'}},
		{Type: tea.KeyRunes, Runes: []rune{'0'}},
		{Type: tea.KeyRunes, Runes: []rune{'2'}},
		{Type: tea.KeyRunes, Runes: []rune{'7'}},
		{Type: tea.KeyTab},
	} {
		updated, _ := m.Update(msg)
		m = updated.(model)
	}

	wantDate := time.Date(2027, 12, 25, 0, 0, 0, 0, loc)
	if !m.createDialog.startDate.Equal(wantDate) {
		t.Fatalf("expected typed date %v, got %v", wantDate, m.createDialog.startDate)
	}
	if m.createDialog.focusIndex != dialogFocusHour {
		t.Fatalf("expected focus to advance to hour, got %d", m.createDialog.focusIndex)
	}
}

func TestModelUpdateDialogTypingDigitsPopulatesTimeAndDurationFields(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 7, 8, 0, 0, 0, loc))
	dialog.hourInput.value = 8
	dialog.minuteInput.value = 0
	dialog.durationInput.value = 30
	_ = dialog.setFocus(dialogFocusHour)

	m := model{
		showCreateDialog: true,
		createDialog:     dialog,
	}

	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'1'}},
		{Type: tea.KeyRunes, Runes: []rune{'8'}},
		{Type: tea.KeyTab},
		{Type: tea.KeyRunes, Runes: []rune{'3'}},
		{Type: tea.KeyRunes, Runes: []rune{'0'}},
		{Type: tea.KeyTab},
		{Type: tea.KeyRunes, Runes: []rune{'1'}},
		{Type: tea.KeyRunes, Runes: []rune{'2'}},
		{Type: tea.KeyRunes, Runes: []rune{'0'}},
		{Type: tea.KeyTab},
	} {
		updated, _ := m.Update(msg)
		m = updated.(model)
	}

	if got := m.createDialog.hourInput.value; got != 18 {
		t.Fatalf("expected typed hour 18, got %d", got)
	}
	if got := m.createDialog.minuteInput.value; got != 30 {
		t.Fatalf("expected typed minute 30, got %d", got)
	}
	if got := m.createDialog.durationInput.value; got != 120 {
		t.Fatalf("expected typed duration 120, got %d", got)
	}
	if m.createDialog.focusIndex != dialogFocusSubmit {
		t.Fatalf("expected focus to advance to submit, got %d", m.createDialog.focusIndex)
	}
}

func TestModelUpdateDialogInvalidTypedMinuteShowsError(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 7, 8, 0, 0, 0, loc))
	dialog.minuteInput.value = 0
	_ = dialog.setFocus(dialogFocusMinute)

	m := model{
		showCreateDialog: true,
		createDialog:     dialog,
	}

	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'1'}},
		{Type: tea.KeyRunes, Runes: []rune{'7'}},
		{Type: tea.KeyTab},
	} {
		updated, _ := m.Update(msg)
		m = updated.(model)
	}

	if m.createDialog.err == nil || !strings.Contains(m.createDialog.err.Error(), "15-minute increments") {
		t.Fatalf("expected minute validation error, got %v", m.createDialog.err)
	}
	if got := m.createDialog.minuteInput.value; got != 0 {
		t.Fatalf("expected invalid minute input to leave the value unchanged, got %d", got)
	}
	if m.createDialog.focusIndex != dialogFocusMinute {
		t.Fatalf("expected focus to remain on minutes after invalid input, got %d", m.createDialog.focusIndex)
	}
}

func TestCreateEventDialogCreateInputUsesNumericFields(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 7, 8, 0, 0, 0, loc))
	dialog.titleInput.SetValue("Standup")
	dialog.startDate = time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	dialog.hourInput.value = 18
	dialog.minuteInput.value = 15
	dialog.durationInput.value = 45

	input, err := dialog.createInput(time.Date(2026, 3, 7, 8, 0, 0, 0, loc))
	if err != nil {
		t.Fatalf("expected numeric dialog fields to build an event input: %v", err)
	}

	wantStart := time.Date(2026, 3, 8, 18, 15, 0, 0, loc)
	wantEnd := wantStart.Add(45 * time.Minute)
	if !input.StartDate.Equal(wantStart) {
		t.Fatalf("expected start %v, got %v", wantStart, input.StartDate)
	}
	if !input.EndDate.Equal(wantEnd) {
		t.Fatalf("expected end %v, got %v", wantEnd, input.EndDate)
	}
}

func TestModelUpdateDialogSubmitStartsCreateCommand(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 7, 8, 0, 0, 0, loc))
	dialog.titleInput.SetValue("Standup")
	dialog.hourInput.value = 9
	dialog.minuteInput.value = 0
	dialog.durationInput.value = 30
	_ = dialog.setFocus(dialogFocusSubmit)

	m := model{
		data: calendarData{
			currentTime: time.Date(2026, 3, 7, 8, 0, 0, 0, loc),
		},
		showCreateDialog: true,
		createDialog:     dialog,
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	gotModel := updated.(model)

	if cmd == nil {
		t.Fatal("expected submit to return a create-event command")
	}
	if !gotModel.createDialog.submitting {
		t.Fatal("expected submit to mark the dialog as submitting")
	}
}

func TestModelUpdateDialogCancelClosesDialog(t *testing.T) {
	dialog, _ := newCreateEventDialog(time.Now())
	_ = dialog.setFocus(dialogFocusCancel)
	m := model{
		showCreateDialog: true,
		createDialog:     dialog,
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	gotModel := updated.(model)

	if cmd != nil {
		t.Fatal("expected cancel to close the dialog without a command")
	}
	if gotModel.showCreateDialog {
		t.Fatal("expected cancel to close the create-event dialog")
	}
}

func TestViewAddsOuterPadding(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	width := 120
	m := model{
		width: width,
		data: calendarData{
			sections: buildDaySections(time.Date(2026, 3, 7, 15, 0, 0, 0, loc), []ical.Event{
				{
					Title:     "Standup",
					StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
					EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
				},
			}),
		},
	}

	view := ansi.Strip(m.View())
	lines := strings.Split(view, "\n")
	if strings.TrimSpace(lines[0]) != "" {
		t.Fatalf("expected top padding line, got %q", lines[0])
	}
	if got := lipgloss.Width(lines[1]); got != width {
		t.Fatalf("expected padded line width %d, got %d\n%s", width, got, lines[1])
	}
	if !strings.HasPrefix(lines[1], strings.Repeat(" ", screenPaddingX)) {
		t.Fatalf("expected horizontal padding on content lines, got %q", lines[1])
	}
}

func TestViewUsesFullTerminalHeightWhenSpaceAllows(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)
	view := ansi.Strip(model{
		width:  120,
		height: 60,
		data: calendarData{
			sections:    buildDaySections(now, nil),
			currentTime: now,
		},
	}.View())

	if got := len(strings.Split(view, "\n")); got != 60 {
		t.Fatalf("expected view to use full terminal height 60, got %d\n%s", got, view)
	}
}

func TestRenderLoadingCalendarLayoutIncludesCalendarChrome(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)

	layout := ansi.Strip(renderLoadingCalendarLayout(now, 120))

	for _, want := range []string{"Yesterday", "Today", "Tomorrow", "All day", "8:00 AM", "9:15 AM"} {
		if !strings.Contains(layout, want) {
			t.Fatalf("expected loading layout to contain %q\n%s", want, layout)
		}
	}

	for _, unwanted := range []string{"Loading calendar...", "┃", "●"} {
		if strings.Contains(layout, unwanted) {
			t.Fatalf("expected loading layout to omit %q\n%s", unwanted, layout)
		}
	}
}

func TestBuildLoadingSectionsUsesViewedDay(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	viewDay := time.Date(2026, 3, 10, 0, 0, 0, 0, loc)

	sections := buildLoadingSections(viewDay)
	wantDays := []time.Time{
		viewDay.AddDate(0, 0, -1),
		viewDay,
		viewDay.AddDate(0, 0, 1),
	}

	if len(sections) != len(wantDays) {
		t.Fatalf("expected %d loading sections, got %d", len(wantDays), len(sections))
	}
	for i, want := range wantDays {
		if !sections[i].date.Equal(want) {
			t.Fatalf("section %d: expected date %v, got %v", i, want, sections[i].date)
		}
	}
}

func TestViewRendersCreateEventDialog(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 8, 16, 15, 0, 0, loc))
	dialog.startDate = time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	dialog.hourInput.value = 16
	dialog.minuteInput.value = 15
	dialog.durationInput.value = 30
	_ = dialog.setFocus(dialogFocusTitle)

	view := ansi.Strip(model{
		width:            120,
		height:           40,
		showCreateDialog: true,
		createDialog:     dialog,
	}.View())

	for _, want := range []string{
		"Title",
		"Date",
		"Time",
		"Duration",
		"Mar 8, 2026 4:15PM – 4:45PM",
		"Mar",
		"08",
		"2026",
		"Create",
		"Cancel",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected dialog view to contain %q\n%s", want, view)
		}
	}
	for _, unwanted := range []string{
		"Event",
		"Reminder",
		"Add Location or Video Call",
		"Alert 15 minutes before start (default)",
		"Add Alert, Repeat, or Travel Time",
		"Default calendar",
		"Tab moves",
		"Dinner",
		"Add Invitees",
		"Add Notes, URL, or Attachments",
		"▴",
		"▾",
	} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("expected dialog view to omit %q\n%s", unwanted, view)
		}
	}
	if summaryIndex, titleIndex := strings.Index(view, "Mar 8, 2026 4:15PM – 4:45PM"), strings.Index(view, "Title"); summaryIndex == -1 || titleIndex == -1 || summaryIndex >= titleIndex {
		t.Fatalf("expected dialog summary above the title input\n%s", view)
	}
	if titleIndex, dateIndex, timeIndex := strings.Index(view, "Title"), strings.Index(view, "Date"), strings.Index(view, "Time"); titleIndex == -1 || dateIndex == -1 || timeIndex == -1 || !(titleIndex < dateIndex && dateIndex < timeIndex) {
		t.Fatalf("expected date field between title and time\n%s", view)
	}
}

func TestRenderCreateEventDialogUses80x24WhenSpaceAllows(t *testing.T) {
	dialog, _ := newCreateEventDialog(time.Now())

	rendered := renderCreateEventDialog(dialog, 120, 40)

	if got := lipgloss.Width(rendered); got != dialogCols {
		t.Fatalf("expected dialog width %d, got %d", dialogCols, got)
	}
	if got := lipgloss.Height(rendered); got != dialogRows {
		t.Fatalf("expected dialog height %d, got %d", dialogRows, got)
	}
}

func TestRenderDialogTextInputKeepsPlaceholderPlainWhenUnfocused(t *testing.T) {
	input := newDialogTextInput("Title", "", 24)
	got := renderDialogTextInput(input, 14, false)
	if got != padRight("Title", 14) {
		t.Fatalf("expected unfocused placeholder %q, got %q", padRight("Title", 14), got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("expected unfocused placeholder to avoid ANSI sequences, got %q", got)
	}
}

func TestRenderDialogTextInputFocusedOmitsPromptPrefix(t *testing.T) {
	input := newDialogTextInput("Title", "", 24)
	_ = input.Focus()

	got := renderDialogTextInput(input, 14, true)
	if strings.Contains(ansi.Strip(got), "| Title") {
		t.Fatalf("expected focused title input to omit the prompt prefix, got %q", ansi.Strip(got))
	}
	if !strings.Contains(ansi.Strip(got), "Title") {
		t.Fatalf("expected focused title input to include the placeholder text, got %q", ansi.Strip(got))
	}
}

func TestModelUpdateDialogRoutesCursorBlinkMessages(t *testing.T) {
	dialog, focusCmd := newCreateEventDialog(time.Now())
	m := model{
		showCreateDialog: true,
		createDialog:     dialog,
	}
	if focusCmd == nil {
		t.Fatal("expected focused dialog to return an initial cursor command")
	}

	updated, cmd := m.Update(focusCmd())
	gotModel := updated.(model)

	if !gotModel.showCreateDialog {
		t.Fatal("expected blink message to leave the dialog open")
	}
	if cmd == nil {
		t.Fatal("expected blink message to schedule another cursor update")
	}
}

func TestRenderDialogHeaderSectionOmitsSeparateTitleLabel(t *testing.T) {
	dialog, _ := newCreateEventDialog(time.Now())
	_ = dialog.setFocus(dialogFocusTitle)

	section := ansi.Strip(renderDialogHeaderSection(dialog, 40))
	lines := strings.Split(section, "\n")
	if len(lines) != 1 {
		t.Fatalf("expected header section to render a single title input row\n%s", section)
	}
	if !strings.Contains(lines[0], "Title") {
		t.Fatalf("expected header section to contain the focused title input\n%s", section)
	}
	if strings.Contains(lines[0], "| Title") {
		t.Fatalf("expected header section to omit the prompt prefix when focused\n%s", section)
	}
	if strings.Contains(section, "▴") || strings.Contains(section, "▾") {
		t.Fatalf("expected header section to omit title chevrons\n%s", section)
	}
}

func TestRenderDialogDateSectionPlacesMonthDayYearBesideLabel(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 8, 7, 45, 0, 0, loc))
	dialog.startDate = time.Date(2026, 3, 8, 0, 0, 0, 0, loc)

	section := ansi.Strip(renderDialogDateSection(dialog, 40))
	if !strings.Contains(section, "Date") || !strings.Contains(section, "Mar") || !strings.Contains(section, "08") || !strings.Contains(section, "2026") {
		t.Fatalf("expected date section to contain label and month/day/year fields\n%s", section)
	}
	if strings.Index(section, "Date") >= strings.Index(section, "Mar") {
		t.Fatalf("expected Date label to appear before the date controls\n%s", section)
	}
}

func TestRenderDialogDateSectionShowsTypedMonthDigitsWhileEditing(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 8, 7, 45, 0, 0, loc))
	dialog.startDate = time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	dialog.focusIndex = dialogFocusMonth
	dialog.numericInputBuffer = "12"

	section := ansi.Strip(renderDialogDateSection(dialog, 40))
	if !strings.Contains(section, "12") {
		t.Fatalf("expected month field to show typed digits while editing\n%s", section)
	}
	if strings.Contains(section, "Mar") {
		t.Fatalf("expected typed month digits to override the month code while editing\n%s", section)
	}
}

func TestRenderDialogScheduleSectionPlacesLabelsBesideInputs(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 8, 7, 45, 0, 0, loc))
	dialog.startDate = time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	dialog.hourInput.value = 7
	dialog.minuteInput.value = 45
	dialog.durationInput.value = 75

	section := ansi.Strip(renderDialogScheduleSection(dialog, 40))
	lines := strings.Split(section, "\n")
	if strings.Contains(section, dialogScheduleSummary(dialog)) {
		t.Fatalf("expected schedule section to omit the summary line\n%s", section)
	}
	var timeLine, durationLine string
	for _, line := range lines {
		if timeLine == "" && strings.Contains(line, "Time") && strings.Contains(line, "07") {
			timeLine = line
		}
		if durationLine == "" && strings.Contains(line, "Duration") && strings.Contains(line, "75m") {
			durationLine = line
		}
	}
	if timeLine == "" || durationLine == "" {
		t.Fatalf("expected schedule section to contain inline labels and values\n%s", section)
	}
	if strings.Index(timeLine, "Time") >= strings.Index(timeLine, "07") {
		t.Fatalf("expected Time label to appear before the time inputs\n%s", section)
	}
	if strings.Index(durationLine, "Duration") >= strings.Index(durationLine, "75m") {
		t.Fatalf("expected Duration label to appear before the duration input\n%s", section)
	}
}

func TestRenderDialogScheduleSectionKeepsThreeDigitDurationWithinWidth(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 8, 7, 45, 0, 0, loc))
	dialog.startDate = time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	dialog.durationInput.value = 120

	const width = 12
	section := ansi.Strip(renderDialogScheduleSection(dialog, width))
	if !strings.Contains(section, "Duration") || !strings.Contains(section, "120m") {
		t.Fatalf("expected duration section to render a three-digit duration without dropping content\n%s", section)
	}

	for _, line := range strings.Split(section, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("expected schedule line width <= %d, got %d: %q\n%s", width, got, line, section)
		}
	}
}

func TestRenderDialogSummarySectionCentersSummary(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Date(2026, 3, 8, 7, 45, 0, 0, loc))
	dialog.startDate = time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	dialog.hourInput.value = 7
	dialog.minuteInput.value = 45
	dialog.durationInput.value = 75

	section := ansi.Strip(renderDialogSummarySection(dialog, 40))
	summary := dialogScheduleSummary(dialog)
	if got := strings.TrimSpace(section); got != summary {
		t.Fatalf("expected summary section to render %q\n%s", summary, section)
	}
	leftPad := len(section) - len(strings.TrimLeft(section, " "))
	rightPad := len(section) - len(strings.TrimRight(section, " "))
	if leftPad == 0 || rightPad == 0 {
		t.Fatalf("expected summary section to be centered\n%s", section)
	}
	if diff := leftPad - rightPad; diff < -1 || diff > 1 {
		t.Fatalf("expected summary padding to be balanced, got left=%d right=%d\n%s", leftPad, rightPad, section)
	}
}

func TestRenderDialogButtonsPlacesActionsOnSameLine(t *testing.T) {
	dialog, _ := newCreateEventDialog(time.Now())

	section := ansi.Strip(renderDialogButtons(dialog, 40))
	lines := strings.Split(section, "\n")
	var actionLine string
	for _, line := range lines {
		if strings.Contains(line, "Create") && strings.Contains(line, "Cancel") {
			actionLine = line
			break
		}
	}
	if actionLine == "" {
		t.Fatalf("expected actions to contain Create and Cancel\n%s", section)
	}
	if strings.Index(actionLine, "Create") >= strings.Index(actionLine, "Cancel") {
		t.Fatalf("expected Create to appear before Cancel on the shared action row\n%s", section)
	}
}

func TestViewCompositesCreateEventDialogOverCalendar(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	dialog, _ := newCreateEventDialog(time.Now())

	view := ansi.Strip(model{
		width:  120,
		height: 40,
		data: calendarData{
			sections: buildDaySections(time.Date(2026, 3, 7, 15, 0, 0, 0, loc), []ical.Event{
				{
					Title:     "Standup",
					StartDate: time.Date(2026, 3, 7, 9, 0, 0, 0, loc),
					EndDate:   time.Date(2026, 3, 7, 10, 0, 0, 0, loc),
				},
			}),
		},
		showCreateDialog: true,
		createDialog:     dialog,
	}.View())

	for _, want := range []string{"Title", "Today", "Tomorrow"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected composited view to contain %q\n%s", want, view)
		}
	}
}

func TestViewRendersLoadingPlaceholder(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)
	view := ansi.Strip(model{
		loading: true,
		width:   120,
		data: calendarData{
			currentTime: now,
		},
	}.View())

	if !strings.Contains(view, "Today") || !strings.Contains(view, "9:15 AM") {
		t.Fatalf("expected loading view to render placeholder content\n%s", view)
	}
	if !strings.Contains(view, "All day") {
		t.Fatalf("expected loading view to reserve all-day buffer space\n%s", view)
	}
	if strings.Contains(view, "No calendar days available.") {
		t.Fatalf("expected loading view to avoid the empty-state fallback\n%s", view)
	}
	if strings.Contains(view, "┃") {
		t.Fatalf("expected loading view to avoid fake event placeholders\n%s", view)
	}
}

func TestRenderCalendarLayoutReservesAllDayBufferWithoutEvents(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 9, 15, 0, 0, loc)

	layout := ansi.Strip(renderCalendarLayout(calendarData{
		sections:    buildDaySections(now, nil),
		currentTime: now,
	}, 120))

	allDay := strings.Index(layout, "All day")
	firstTimeline := strings.Index(layout, "8:00 AM")
	if allDay == -1 || firstTimeline == -1 {
		t.Fatalf("expected reserved all-day buffer and timeline labels\n%s", layout)
	}
	if allDay > firstTimeline {
		t.Fatalf("expected all-day buffer to appear before the timeline\n%s", layout)
	}
}

func TestAllDaySectionLineCountReservesTwoRows(t *testing.T) {
	tests := []struct {
		name  string
		lines [][]string
		want  int
	}{
		{name: "no events", lines: [][]string{{}, {}, {}}, want: 2},
		{name: "single event", lines: [][]string{{"Holiday"}, {}, {}}, want: 2},
		{name: "multiple events", lines: [][]string{{"Holiday", "OOO"}, {}, {}}, want: 2},
		{name: "more than two events", lines: [][]string{{"Holiday", "OOO", "Travel"}, {}, {}}, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allDaySectionLineCount(tt.lines); got != tt.want {
				t.Fatalf("expected %d all-day rows, got %d", tt.want, got)
			}
		})
	}
}

func TestLoadingSlotWindowKeepsCurrentTimeVisible(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 22, 30, 0, 0, loc)

	windowStart, windowEnd := loadingSlotWindow(now)
	currentSlot, ok := currentTimeMarkerSlot(now, windowStart, windowEnd)
	if !ok {
		t.Fatalf("expected loading window %d-%d to include the current time", windowStart, windowEnd)
	}
	if currentSlot < windowStart || currentSlot > windowEnd {
		t.Fatalf("expected current slot %d to fall within %d-%d", currentSlot, windowStart, windowEnd)
	}
}

func TestDisplayEventTitleNormalizesEventTitles(t *testing.T) {
	tests := []struct {
		name  string
		event ical.Event
		want  string
	}{
		{
			name:  "non recurring title",
			event: ical.Event{Title: "Standup"},
			want:  "Standup",
		},
		{
			name:  "recurring title",
			event: ical.Event{Title: "Standup", Recurring: true},
			want:  "Standup",
		},
		{
			name:  "recurring untitled event",
			event: ical.Event{Recurring: true},
			want:  "(untitled event)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayEventTitle(tt.event); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestDisplayEventMarkerMarksRecurringEvents(t *testing.T) {
	tests := []struct {
		name  string
		event ical.Event
		want  string
	}{
		{
			name:  "non recurring title",
			event: ical.Event{Title: "Standup"},
			want:  "",
		},
		{
			name:  "recurring title",
			event: ical.Event{Title: "Standup", Recurring: true},
			want:  recurringMarker,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayEventMarker(tt.event); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func assertTitles(t *testing.T, events []ical.Event, want []string) {
	t.Helper()

	if len(events) != len(want) {
		t.Fatalf("expected %d events, got %d", len(want), len(events))
	}

	for i, title := range want {
		if events[i].Title != title {
			t.Fatalf("event %d: expected %q, got %q", i, title, events[i].Title)
		}
	}
}

func flattenLines(lines [][]string) string {
	var b strings.Builder
	for _, slotLines := range lines {
		for _, line := range slotLines {
			b.WriteString(line)
		}
	}
	return b.String()
}

func runeIndex(text, target string, occurrence int) int {
	if occurrence <= 0 {
		return -1
	}

	targetRune := []rune(target)
	if len(targetRune) != 1 {
		return -1
	}

	seen := 0
	for i, r := range []rune(text) {
		if r != targetRune[0] {
			continue
		}
		seen++
		if seen == occurrence {
			return i
		}
	}

	return -1
}

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location %q: %v", name, err)
	}
	return loc
}

func TestDayColumnAtX(t *testing.T) {
	widths := []int{34, 34, 34}
	cases := []struct {
		localX    int
		wantIndex int
		wantOK    bool
	}{
		{localX: -1, wantOK: false},
		{localX: 9, wantOK: false}, // time-axis gutter
		{localX: 10, wantIndex: 0, wantOK: true},
		{localX: 43, wantIndex: 0, wantOK: true},
		{localX: 44, wantOK: false}, // separator between day 0 and 1
		{localX: 46, wantIndex: 1, wantOK: true},
		{localX: 79, wantIndex: 1, wantOK: true},
		{localX: 82, wantIndex: 2, wantOK: true},
		{localX: 115, wantIndex: 2, wantOK: true},
		{localX: 116, wantOK: false}, // past the last column
	}

	for _, tc := range cases {
		index, ok := dayColumnAtX(tc.localX, widths)
		if ok != tc.wantOK {
			t.Fatalf("dayColumnAtX(%d) ok = %v, want %v", tc.localX, ok, tc.wantOK)
		}
		if ok && index != tc.wantIndex {
			t.Fatalf("dayColumnAtX(%d) index = %d, want %d", tc.localX, index, tc.wantIndex)
		}
	}
}

func TestCalendarViewGeometrySlotAt(t *testing.T) {
	geo := calendarViewGeometry{
		headerRows:    5,
		timedRowSlots: []int{0, 1, 2, 3, 4, 5, 6, 7},
		visibleStart:  2,
		visibleCount:  3,
	}

	cases := []struct {
		y        int
		wantSlot int
		wantOK   bool
	}{
		{y: 5, wantOK: false},             // above the timed rows
		{y: 6, wantSlot: 2, wantOK: true}, // first visible row -> visibleStart
		{y: 8, wantSlot: 4, wantOK: true}, // last visible row
		{y: 9, wantOK: false},             // beyond the viewport
	}

	for _, tc := range cases {
		slot, ok := geo.slotAt(tc.y)
		if ok != tc.wantOK {
			t.Fatalf("slotAt(%d) ok = %v, want %v", tc.y, ok, tc.wantOK)
		}
		if ok && slot != tc.wantSlot {
			t.Fatalf("slotAt(%d) slot = %d, want %d", tc.y, slot, tc.wantSlot)
		}
	}
}

func TestModelWithSelectionPreviewsDragEvent(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	day := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	data := calendarData{sections: buildDaySections(day, nil)}

	preview := data.withSelection(day, 18, 20) // 9:00 AM through 10:30 AM

	if len(data.sections[1].events) != 0 {
		t.Fatalf("withSelection mutated the original data: %d events", len(data.sections[1].events))
	}

	todayEvents := preview.sections[1].events
	if len(todayEvents) != 1 {
		t.Fatalf("expected 1 provisional event on today, got %d", len(todayEvents))
	}

	event := todayEvents[0]
	wantStart := time.Date(2026, 3, 7, 9, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 3, 7, 10, 30, 0, 0, loc)
	if !event.StartDate.Equal(wantStart) || !event.EndDate.Equal(wantEnd) {
		t.Fatalf("provisional event = %s-%s, want %s-%s", event.StartDate, event.EndDate, wantStart, wantEnd)
	}

	for _, i := range []int{0, 2} {
		if len(preview.sections[i].events) != 0 {
			t.Fatalf("expected no provisional event on section %d", i)
		}
	}
}

func TestMouseWheelScrollsTimeline(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 15, 0, 0, 0, loc)
	m := model{
		width:             120,
		height:            40,
		timedScrollOffset: 5,
		data: calendarData{
			sections:    buildDaySections(now, nil),
			currentTime: now,
		},
	}

	updated, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	if got := updated.(model).timedScrollOffset; got != 4 {
		t.Fatalf("wheel up offset = %d, want 4", got)
	}

	updated, _ = updated.(model).Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	if got := updated.(model).timedScrollOffset; got != 5 {
		t.Fatalf("wheel down offset = %d, want 5", got)
	}
}

func TestMouseIgnoredWhileDialogOpen(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 15, 0, 0, 0, loc)
	m := model{
		width:            120,
		height:           40,
		showCreateDialog: true,
		data: calendarData{
			sections:    buildDaySections(now, nil),
			currentTime: now,
		},
	}

	updated, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 50, Y: 8})
	if updated.(model).dragging {
		t.Fatal("expected mouse press to be ignored while the create dialog is open")
	}
}

func TestMouseDragOpensPrefilledDialog(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 15, 0, 0, 0, loc)
	m := model{
		width:  120,
		height: 40,
		data: calendarData{
			sections:    buildDaySections(now, nil),
			currentTime: now,
		},
	}

	lines := strings.Split(m.View(), "\n")
	startY := lineIndexContaining(t, lines, "10:00 AM")
	endY := lineIndexContaining(t, lines, "11:00 AM")
	headerY := lineIndexContaining(t, lines, "Mar 7")
	x := columnContaining(t, lines[headerY], "Mar 7")

	updated, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: x, Y: startY})
	dragged := updated.(model)
	if !dragged.dragging {
		t.Fatal("expected mouse press to begin a drag")
	}

	updated, _ = dragged.Update(tea.MouseMsg{Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft, X: x, Y: endY})
	updated, cmd := updated.(model).Update(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonNone, X: x, Y: endY})
	final := updated.(model)

	if final.dragging {
		t.Fatal("expected drag to end on mouse release")
	}
	if !final.showCreateDialog {
		t.Fatal("expected the create dialog to open after a drag")
	}
	if cmd == nil {
		t.Fatal("expected a command from opening the dialog")
	}

	if got := final.createDialog.startDate; !beginningOfDay(got).Equal(time.Date(2026, 3, 7, 0, 0, 0, 0, loc)) {
		t.Fatalf("dialog start day = %s, want 2026-03-07", got)
	}
	if got := final.createDialog.hourInput.value; got != 10 {
		t.Fatalf("dialog start hour = %d, want 10", got)
	}
	if got := final.createDialog.minuteInput.value; got != 0 {
		t.Fatalf("dialog start minute = %d, want 0", got)
	}
	if got := final.createDialog.durationInput.value; got != 90 {
		t.Fatalf("dialog duration = %d, want 90", got)
	}
}

func TestMouseDragUpwardNormalizesSelection(t *testing.T) {
	loc := time.FixedZone("test", -8*60*60)
	now := time.Date(2026, 3, 7, 15, 0, 0, 0, loc)
	m := model{
		width:  120,
		height: 40,
		data: calendarData{
			sections:    buildDaySections(now, nil),
			currentTime: now,
		},
	}

	lines := strings.Split(m.View(), "\n")
	lowY := lineIndexContaining(t, lines, " 2:00 PM")
	highY := lineIndexContaining(t, lines, " 1:00 PM")
	headerY := lineIndexContaining(t, lines, "Mar 8")
	x := columnContaining(t, lines[headerY], "Mar 8")

	updated, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: x, Y: lowY})
	updated, _ = updated.(model).Update(tea.MouseMsg{Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft, X: x, Y: highY})
	updated, _ = updated.(model).Update(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonNone, X: x, Y: highY})
	final := updated.(model)

	if got := final.createDialog.hourInput.value; got != 13 {
		t.Fatalf("dialog start hour = %d, want 13 (1:00 PM)", got)
	}
	if got := final.createDialog.durationInput.value; got != 90 {
		t.Fatalf("dialog duration = %d, want 90", got)
	}
	if !beginningOfDay(final.createDialog.startDate).Equal(time.Date(2026, 3, 8, 0, 0, 0, 0, loc)) {
		t.Fatalf("dialog start day = %s, want 2026-03-08 (Tomorrow column)", final.createDialog.startDate)
	}
}

func lineIndexContaining(t *testing.T, lines []string, sub string) int {
	t.Helper()
	for i, line := range lines {
		if strings.Contains(ansi.Strip(line), sub) {
			return i
		}
	}
	t.Fatalf("no rendered line contains %q", sub)
	return -1
}

func columnContaining(t *testing.T, line, sub string) int {
	t.Helper()
	plain := ansi.Strip(line)
	idx := strings.Index(plain, sub)
	if idx < 0 {
		t.Fatalf("line %q does not contain %q", plain, sub)
	}
	return len([]rune(plain[:idx]))
}
