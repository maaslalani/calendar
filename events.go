package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func renderTimedLines(day time.Time, events []ical.Event, calendarColors map[string]string, windowStart, windowEnd, dayWidth int) [][]string {
	return renderTimedLinesAt(day, events, calendarColors, windowStart, windowEnd, dayWidth, time.Time{})
}

func renderTimedLinesAt(day time.Time, events []ical.Event, calendarColors map[string]string, windowStart, windowEnd, dayWidth int, now time.Time) [][]string {
	blocks := buildTimedEventBlocks(day, events)
	layouts := buildTimedEventLayouts(blocks, calendarColors, dayWidth)
	blocksBySlot := indexTimedEventBlocks(blocks, windowStart, windowEnd)
	titleMarker := newTimedTitleMarker(day, now, windowStart, windowEnd)

	lines := make([][]string, len(blocksBySlot))
	for i, active := range blocksBySlot {
		if len(active) == 0 {
			continue
		}

		slot := windowStart + i
		lines[i] = renderTimedBlockLinesWithMarker(active, slot, windowStart, windowEnd, calendarColors, layouts[active[0].cluster], titleMarker)
	}

	return lines
}

func buildTimedEventBlocks(day time.Time, events []ical.Event) []timedEventBlock {
	blocks := make([]timedEventBlock, 0, len(events))
	for _, event := range events {
		startSlot, endSlot, ok := eventSlotRange(day, event)
		if !ok {
			continue
		}
		blocks = append(blocks, timedEventBlock{
			event:     event,
			startSlot: startSlot,
			endSlot:   endSlot,
			accent:    timedEventAccent(day, event),
		})
	}

	sort.SliceStable(blocks, func(i, j int) bool {
		if blocks[i].startSlot != blocks[j].startSlot {
			return blocks[i].startSlot < blocks[j].startSlot
		}
		if blocks[i].endSlot != blocks[j].endSlot {
			return blocks[i].endSlot < blocks[j].endSlot
		}
		return displayEventTitle(blocks[i].event) < displayEventTitle(blocks[j].event)
	})
	assignTimedEventClusters(blocks)
	assignTimedEventLayers(blocks)

	return blocks
}

func indexTimedEventBlocks(blocks []timedEventBlock, windowStart, windowEnd int) [][]*timedEventBlock {
	blocksBySlot := make([][]*timedEventBlock, max(0, windowEnd-windowStart))
	ordered := make([]*timedEventBlock, len(blocks))
	for i := range blocks {
		ordered[i] = &blocks[i]
	}
	sortTimedEventBlockPointersByLayer(ordered)

	slotCounts := make([]int, len(blocksBySlot))
	for _, block := range ordered {
		start := max(block.startSlot, windowStart)
		end := min(block.endSlot, windowEnd)
		for slot := start; slot < end; slot++ {
			slotCounts[slot-windowStart]++
		}
	}
	for i, count := range slotCounts {
		if count > 0 {
			blocksBySlot[i] = make([]*timedEventBlock, 0, count)
		}
	}

	for _, block := range ordered {
		start := max(block.startSlot, windowStart)
		end := min(block.endSlot, windowEnd)
		for slot := start; slot < end; slot++ {
			blocksBySlot[slot-windowStart] = append(blocksBySlot[slot-windowStart], block)
		}
	}

	return blocksBySlot
}

func sortTimedEventBlockPointersByLayer(blocks []*timedEventBlock) {
	sort.SliceStable(blocks, func(i, j int) bool {
		if blocks[i].layer != blocks[j].layer {
			return blocks[i].layer < blocks[j].layer
		}
		if blocks[i].startSlot != blocks[j].startSlot {
			return blocks[i].startSlot < blocks[j].startSlot
		}
		return displayEventTitle(blocks[i].event) < displayEventTitle(blocks[j].event)
	})
}

func activeBlocksAtTime(day time.Time, blocks []timedEventBlock, now time.Time) []*timedEventBlock {
	active := make([]*timedEventBlock, 0, len(blocks))
	dayStart := beginningOfDay(day)
	dayEnd := dayStart.AddDate(0, 0, 1)

	for i := range blocks {
		block := &blocks[i]
		start := maxTime(block.event.StartDate, dayStart)
		end := minTime(block.event.EndDate, dayEnd)
		if !end.After(start) {
			continue
		}
		if now.Before(start) || !now.Before(end) {
			continue
		}
		active = append(active, block)
	}

	sortTimedEventBlockPointersByLayer(active)
	return active
}

func renderTimedBlockLines(active []*timedEventBlock, slot, windowStart, windowEnd int, calendarColors map[string]string, layout timedEventLayout) []string {
	return renderTimedBlockLinesWithMarker(active, slot, windowStart, windowEnd, calendarColors, layout, timedTitleMarker{})
}

