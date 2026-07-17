package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// calendarViewGeometry captures the on-screen placement of the calendar grid so
// mouse coordinates can be mapped back to a day and time slot.
type calendarViewGeometry struct {
	headerRows    int
	sectionDates  []time.Time
	sectionWidths []int
	timedRowSlots []int
	visibleStart  int
	visibleCount  int
}

func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.showCreateDialog {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.timedScrollOffset = max(0, m.timedScrollOffset-1)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.timedScrollOffset = min(m.timedScrollOffset+1, m.maxTimedScrollOffset())
		return m, nil
	}

	switch msg.Action {
	case tea.MouseActionPress:
		return m.beginDrag(msg)
	case tea.MouseActionMotion:
		return m.extendDrag(msg)
	case tea.MouseActionRelease:
		return m.finishDrag()
	}

	return m, nil
}

func (m model) beginDrag(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	geo, ok := m.calendarViewGeometry()
	if !ok {
		return m, nil
	}
	day, dayOK := geo.dayAt(msg.X)
	slot, slotOK := geo.slotAt(msg.Y)
	if !dayOK || !slotOK {
		return m, nil
	}

	m.dragging = true
	m.dragMoved = false
	m.dragDay = day
	m.dragStartSlot = slot
	m.dragEndSlot = slot
	return m, nil
}

func (m model) extendDrag(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if !m.dragging {
		return m, nil
	}

	// Cell motion events are only emitted while a button is held, so any motion
	// here marks the gesture as a real drag rather than a click.
	m.dragMoved = true

	geo, ok := m.calendarViewGeometry()
	if !ok {
		return m, nil
	}
	if slot, slotOK := geo.slotAt(msg.Y); slotOK {
		m.dragEndSlot = slot
	}
	return m, nil
}

func (m model) finishDrag() (tea.Model, tea.Cmd) {
	if !m.dragging {
		return m, nil
	}

	moved := m.dragMoved
	day := m.dragDay
	startSlot := min(m.dragStartSlot, m.dragEndSlot)
	endSlot := max(m.dragStartSlot, m.dragEndSlot)
	m.dragging = false
	m.dragMoved = false
	m.dragDay = time.Time{}
	m.dragStartSlot = 0
	m.dragEndSlot = 0

	// A click without any drag motion should not create an event; only an
	// actual drag opens the create dialog.
	if !moved {
		return m, nil
	}

	m.draftActive = true
	m.draftDay = day
	m.draftStartSlot = startSlot
	m.draftEndSlot = endSlot

	return m.openCreateDialogForSelection(day, startSlot, endSlot)
}

func (m model) openCreateDialogForSelection(day time.Time, startSlot, endSlot int) (tea.Model, tea.Cmd) {
	start := beginningOfDay(day).Add(time.Duration(startSlot) * slotDuration)
	durationMinutes := (endSlot - startSlot + 1) * int(slotDuration/time.Minute)

	m.showCreateDialog = true
	var cmd tea.Cmd
	m.createDialog, cmd = newCreateEventDialogAt(start, durationMinutes)
	return m, cmd
}

// renderData returns the calendar data to display, injecting the in-progress
// drag selection so it previews as a provisional event. The same preview stays
// visible as a draft while the create dialog is open so it is not hidden behind
// the dialog.
func (m model) renderData() calendarData {
	switch {
	case m.dragging && m.dragMoved:
		return m.data.withSelection(m.dragDay, min(m.dragStartSlot, m.dragEndSlot), max(m.dragStartSlot, m.dragEndSlot))
	case m.showCreateDialog && m.draftActive:
		return m.data.withSelection(m.draftDay, m.draftStartSlot, m.draftEndSlot)
	default:
		return m.data
	}
}

func (m model) calendarViewGeometry() (calendarViewGeometry, bool) {
	contentWidth := contentWidthForTerminal(m.width)
	if contentWidth <= 0 {
		return calendarViewGeometry{}, false
	}
	calendarHeight := m.calendarViewportHeight(contentHeightForTerminal(m.height))
	if calendarHeight <= 0 {
		return calendarViewGeometry{}, false
	}

	sections, dates, ok := m.hitRenderSections(contentWidth)
	if !ok || len(dates) == 0 {
		return calendarViewGeometry{}, false
	}

	visibleStart, visibleCount := visibleTimedRowRange(sections, calendarHeight, m.timedScrollOffset)
	return calendarViewGeometry{
		headerRows:    len(sections.headerRows),
		sectionDates:  dates,
		sectionWidths: newCalendarLayout(len(dates), contentWidth).sectionWidths,
		timedRowSlots: sections.timedRowSlots,
		visibleStart:  visibleStart,
		visibleCount:  visibleCount,
	}, true
}

func (m model) hitRenderSections(contentWidth int) (calendarRenderSections, []time.Time, bool) {
	switch {
	case m.err != nil:
		return calendarRenderSections{}, nil, false
	case m.loading:
		viewDay := m.currentViewDay()
		sections := buildLoadingCalendarRenderSections(viewDay, m.renderNow(), contentWidth, calendarSlotWindow{})
		return sections, sectionDatesFrom(buildLoadingSections(viewDay)), true
	default:
		if len(m.data.sections) == 0 {
			return calendarRenderSections{}, nil, false
		}
		sections := buildCalendarRenderSections(m.data, contentWidth, calendarSlotWindow{})
		return sections, sectionDatesFrom(m.data.sections), true
	}
}

func (g calendarViewGeometry) slotAt(y int) (int, bool) {
	row := y - screenPaddingY - g.headerRows
	if row < 0 || row >= g.visibleCount {
		return 0, false
	}
	index := g.visibleStart + row
	if index < 0 || index >= len(g.timedRowSlots) {
		return 0, false
	}
	return g.timedRowSlots[index], true
}

func (g calendarViewGeometry) dayAt(x int) (time.Time, bool) {
	index, ok := dayColumnAtX(x-screenPaddingX, g.sectionWidths)
	if !ok || index >= len(g.sectionDates) {
		return time.Time{}, false
	}
	return g.sectionDates[index], true
}

// dayColumnAtX maps a content-relative column to the index of the day section
// it falls within, skipping the time axis gutter and inter-column separators.
func dayColumnAtX(localX int, sectionWidths []int) (int, bool) {
	if localX < 0 {
		return 0, false
	}

	separatorWidth := lipgloss.Width(columnSeparator)
	cursor := timeColumnWidth
	for i, width := range sectionWidths {
		cursor += separatorWidth
		if localX >= cursor && localX < cursor+width {
			return i, true
		}
		cursor += width
	}

	return 0, false
}

func sectionDatesFrom(sections []daySection) []time.Time {
	dates := make([]time.Time, len(sections))
	for i, section := range sections {
		dates[i] = section.date
	}
	return dates
}
