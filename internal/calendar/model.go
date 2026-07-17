package calendar

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type loadCalendarMsg struct {
	data calendarData
	err  error
}

type currentTimeTickMsg time.Time

type calendarRefreshTickMsg time.Time

type createEventMsg struct {
	err error
}

type model struct {
	loading           bool
	data              calendarData
	err               error
	width             int
	height            int
	viewDayOffset     int
	timedScrollOffset int
	lastScrollAt      time.Time
	scrollAxis        scrollAxis
	scrollIntentX     int
	scrollIntentY     int
	dayScrollProgress int
	focusPending      bool
	watchChanges      <-chan struct{}
	watchCancel       context.CancelFunc
	showCreateDialog  bool
	createDialog      createEventDialog
	dragging          bool
	dragMoved         bool
	dragDay           time.Time
	dragStartSlot     int
	dragEndSlot       int
	draftActive       bool
	draftDay          time.Time
	draftStartSlot    int
	draftEndSlot      int
}

func initialModel() model {
	m := model{loading: true, focusPending: true}
	if calendarDemoEnabled() {
		m.data.currentTime = demoCurrentTime(time.Now())
	}
	return m
}

func (m model) Init() tea.Cmd {
	now := time.Now()
	viewDay := beginningOfDay(now).AddDate(0, 0, m.viewDayOffset)
	cmds := []tea.Cmd{loadCalendarCmd(viewDay, now), currentTimeTickCmd(now), calendarRefreshTickCmd()}
	if !calendarDemoEnabled() {
		cmds = append(cmds, startCalendarWatchCmd())
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case cursor.BlinkMsg:
		if m.showCreateDialog {
			return m, m.createDialog.updateFocusedInput(msg)
		}
	case tea.KeyMsg:
		if m.showCreateDialog {
			return m.updateCreateDialog(msg)
		}
		return m.updateKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applyPendingFocus()
		m.clampTimedScrollOffset()
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case loadCalendarMsg:
		m.loading = false
		m.data = msg.data
		m.err = msg.err
		m.applyPendingFocus()
		m.clampTimedScrollOffset()
	case currentTimeTickMsg:
		m.setCurrentTime(time.Time(msg))
		return m, currentTimeTickCmd(time.Time(msg))
	case calendarRefreshTickMsg:
		m.setCurrentTime(time.Time(msg))
		return m, tea.Batch(m.reloadCalendarCmd(), calendarRefreshTickCmd())
	case watchCalendarReadyMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.watchChanges = msg.changes
		m.watchCancel = msg.cancel
		return m, waitForCalendarChangeCmd(msg.changes)
	case watchCalendarChangedMsg:
		return m, tea.Batch(m.reloadCalendarCmd(), waitForCalendarChangeCmd(m.watchChanges))
	case watchCalendarStoppedMsg:
		m.stopCalendarWatch()
		if calendarDemoEnabled() {
			return m, nil
		}
		return m, startCalendarWatchCmd()
	case createEventMsg:
		m.createDialog.submitting = false
		if msg.err != nil {
			m.createDialog.err = msg.err
			return m, nil
		}

		m.closeCreateDialog()
		m.clampTimedScrollOffset()
		return m, m.reloadCalendarCmd()
	}

	return m, nil
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.stopCalendarWatch()
		return m, tea.Quit
	case "t":
		m.viewDayOffset = 0
		m.focusPending = true
		return m, m.reloadCalendarCmd()
	case "h", "left":
		m.viewDayOffset--
		return m, m.reloadCalendarCmd()
	case "l", "right":
		m.viewDayOffset++
		return m, m.reloadCalendarCmd()
	case "j", "down":
		m.timedScrollOffset = min(m.timedScrollOffset+1, m.maxTimedScrollOffset())
	case "k", "up":
		m.timedScrollOffset = max(0, m.timedScrollOffset-1)
	case "n":
		m.showCreateDialog = true
		var cmd tea.Cmd
		m.createDialog, cmd = newCreateEventDialogForDay(m.currentViewDay(), m.referenceTime())
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	contentWidth := contentWidthForTerminal(m.width)
	contentHeight := contentHeightForTerminal(m.height)
	content := m.baseViewContent(contentWidth, contentHeight)
	if m.showCreateDialog {
		content = compositeLayer(
			content,
			renderCreateEventDialog(m.createDialog, contentWidth, contentHeight),
			contentWidth,
			contentHeight,
		)
	}
	return renderScreen(content, m.width, m.height)
}

func (m model) baseViewContent(contentWidth, contentHeight int) string {
	footer := footerView(m.showCreateDialog, contentWidth)
	calendarHeight := contentHeight
	if footer != "" {
		calendarHeight = shrinkContentHeight(calendarHeight, footer)
	}

	var content string
	switch {
	case m.loading:
		now := m.data.currentTime
		if now.IsZero() {
			now = time.Now()
		}
		content = renderLoadingCalendarLayout(m.currentViewDay(), now, contentWidth, calendarHeight, m.timedScrollOffset)
	case m.err != nil:
		content = friendlyError(m.err)
	default:
		content = renderCalendarLayout(m.renderData(), contentWidth, calendarHeight, m.timedScrollOffset)
	}

	if footer != "" {
		content += "\n\n" + footer
	}
	return content
}