func renderTimedBlockLinesWithMarker(active []*timedEventBlock, slot, windowStart, windowEnd int, calendarColors map[string]string, layout timedEventLayout, titleMarker timedTitleMarker) []string {
	if len(active) == 0 {
		return nil
	}

	lineParts := make([]string, len(layout.columnWidths))
	for layer, width := range layout.columnWidths {
		lineParts[layer] = strings.Repeat(" ", width)
	}

	for _, block := range active {
		color := eventCalendarColor(block.event, calendarColors)
		accents := []eventAccentSegment{{accent: block.accent, color: color}}
		backgroundColor := eventBackgroundColor(color)
		width := layout.columnWidths[block.layer]

		marker := ""
		if slot == max(block.startSlot, windowStart) {
			marker = displayEventMarker(block.event)
		}

		title := timedBlockTitleLineWithMarker(block, slot, windowStart, windowEnd, accents, backgroundColor, width, titleMarker)
		lineParts[block.layer] = renderEventLine(title, marker, color, accents, width, backgroundColor)
	}

	return []string{strings.Join(lineParts, layout.separator)}
}

func timedBlockTitleLine(block *timedEventBlock, slot, windowStart, windowEnd int, accents []eventAccentSegment, backgroundColor string, width int) string {
	return timedBlockTitleLineWithMarker(block, slot, windowStart, windowEnd, accents, backgroundColor, width, timedTitleMarker{})
}

func timedBlockTitleLineWithMarker(block *timedEventBlock, slot, windowStart, windowEnd int, accents []eventAccentSegment, backgroundColor string, width int, titleMarker timedTitleMarker) string {
	if block.startSlot < windowStart {
		return ""
	}

	lineIndex := slot - block.startSlot
	if titleMarker.carriesTitle(block) {
		if slot >= titleMarker.slot {
			lineIndex++
		}
	}

	lines := timedBlockTitleLines(block, windowEnd, accents, backgroundColor, width)
	if lineIndex < 0 || lineIndex >= len(lines) {
		return ""
	}
	return lines[lineIndex]
}

func timedBlockMarkerTitleLine(block *timedEventBlock, windowStart, windowEnd int, accents []eventAccentSegment, backgroundColor string, width int, titleMarker timedTitleMarker) string {
	if block.startSlot < windowStart || !titleMarker.carriesTitle(block) {
		return ""
	}

	lines := timedBlockTitleLines(block, windowEnd, accents, backgroundColor, width)
	lineIndex := titleMarker.slot - block.startSlot
	if lineIndex < 0 || lineIndex >= len(lines) {
		return ""
	}
	return lines[lineIndex]
}

func timedBlockTitleLines(block *timedEventBlock, windowEnd int, accents []eventAccentSegment, backgroundColor string, width int) []string {
	prefixWidth := lipgloss.Width(renderAccentPrefix(accents, backgroundColor))
	wrapWidth := width - prefixWidth - eventMarkerWidth(displayEventMarker(block.event)) - eventTitleRightPad
	maxLines := min(block.endSlot, windowEnd) - block.startSlot
	return wrapEventTitle(displayEventTitle(block.event), wrapWidth, maxLines)
}

type timedTitleMarker struct {
	slot    int
	at      time.Time
	visible bool
}

func newTimedTitleMarker(day, now time.Time, windowStart, windowEnd int) timedTitleMarker {
	slot, visible := currentTimeMarkerSlot(now, windowStart, windowEnd)
	if !visible {
		return timedTitleMarker{}
	}

	return timedTitleMarker{
		slot:    slot,
		at:      wallClockTimeOnDay(day, now),
		visible: true,
	}
}

func (marker timedTitleMarker) carriesTitle(block *timedEventBlock) bool {
	return marker.visible &&
		marker.slot > block.startSlot &&
		marker.at.After(block.event.StartDate) &&
		marker.at.Before(block.event.EndDate)
}

func wrapEventTitle(title string, width, maxLines int) []string {
	if width <= 0 || maxLines <= 0 {
		return nil
	}
	if maxLines == 1 {
		return []string{title}
	}

	lines := strings.Split(ansi.Wrap(title, width, ""), "\n")
	if len(lines) <= maxLines {
		return lines
	}

	lines[maxLines-1] = strings.Join(lines[maxLines-1:], " ")
	return lines[:maxLines]
}

func renderEventSummaryLine(title, marker, color string, width int) string {
	if width <= 0 {
		return ""
	}

	return eventForegroundStyle(color).Render(renderEventBody(title, marker, color, "", width, true))
}

