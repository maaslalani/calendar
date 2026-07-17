package calendar

import (
	"fmt"
	"strings"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type calendarRenderSections struct {
	headerRows    []string
	timedRows     []string
	timedRowSlots []int
	footerRows    []string
}

func renderLoadingCalendarLayout(viewDay, now time.Time, terminalWidth, terminalHeight, scrollOffset int) string {
	return renderCalendarRenderSections(buildLoadingCalendarRenderSections(viewDay, now, terminalWidth), terminalHeight, scrollOffset)
}

func maxLoadingCalendarScroll(viewDay, now time.Time, terminalWidth, terminalHeight int) int {
	return maxCalendarRenderScrollOffset(buildLoadingCalendarRenderSections(viewDay, now, terminalWidth), terminalHeight)
}

func buildLoadingCalendarRenderSections(viewDay, now time.Time, terminalWidth int) calendarRenderSections {
	if now.IsZero() {
		now = time.Now()
	}
	if viewDay.IsZero() {
		viewDay = beginningOfDay(now)
	} else {
		viewDay = beginningOfDay(viewDay)
	}

	sections := buildLoadingSections(viewDay)
	layout := newCalendarLayout(len(sections), terminalWidth)
	allDayLines := make([][]string, len(sections))
	windowStart, windowEnd := 0, slotsPerDay
	headerRows := renderSectionHeaderRows(sections, now, layout)
	headerRows = appendAllDaySectionRows(headerRows, allDayLines, layout)
	timedLines := make([][]string, len(sections))
	for i := range timedLines {
		timedLines[i] = make([]string, windowEnd-windowStart)
	}

	timedRows, timedRowSlots := renderTimedWindowRows(sections, now, nil, layout, windowStart, windowEnd, timedLines)
	return calendarRenderSections{
		headerRows:    headerRows,
		timedRows:     timedRows,
		timedRowSlots: timedRowSlots,
	}
}

func buildLoadingSections(viewDay time.Time) []daySection {
	return buildDaySections(beginningOfDay(viewDay), nil)
}

func loadingStartSlot(now time.Time) int {
	windowSize := defaultWindowTo - defaultWindowFrom
	if windowSize <= 0 {
		return defaultWindowFrom
	}

	currentSlot, showCurrentLine := currentTimeMarkerSlot(now, 0, slotsPerDay)
	if !showCurrentLine || (defaultWindowFrom <= currentSlot && currentSlot <= defaultWindowTo) {
		return defaultWindowFrom
	}

	windowStart := currentSlot - windowSize/2
	windowEnd := windowStart + windowSize
	if windowStart < 0 {
		windowStart = 0
		windowEnd = min(slotsPerDay, windowSize)
	}
	if windowEnd > slotsPerDay {
		windowEnd = slotsPerDay
		windowStart = max(0, windowEnd-windowSize)
	}

	return windowStart
}

func renderCalendarLayout(data calendarData, terminalWidth, terminalHeight, scrollOffset int) string {
	if len(data.sections) == 0 {
		return "No calendar days available."
	}

	return renderCalendarRenderSections(buildCalendarRenderSections(data, terminalWidth), terminalHeight, scrollOffset)
}

func maxCalendarLayoutScroll(data calendarData, terminalWidth, terminalHeight int) int {
	if len(data.sections) == 0 {
		return 0
	}
	return maxCalendarRenderScrollOffset(buildCalendarRenderSections(data, terminalWidth), terminalHeight)
}

func buildCalendarRenderSections(data calendarData, terminalWidth int) calendarRenderSections {
	layout := newCalendarLayout(len(data.sections), terminalWidth)
	allDayLines := make([][]string, len(data.sections))
	timedEvents := make([][]ical.Event, len(data.sections))

	for i, section := range data.sections {
		allDayEvents, sectionTimedEvents := partitionDayEvents(section.events)
		allDayLines[i] = renderAllDayLines(allDayEvents, data.calendarColors, layout.sectionWidths[i])
		timedEvents[i] = sectionTimedEvents
	}
	windowStart, windowEnd := 0, slotsPerDay
	timedLines := make([][]string, len(data.sections))
	for i, section := range data.sections {
		timedLines[i] = renderTimedLines(section.date, timedEvents[i], data.calendarColors, windowStart, windowEnd, layout.sectionWidths[i], data.currentTime)
	}

	headerRows := renderSectionHeaderRows(data.sections, data.currentTime, layout)
	headerRows = appendAllDaySectionRows(headerRows, allDayLines, layout)
	footerRows := renderLegendRows(data.legend)

	timedRows, timedRowSlots := renderTimedWindowRows(data.sections, data.currentTime, data.calendarColors, layout, windowStart, windowEnd, timedLines)
	return calendarRenderSections{
		headerRows:    headerRows,
		timedRows:     timedRows,
		timedRowSlots: timedRowSlots,
		footerRows:    footerRows,
	}
}

func renderCalendarRenderSections(sections calendarRenderSections, terminalHeight, scrollOffset int) string {
	rows := make([]string, 0, len(sections.headerRows)+len(sections.timedRows)+len(sections.footerRows))
	rows = append(rows, sections.headerRows...)
	rows = append(rows, visibleTimedRows(sections, terminalHeight, scrollOffset)...)
	rows = append(rows, sections.footerRows...)
	return strings.Join(rows, "\n")
}

func visibleTimedRows(sections calendarRenderSections, terminalHeight, scrollOffset int) []string {
	if terminalHeight <= 0 {
		return sections.timedRows
	}

	start, count := visibleTimedRowRange(sections, terminalHeight, scrollOffset)
	if count <= 0 {
		return nil
	}
	if start == 0 && count >= len(sections.timedRows) {
		return sections.timedRows
	}
	return sections.timedRows[start : start+count]
}

// visibleTimedRowRange reports the index of the first visible timed row and the
// number of timed rows on screen for the given viewport height and scroll
// offset. It mirrors the slicing performed by visibleTimedRows so callers can
// map screen coordinates back to timed rows.
func visibleTimedRowRange(sections calendarRenderSections, terminalHeight, scrollOffset int) (int, int) {
	if terminalHeight <= 0 {
		return 0, len(sections.timedRows)
	}

	viewportRows := timedViewportRowCount(sections, terminalHeight)
	if viewportRows <= 0 {
		return 0, 0
	}
	if viewportRows >= len(sections.timedRows) {
		return 0, len(sections.timedRows)
	}

	start := min(max(0, scrollOffset), maxCalendarRenderScrollOffset(sections, terminalHeight))
	return start, viewportRows
}

func timedViewportRowCount(sections calendarRenderSections, terminalHeight int) int {
	if terminalHeight <= 0 {
		return len(sections.timedRows)
	}

	availableRows := terminalHeight - len(sections.headerRows) - len(sections.footerRows)
	if availableRows < 0 {
		return 0
	}
	return availableRows
}

func maxCalendarRenderScrollOffset(sections calendarRenderSections, terminalHeight int) int {
	viewportRows := timedViewportRowCount(sections, terminalHeight)
	if viewportRows <= 0 {
		return 0
	}
	if viewportRows >= len(sections.timedRows) {
		return 0
	}
	return len(sections.timedRows) - viewportRows
}

func renderTimedWindowRows(sections []daySection, now time.Time, calendarColors map[string]string, layout calendarLayout, windowStart, windowEnd int, timedLines [][]string) ([]string, []int) {
	rows := make([]string, 0, windowEnd-windowStart+1)
	slots := make([]int, 0, windowEnd-windowStart+1)
	currentLineSlot, showCurrentLine := currentTimeMarkerSlot(now, windowStart, windowEnd)
	for slot := windowStart; slot < windowEnd; slot++ {
		if showCurrentLine && currentLineSlot == slot {
			rows = append(rows, renderCurrentTimeRow(now, sections, calendarColors, layout, windowStart, windowEnd))
			slots = append(slots, slot)
		}

		cells := make([]string, len(sections))
		for i := range sections {
			cells[i] = timedLines[i][slot-windowStart]
		}

		labelStyle := timeAxisStyle
		if slotHasVisibleEventBoundary(sections, slot) {
			labelStyle = activeTimeAxisStyle
		}
		rows = append(rows, renderRow(timelineLabel(slot), cells, labelStyle, dayCellStyle, layout))
		slots = append(slots, slot)
	}

	if showCurrentLine && currentLineSlot == windowEnd {
		rows = append(rows, renderCurrentTimeRow(now, sections, calendarColors, layout, windowStart, windowEnd))
		slots = append(slots, min(windowEnd, slotsPerDay-1))
	}

	return rows, slots
}

func renderLegendRows(items []calendarLegendItem) []string {
	if legend := renderLegend(items); legend != "" {
		return []string{"", legend}
	}
	return nil
}

func appendAllDaySectionRows(rows []string, allDayLines [][]string, layout calendarLayout) []string {
	if len(allDayLines) == 0 {
		return rows
	}

	rowCount := allDayRegionRowCount(allDayLines)
	for lineIndex := 0; lineIndex < rowCount; lineIndex++ {
		cells := make([]string, len(allDayLines))
		for i := range allDayLines {
			if lineIndex < len(allDayLines[i]) {
				cells[i] = allDayLines[i][lineIndex]
			}
		}

		label := ""
		if lineIndex == 0 {
			label = "All day"
		}

		rows = append(rows, renderRow(label, cells, timeAxisStyle, dayCellStyle, layout))
	}

	return rows
}

func allDaySectionLineCount(allDayLines [][]string) int {
	lineCount := minAllDayBufferRows
	for _, lines := range allDayLines {
		lineCount = max(lineCount, len(lines))
	}
	return lineCount
}

// allDayRegionRowCount returns the fixed number of rows reserved for the
// all-day region. Event lines fill from the top and any remaining rows act as
// the buffer before the timeline. Keeping this constant stops the timeline from
// shifting as the all-day event count changes.
func allDayRegionRowCount(allDayLines [][]string) int {
	return max(minAllDayBufferRows+1, allDaySectionLineCount(allDayLines))
}

func partitionDayEvents(events []ical.Event) ([]ical.Event, []ical.Event) {
	var allDay, timed []ical.Event
	for _, event := range events {
		if event.AllDay {
			allDay = append(allDay, event)
			continue
		}
		timed = append(timed, event)
	}

	return allDay, timed
}

func sectionLabels(sections []daySection, reference time.Time) []string {
	labels := make([]string, len(sections))
	for i, section := range sections {
		if reference.IsZero() {
			labels[i] = section.label
			continue
		}
		labels[i] = relativeDayLabel(section.date, reference)
	}
	return labels
}

func relativeDayLabel(day, reference time.Time) string {
	day = beginningOfDay(day)
	reference = beginningOfDay(reference.In(day.Location()))
	diff := localDateOrdinal(day) - localDateOrdinal(reference)

	switch diff {
	case -1:
		return "Yesterday"
	case 0:
		return "Today"
	case 1:
		return "Tomorrow"
	}

	distance := diff
	if distance < 0 {
		distance = -distance
	}

	value := distance
	unit := "day"
	switch {
	case distance >= 365:
		value = (distance + 182) / 365
		unit = "year"
	case distance >= 30:
		value = (distance + 15) / 30
		unit = "month"
	}
	if value != 1 {
		unit += "s"
	}

	if diff < 0 {
		return fmt.Sprintf("%d %s ago", value, unit)
	}
	return fmt.Sprintf("In %d %s", value, unit)
}

func localDateOrdinal(t time.Time) int {
	year, month, day := t.Date()
	return int(time.Date(year, month, day, 12, 0, 0, 0, time.UTC).Unix() / int64(24*time.Hour/time.Second))
}

func sectionDates(sections []daySection) []string {
	dates := make([]string, len(sections))
	for i, section := range sections {
		dates[i] = section.date.Format("Mon Jan 2")
	}
	return dates
}

func renderSectionHeaderRows(sections []daySection, reference time.Time, layout calendarLayout) []string {
	dateStyles := make([]lipgloss.Style, len(sections))
	labelStyles := make([]lipgloss.Style, len(sections))
	for i, section := range sections {
		dateStyles[i] = sectionHeaderCellStyle(section, reference, headerCellStyle)
		labelStyles[i] = sectionHeaderCellStyle(section, reference, dateCellStyle)
	}

	return []string{
		renderStyledRow("", sectionDates(sections), timeAxisStyle, dateStyles, layout),
		renderStyledRow("", sectionLabels(sections, reference), timeAxisStyle, labelStyles, layout),
	}
}

func sectionHeaderCellStyle(section daySection, reference time.Time, baseStyle lipgloss.Style) lipgloss.Style {
	if isTodaySection(section, reference) {
		return baseStyle.Foreground(currentTimeColor)
	}
	return baseStyle
}

func isTodaySection(section daySection, reference time.Time) bool {
	if !reference.IsZero() {
		return beginningOfDay(section.date).Equal(beginningOfDay(reference.In(section.date.Location())))
	}
	return strings.TrimSpace(section.label) == "Today"
}

func newCalendarLayout(sectionCount, terminalWidth int) calendarLayout {
	separatorWidth := lipgloss.Width(columnSeparator)
	defaultWidth := timeColumnWidth + sectionCount*dayColumnWidth + sectionCount*separatorWidth
	if terminalWidth <= 0 {
		terminalWidth = defaultWidth
	}

	availableWidth := max(sectionCount, terminalWidth-timeColumnWidth-sectionCount*separatorWidth)

	return calendarLayout{
		sectionWidths: splitWidths(availableWidth, sectionCount),
		separator:     columnSeparator,
	}
}

func renderRow(timeLabel string, cells []string, timeStyle, cellStyle lipgloss.Style, layout calendarLayout) string {
	cellStyles := make([]lipgloss.Style, len(cells))
	for i := range cellStyles {
		cellStyles[i] = cellStyle
	}
	return renderStyledRow(timeLabel, cells, timeStyle, cellStyles, layout)
}

func renderStyledRow(timeLabel string, cells []string, timeStyle lipgloss.Style, cellStyles []lipgloss.Style, layout calendarLayout) string {
	parts := make([]string, 0, len(cells)+1)
	parts = append(parts, timeStyle.Render(timeLabel))

	for i, cell := range cells {
		style := lipgloss.NewStyle()
		if i < len(cellStyles) {
			style = cellStyles[i]
		}
		parts = append(parts, style.Width(layout.sectionWidths[i]).Render(cell))
	}

	return strings.Join(parts, layout.separator)
}

func renderCurrentTimeRow(now time.Time, sections []daySection, calendarColors map[string]string, layout calendarLayout, windowStart, windowEnd int) string {
	var b strings.Builder
	b.WriteString(currentTimeAxisStyle.Render(now.Format("3:04 PM")))
	b.WriteString(layout.separator)

	connectorWidth := lipgloss.Width(layout.separator)
	for i, section := range sections {
		b.WriteString(renderCurrentTimeSectionLine(section, now, calendarColors, layout.sectionWidths[i], windowStart, windowEnd))
		if i == len(sections)-1 {
			continue
		}

		connectorStyle := currentTimeConnectorStyle(section, sections[i+1], now)
		b.WriteString(connectorStyle.Render(strings.Repeat("─", connectorWidth)))
	}

	return b.String()
}

func renderCurrentTimeSectionLine(section daySection, now time.Time, calendarColors map[string]string, width, windowStart, windowEnd int) string {
	lineStyle := currentTimeStyleForSection(section, now)
	line := strings.Repeat("─", width)
	if width <= 0 {
		return lineStyle.Render(line)
	}

	_, timedEvents := partitionDayEvents(section.events)
	blocks := buildTimedEventBlocks(section.date, timedEvents)
	if len(blocks) == 0 {
		return lineStyle.Render(line)
	}

	titleMarker := newTimedTitleMarker(section.date, now, windowStart, windowEnd)
	active := activeBlocksAtTime(section.date, blocks, wallClockTimeOnDay(section.date, now))
	if len(active) == 0 {
		return lineStyle.Render(line)
	}

	layouts := buildTimedEventLayouts(blocks, calendarColors, width)
	return renderCurrentTimeActiveLine(active, calendarColors, layouts[active[0].cluster], lineStyle, titleMarker, windowStart, windowEnd)
}

func renderCurrentTimeActiveLine(active []*timedEventBlock, calendarColors map[string]string, layout timedEventLayout, lineStyle lipgloss.Style, titleMarker timedTitleMarker, windowStart, windowEnd int) string {
	if len(layout.columnWidths) == 0 {
		return ""
	}

	lineParts := make([]string, len(layout.columnWidths))
	for layer, width := range layout.columnWidths {
		lineParts[layer] = lineStyle.Render(strings.Repeat("─", width))
	}

	for _, block := range active {
		color := eventCalendarColor(block.event, calendarColors)
		backgroundColor := eventBackgroundColor(color)
		width := layout.columnWidths[block.layer]
		if width <= 0 {
			continue
		}
		prefixWidth := lipgloss.Width(renderEventAccent(block.accent, color, backgroundColor))
		title := timedBlockMarkerTitleLine(block, windowStart, windowEnd, prefixWidth, width, titleMarker)
		lineParts[block.layer] = renderCurrentTimeEventLine(title, block.accent, color, backgroundColor, width, lineStyle)
	}

	separator := lineStyle.Render(strings.Repeat("─", lipgloss.Width(layout.separator)))
	return strings.Join(lineParts, separator)
}

func renderCurrentTimeEventLine(title, accent, color, backgroundColor string, width int, lineStyle lipgloss.Style) string {
	if width <= 0 {
		return ""
	}

	junction := currentTimeEventJunction(accent)
	junctionWidth := lipgloss.Width(junction)
	prefixWidth := min(width, lipgloss.Width(accent))
	leadingWidth := max(0, prefixWidth-junctionWidth)
	textWidth := max(0, width-junctionWidth-leadingWidth)
	title = ansi.Truncate(title, textWidth, "…")
	trailingWidth := max(0, textWidth-lipgloss.Width(title))
	lineStyle = lineStyle.Background(lipgloss.Color(backgroundColor))

	var b strings.Builder
	b.WriteString(colorStyle(color).Background(lipgloss.Color(backgroundColor)).Render(junction))
	b.WriteString(lineStyle.Render(strings.Repeat("─", leadingWidth)))
	if title != "" {
		b.WriteString(eventBackgroundStyle(backgroundColor).Render(title))
	}
	b.WriteString(lineStyle.Render(strings.Repeat("─", trailingWidth)))
	return b.String()
}

func currentTimeEventJunction(accent string) string {
	if accent == eventAccent {
		return "╂"
	}
	return "┼"
}

func currentTimeStyleForSection(section daySection, now time.Time) lipgloss.Style {
	if isCurrentDaySection(section, now) {
		return currentTimeLineStyle
	}
	return currentTimeEdgeStyle
}

func currentTimeConnectorStyle(left, right daySection, now time.Time) lipgloss.Style {
	if isCurrentDaySection(left, now) || isCurrentDaySection(right, now) {
		return currentTimeLineStyle
	}
	return currentTimeEdgeStyle
}

func isCurrentDaySection(section daySection, now time.Time) bool {
	return beginningOfDay(section.date).Equal(beginningOfDay(now))
}

func currentTimeMarkerSlot(now time.Time, windowStart, windowEnd int) (int, bool) {
	if now.IsZero() {
		return 0, false
	}

	dayStart := beginningOfDay(now)
	offset := wallClockOffset(dayStart, now)
	visibleStart := time.Duration(windowStart) * slotDuration
	visibleEnd := time.Duration(windowEnd) * slotDuration
	if offset < visibleStart || offset > visibleEnd {
		return 0, false
	}

	slot := wallClockSlot(dayStart, now, true)
	if slot < windowStart || slot > windowEnd {
		return 0, false
	}

	return slot, true
}

func slotHasVisibleEventBoundary(sections []daySection, slot int) bool {
	for _, section := range sections {
		if slotHasEventBoundary(section.date, section.events, slot) {
			return true
		}
	}
	return false
}

func slotHasEventBoundary(day time.Time, events []ical.Event, slot int) bool {
	for _, event := range events {
		if event.AllDay {
			continue
		}
		if timeFallsOnSlotBoundary(day, event.StartDate, slot) || timeFallsOnSlotBoundary(day, event.EndDate, slot) {
			return true
		}
	}
	return false
}

func renderAllDayLines(events []ical.Event, calendarColors map[string]string, width int) []string {
	visibleCount := len(events)
	overflow := false
	if visibleCount > maxAllDayEvents {
		// Reserve the last row for a "+N more" indicator rather than dropping
		// the overflowing events silently.
		visibleCount = maxAllDayEvents - 1
		overflow = true
	}

	lines := make([]string, 0, maxAllDayEvents)
	for _, event := range events[:visibleCount] {
		lines = append(lines, renderEventSummaryLine(displayEventTitle(event), eventCalendarColor(event, calendarColors), width))
	}

	if overflow {
		lines = append(lines, renderAllDayMoreLine(len(events)-visibleCount, width))
	}

	return lines
}