func (m model) calendarViewportHeight(contentHeight int) int {
	contentWidth := contentWidthForTerminal(m.width)
	if footer := footerView(m.showCreateDialog, contentWidth); footer != "" {
		contentHeight = shrinkContentHeight(contentHeight, footer)
	}
	return contentHeight
}

func shrinkContentHeight(contentHeight int, banner string) int {
	if contentHeight <= 0 {
		return contentHeight
	}
	contentHeight -= lipgloss.Height(banner)
	if contentHeight > 0 {
		contentHeight--
	}
	return contentHeight
}

func loadCalendarCmd(viewDay, now time.Time) tea.Cmd {
	return func() tea.Msg {
		data, err := loadCalendar(viewDay, now)
		return loadCalendarMsg{
			data: data,
			err:  err,
		}
	}
}

func (m model) reloadCalendarCmd() tea.Cmd {
	return loadCalendarCmd(m.currentViewDay(), m.referenceTime())
}

func currentTimeTickCmd(now time.Time) tea.Cmd {
	next := now.Truncate(currentTimeTickEvery).Add(currentTimeTickEvery)
	return tea.Tick(time.Until(next), func(t time.Time) tea.Msg {
		return currentTimeTickMsg(t)
	})
}

func calendarRefreshTickCmd() tea.Cmd {
	return tea.Tick(calendarRefreshEvery, func(t time.Time) tea.Msg {
		return calendarRefreshTickMsg(t)
	})
}

func contentWidthForTerminal(width int) int {
	return max(0, width-screenPaddingX*2)
}

func contentHeightForTerminal(height int) int {
	return max(0, height-screenPaddingY*2)
}

func renderScreen(content string, width, height int) string {
	style := screenStyle
	if width > 0 {
		style = style.Width(width)
	}
	if height > lipgloss.Height(content)+screenPaddingY*2 {
		style = style.Height(height)
	}
	return style.Render(content)
}

func (m model) referenceTime() time.Time {
	if !m.data.currentTime.IsZero() {
		return m.data.currentTime
	}
	return time.Now()
}

func (m model) currentViewDay() time.Time {
	return beginningOfDay(m.referenceTime()).AddDate(0, 0, m.viewDayOffset)
}

func (m *model) setCurrentTime(now time.Time) {
	if calendarDemoEnabled() {
		now = demoCurrentTime(now)
	}
	m.data.currentTime = now
	m.clampTimedScrollOffset()
}

func (m model) maxTimedScrollOffset() int {
	contentWidth := contentWidthForTerminal(m.width)
	contentHeight := m.calendarViewportHeight(contentHeightForTerminal(m.height))
	if contentHeight <= 0 {
		return 0
	}

	switch {
	case m.loading:
		now := m.data.currentTime
		if now.IsZero() {
			now = time.Now()
		}
		return maxLoadingCalendarScroll(m.currentViewDay(), now, contentWidth, contentHeight)
	case m.err != nil:
		return 0
	default:
		return maxCalendarLayoutScroll(m.data, contentWidth, contentHeight)
	}
}

func (m *model) clampTimedScrollOffset() {
	m.timedScrollOffset = max(0, min(m.timedScrollOffset, m.maxTimedScrollOffset()))
}

func (m *model) closeCreateDialog() {
	m.showCreateDialog = false
	m.createDialog = createEventDialog{}
	m.clearDraft()
}

func (m *model) clearDraft() {
	m.draftActive = false
	m.draftDay = time.Time{}
	m.draftStartSlot = 0
	m.draftEndSlot = 0
}

func (m model) renderNow() time.Time {
	if m.loading && m.data.currentTime.IsZero() {
		return time.Now()
	}
	return m.data.currentTime
}

func (m model) focusStartSlot(now time.Time) int {
	if m.loading || len(m.data.sections) == 0 {
		return loadingStartSlot(now)
	}
	return visibleStartSlot(m.data.sections)
}

func (m model) focusScrollOffset() int {
	now := m.renderNow()
	focusStart := m.focusStartSlot(now)
	offset := focusStart
	if slot, ok := currentTimeMarkerSlot(now, 0, slotsPerDay); ok && slot < focusStart {
		offset++
	}
	return max(0, min(offset, m.maxTimedScrollOffset()))
}

func (m *model) applyPendingFocus() {
	if !m.focusPending {
		return
	}
	if m.calendarViewportHeight(contentHeightForTerminal(m.height)) <= 0 {
		return
	}

	m.timedScrollOffset = m.focusScrollOffset()
	if !m.loading {
		m.focusPending = false
	}
}