func renderAllDayMoreLine(hidden, width int) string {
	if width <= 0 {
		return ""
	}

	return allDayMoreStyle.Render(renderEventBody(fmt.Sprintf("+%d more", hidden), "", "", "", width, true))
}

func visibleSlotWindow(sections []daySection) (int, int) {
	minSlot := slotsPerDay
	maxSlot := 0

	for _, section := range sections {
		for _, event := range section.events {
			if event.AllDay {
				continue
			}

			startSlot, endSlot, ok := eventSlotRange(section.date, event)
			if !ok {
				continue
			}

			minSlot = min(minSlot, startSlot)
			maxSlot = max(maxSlot, endSlot)
		}
	}

	if minSlot == slotsPerDay {
		return defaultWindowFrom, defaultWindowTo
	}

	return max(0, minSlot-2), min(slotsPerDay, maxSlot+2)
}

func eventSlotRange(day time.Time, event ical.Event) (int, int, bool) {
	if event.AllDay {
		return 0, 0, false
	}

	dayStart := beginningOfDay(day)
	dayEnd := dayStart.AddDate(0, 0, 1)
	if !eventOverlapsDay(event, dayStart, dayEnd) {
		return 0, 0, false
	}

	start := maxTime(event.StartDate, dayStart)
	end := minTime(event.EndDate, dayEnd)
	if !end.After(start) {
		return 0, 0, false
	}

	startSlot := wallClockSlot(dayStart, start, false)
	endSlot := wallClockSlot(dayStart, end, true)

	if endSlot <= startSlot {
		endSlot = min(slotsPerDay, startSlot+1)
	}

	return startSlot, endSlot, true
}

func timelineLabel(slot int) string {
	slotTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(slot) * slotDuration)
	return slotTime.Format("3:04 PM")
}

func renderEventLine(text, marker, color string, accents []eventAccentSegment, width int, backgroundColor string) string {
	if width <= 0 {
		return ""
	}

	prefix := renderAccentPrefix(accents, backgroundColor)
	prefixWidth := lipgloss.Width(prefix)
	if prefixWidth >= width {
		return ansi.Truncate(prefix, width, "")
	}

	textWidth := width - prefixWidth
	return prefix + eventBackgroundStyle(backgroundColor).Render(renderEventBody(text, marker, color, backgroundColor, textWidth, false))
}

func padRight(text string, width int) string {
	if padding := width - lipgloss.Width(text); padding > 0 {
		return text + strings.Repeat(" ", padding)
	}
	return text
}

func padCenter(text string, width int) string {
	if padding := width - lipgloss.Width(text); padding > 0 {
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
	}
	return text
}

func renderEventBody(text, marker, color, backgroundColor string, width int, center bool) string {
	if width <= 0 {
		return ""
	}

	suffix := renderEventMarker(marker, color, backgroundColor, width)
	textWidth := max(0, width-lipgloss.Width(suffix))
	if textWidth == 0 {
		return suffix
	}

	text = ansi.Truncate(text, textWidth, "…")
	if center {
		return padCenter(text, textWidth) + suffix
	}
	return padRight(text, textWidth) + suffix
}

func renderEventMarker(marker, color, backgroundColor string, width int) string {
	marker = strings.TrimSpace(marker)
	if marker == "" || width <= 0 {
		return ""
	}

	if lipgloss.Width(marker) >= width {
		return recurringMarkerStyle(color, backgroundColor).Render(ansi.Truncate(marker, width, ""))
	}

	padding := strings.Repeat(" ", min(recurringMarkerPad, width-lipgloss.Width(marker)))
	return recurringMarkerStyle(color, backgroundColor).Render(marker + padding)
}

func eventMarkerWidth(marker string) int {
	marker = strings.TrimSpace(marker)
	if marker == "" {
		return 0
	}
	return recurringMarkerPad + lipgloss.Width(marker)
}

func renderAccentPrefix(accents []eventAccentSegment, backgroundColor string) string {
	var b strings.Builder
	for _, accent := range accents {
		b.WriteString(colorStyle(accent.color).Background(lipgloss.Color(backgroundColor)).Render(accent.accent))
	}
	return b.String()
}
func buildTimedEventLayouts(blocks []timedEventBlock, calendarColors map[string]string, dayWidth int) map[int]timedEventLayout {
	layouts := make(map[int]timedEventLayout)
	blocksByCluster := make(map[int][]*timedEventBlock)

	for i := range blocks {
		block := &blocks[i]
		blocksByCluster[block.cluster] = append(blocksByCluster[block.cluster], block)
	}

	for cluster, clusterBlocks := range blocksByCluster {
		columnCount := 0
		for _, block := range clusterBlocks {
			columnCount = max(columnCount, block.layer+1)
		}

		widths, separator := timedEventColumnLayout(clusterBlocks, calendarColors, columnCount, dayWidth)
		layouts[cluster] = timedEventLayout{
			columnWidths: widths,
			separator:    separator,
		}
	}

	return layouts
}

