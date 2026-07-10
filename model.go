package main

import (
	"context"
	"errors"
	"strings"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
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

type watchCalendarReadyMsg struct {
	changes <-chan struct{}
	cancel  context.CancelFunc
	err     error
}

type watchCalendarChangedMsg struct{}

type watchCalendarStoppedMsg struct{}

type createEventMsg struct {
	event *ical.Event
	err   error
}
type model struct {
	loading           bool
	data              calendarData
	err               error
	width             int
	height            int
	viewDayOffset     int
	timedScrollOffset int
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
	return model{loading: true, focusPending: true}
}

func (m model) Init() tea.Cmd {
	now := time.Now()
	viewDay := beginningOfDay(now).AddDate(0, 0, m.viewDayOffset)
	return tea.Batch(loadCalendarCmd(viewDay, now), currentTimeTickCmd(now), calendarRefreshTickCmdFunc(), startCalendarWatchCmd())
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
		switch msg.String() {
		case "ctrl+c", "q":
			if m.watchCancel != nil {
				m.watchCancel()
				m.watchCancel = nil
				m.watchChanges = nil
			}
			return m, tea.Quit
		case "t":
			m.viewDayOffset = 0
			m.focusPending = true
			now := m.referenceTime()
			return m, loadCalendarCmd(m.currentViewDay(), now)
		case "h":
			m.viewDayOffset--
			return m, loadCalendarCmd(m.currentViewDay(), m.referenceTime())
		case "l":
			m.viewDayOffset++
			return m, loadCalendarCmd(m.currentViewDay(), m.referenceTime())
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
		m.data.currentTime = time.Time(msg)
		m.clampTimedScrollOffset()
		return m, currentTimeTickCmd(time.Time(msg))
	case calendarRefreshTickMsg:
		m.data.currentTime = time.Time(msg)
		m.clampTimedScrollOffset()
		return m, tea.Batch(loadCalendarCmd(m.currentViewDay(), time.Time(msg)), calendarRefreshTickCmdFunc())
	case watchCalendarReadyMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.watchChanges = msg.changes
		m.watchCancel = msg.cancel
		return m, waitForCalendarChangeCmd(msg.changes)
	case watchCalendarChangedMsg:
		return m, tea.Batch(loadCalendarCmd(m.currentViewDay(), m.referenceTime()), waitForCalendarChangeCmd(m.watchChanges))
	case watchCalendarStoppedMsg:
		if m.watchCancel != nil {
			m.watchCancel()
			m.watchCancel = nil
		}
		m.watchChanges = nil
		return m, startCalendarWatchCmd()
	case createEventMsg:
		m.createDialog.submitting = false
		if msg.err != nil {
			m.createDialog.err = msg.err
			return m, nil
		}

		m.closeCreateDialog()
		m.clampTimedScrollOffset()
		return m, loadCalendarCmd(m.currentViewDay(), m.referenceTime())
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
	return renderScreenWithHeight(content, m.width, m.height)
}

func (m model) baseViewContent(contentWidth, contentHeight int) string {
	footer := footerView(m.showCreateDialog, contentWidth)
	calendarHeight := m.calendarViewportHeight(contentHeight)

	var b strings.Builder
	switch {
	case m.loading:
		now := m.data.currentTime
		if now.IsZero() {
			now = time.Now()
		}
		b.WriteString(renderLoadingCalendarLayoutForDayWithHeightAndScroll(m.currentViewDay(), now, contentWidth, calendarHeight, m.timedScrollOffset))
	case m.err != nil:
		b.WriteString(friendlyError(m.err))
	default:
		b.WriteString(renderCalendarLayoutWithHeightAndScroll(m.renderData(), contentWidth, calendarHeight, m.timedScrollOffset))
	}

	content := b.String()
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

func currentTimeTickCmd(now time.Time) tea.Cmd {
	next := now.Truncate(currentTimeTickEvery).Add(currentTimeTickEvery)
	return tea.Tick(time.Until(next), func(t time.Time) tea.Msg {
		return currentTimeTickMsg(t)
	})
}

func calendarRefreshTickCmdFunc() tea.Cmd {
	return tea.Tick(calendarRefreshEvery, func(t time.Time) tea.Msg {
		return calendarRefreshTickMsg(t)
	})
}

func startCalendarWatchCmd() tea.Cmd {
	return func() tea.Msg {
		client, err := ical.New()
		if err != nil {
			return watchCalendarReadyMsg{err: err}
		}

		changes, cancel, err := openCalendarWatch(client.WatchChanges, time.Sleep)
		if err != nil {
			return watchCalendarReadyMsg{err: err}
		}

		return watchCalendarReadyMsg{
			changes: changes,
			cancel:  cancel,
		}
	}
}

func openCalendarWatch(start func(context.Context) (<-chan struct{}, error), sleep func(time.Duration)) (<-chan struct{}, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())

	for attempt := range watchStartRetries {
		changes, err := start(ctx)
		if err == nil {
			return changes, cancel, nil
		}
		if !isWatchAlreadyActiveError(err) || attempt == watchStartRetries-1 {
			cancel()
			return nil, nil, err
		}

		sleep(watchRetryDelay * time.Duration(1<<attempt))
	}

	cancel()
	return nil, nil, errors.New("calendar: failed to start watcher")
}

func isWatchAlreadyActiveError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "watcher already active")
}

func waitForCalendarChangeCmd(changes <-chan struct{}) tea.Cmd {
	if changes == nil {
		return nil
	}

	return func() tea.Msg {
		if _, ok := <-changes; !ok {
			return watchCalendarStoppedMsg{}
		}
		return watchCalendarChangedMsg{}
	}
}

func contentWidthForTerminal(width int) int {
	if width <= 0 {
		return 0
	}

	contentWidth := width - screenPaddingX*2
	if contentWidth < 0 {
		return 0
	}

	return contentWidth
}

func contentHeightForTerminal(height int) int {
	if height <= 0 {
		return 0
	}

	contentHeight := height - screenPaddingY*2
	if contentHeight < 0 {
		return 0
	}

	return contentHeight
}

func renderScreenWithHeight(content string, width, height int) string {
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
		return maxLoadingCalendarScroll(m.currentViewDay(), now, contentWidth, contentHeight, calendarSlotWindow{})
	case m.err != nil:
		return 0
	default:
		return maxCalendarLayoutScroll(m.data, contentWidth, contentHeight, calendarSlotWindow{})
	}
}

func (m *model) clampTimedScrollOffset() {
	if m == nil {
		return
	}
	m.timedScrollOffset = min(m.timedScrollOffset, m.maxTimedScrollOffset())
	if m.timedScrollOffset < 0 {
		m.timedScrollOffset = 0
	}
}

func (m *model) closeCreateDialog() {
	if m == nil {
		return
	}
	m.showCreateDialog = false
	m.createDialog = createEventDialog{}
	m.clearDraft()
}

func (m *model) clearDraft() {
	if m == nil {
		return
	}
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
		start, _ := loadingSlotWindow(now)
		return start
	}
	start, _ := visibleSlotWindow(m.data.sections)
	return start
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
	if m == nil || !m.focusPending {
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