func timedEventColumnLayout(blocks []*timedEventBlock, calendarColors map[string]string, columnCount, dayWidth int) ([]int, string) {
	separatorWidth := 1
	if columnCount <= 1 {
		separatorWidth = 0
	}

	minWidths := make([]int, columnCount)
	desiredWidths := make([]int, columnCount)
	for _, block := range blocks {
		layer := block.layer
		color := eventCalendarColor(block.event, calendarColors)
		prefixWidth := lipgloss.Width(renderAccentPrefix([]eventAccentSegment{{
			accent: block.accent,
			color:  color,
		}}, eventBackgroundColor(color)))
		minWidths[layer] = max(minWidths[layer], prefixWidth+eventMarkerWidth(displayEventMarker(block.event)))
		desiredWidths[layer] = max(desiredWidths[layer], prefixWidth+eventDisplayWidth(block.event))
	}

	for separatorWidth > 0 && sumWidths(minWidths)+separatorWidth*(columnCount-1) > dayWidth {
		separatorWidth--
	}

	availableWidth := dayWidth - separatorWidth*(columnCount-1)
	if sumWidths(minWidths) > availableWidth {
		return splitWidths(availableWidth, columnCount), strings.Repeat(" ", separatorWidth)
	}

	widths := slices.Clone(minWidths)
	extraWidth := availableWidth - sumWidths(widths)
	for extraWidth > 0 {
		progress := false
		for i := range widths {
			if widths[i] >= desiredWidths[i] {
				continue
			}
			widths[i]++
			extraWidth--
			progress = true
			if extraWidth == 0 {
				break
			}
		}
		if !progress {
			break
		}
	}

	for extraWidth > 0 {
		for i := range widths {
			widths[i]++
			extraWidth--
			if extraWidth == 0 {
				break
			}
		}
	}

	return widths, strings.Repeat(" ", separatorWidth)
}

func splitWidths(totalWidth, count int) []int {
	if count <= 0 {
		return nil
	}

	widths := make([]int, count)
	baseWidth := totalWidth / count
	remainder := totalWidth % count
	for i := range widths {
		widths[i] = baseWidth
		if i < remainder {
			widths[i]++
		}
	}

	return widths
}

func sumWidths(widths []int) int {
	total := 0
	for _, width := range widths {
		total += width
	}
	return total
}

func assignTimedEventClusters(blocks []timedEventBlock) {
	cluster := -1
	clusterEnd := 0

	for i := range blocks {
		if cluster == -1 || blocks[i].startSlot >= clusterEnd {
			cluster++
			clusterEnd = blocks[i].endSlot
		} else {
			clusterEnd = max(clusterEnd, blocks[i].endSlot)
		}

		blocks[i].cluster = cluster
	}
}

func assignTimedEventLayers(blocks []timedEventBlock) {
	layerEnds := make([]int, 0, len(blocks))
	currentCluster := -1

	for i := range blocks {
		if blocks[i].cluster != currentCluster {
			layerEnds = layerEnds[:0]
			currentCluster = blocks[i].cluster
		}

		layer := len(layerEnds)
		for candidate, endSlot := range layerEnds {
			if blocks[i].startSlot >= endSlot {
				layer = candidate
				break
			}
		}

		blocks[i].layer = layer
		if layer == len(layerEnds) {
			layerEnds = append(layerEnds, blocks[i].endSlot)
			continue
		}

		layerEnds[layer] = blocks[i].endSlot
	}
}

func timedEventAccent(day time.Time, event ical.Event) string {
	if clippedEventDuration(day, event) <= slotDuration/2 {
		return shortEventAccent
	}
	return eventAccent
}

func clippedEventDuration(day time.Time, event ical.Event) time.Duration {
	dayStart := beginningOfDay(day)
	dayEnd := dayStart.AddDate(0, 0, 1)
	start := maxTime(event.StartDate, dayStart)
	end := minTime(event.EndDate, dayEnd)
	if !end.After(start) {
		return 0
	}
	return end.Sub(start)
}

func displayEventTitle(event ical.Event) string {
	title := strings.TrimSpace(event.Title)
	if title == "" {
		title = "(untitled event)"
	}
	return title
}

func displayEventMarker(event ical.Event) string {
	if event.Recurring {
		return recurringMarker
	}
	return ""
}

func eventDisplayWidth(event ical.Event) int {
	width := lipgloss.Width(displayEventTitle(event))
	if marker := displayEventMarker(event); marker != "" {
		width += eventMarkerWidth(marker)
	}
	return width
}
