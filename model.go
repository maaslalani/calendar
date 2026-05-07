package main

import (
	"context"
	"errors"
	ical "github.com/BRO3886/go-eventkit/calendar"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"time"
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
	pinnedSlotWindow  calendarSlotWindow
	watchChanges      <-chan struct{}
	watchCancel       context.CancelFunc
	activeCalendarID  string
	statusMessage     string
	showCreateDialog  bool
	createDialog      createEventDialog
}

func initialModel() model {
	return model{loading: true}
}

func (m model) Init() tea.Cmd {
	now := time.Now()
	viewDay := beginningOfDay(now).AddDate(0, 0, m.viewDayOffset)
	return tea.Batch(loadCalendarCmd(viewDay, now, m.activeCalendarID), currentTimeTickCmd(now), calendarRefreshTickCmdFunc(), startCalendarWatchCmd())
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
		case "r":
			m.viewDayOffset = 0
			m.timedScrollOffset = 0
			m.pinnedSlotWindow = calendarSlotWindow{}
			now := m.referenceTime()
			return m, loadCalendarCmd(m.currentViewDay(), now, m.activeCalendarID)
		case "h":
			m.pinCurrentSlotWindow()
			m.viewDayOffset--
			return m, loadCalendarCmd(m.currentViewDay(), m.referenceTime(), m.activeCalendarID)
		case "l":
			m.pinCurrentSlotWindow()
			m.viewDayOffset++
			return m, loadCalendarCmd(m.currentViewDay(), m.referenceTime(), m.activeCalendarID)
		case "j", "down":
			m.timedScrollOffset = min(m.timedScrollOffset+1, m.maxTimedScrollOffset())
		case "k", "up":
			m.timedScrollOffset = max(0, m.timedScrollOffset-1)
		case "n":
			m.showCreateDialog = true
			var cmd tea.Cmd
			m.createDialog, cmd = newCreateEventDialogForDay(m.currentViewDay(), m.referenceTime())
			return m, cmd
		case "c":
			nextID, label := nextCalendarFilter(m.activeCalendarID, m.data.calendars)
			m.activeCalendarID = nextID
			m.statusMessage = "Showing " + label
			m.timedScrollOffset = 0
			return m, loadCalendarCmd(m.currentViewDay(), m.referenceTime(), m.activeCalendarID)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampTimedScrollOffset()
	case loadCalendarMsg:
		m.loading = false
		m.data = msg.data
		m.err = msg.err
		m.clampTimedScrollOffset()
	case currentTimeTickMsg:
		m.data.currentTime = time.Time(msg)
		m.clampTimedScrollOffset()
		return m, currentTimeTickCmd(time.Time(msg))
	case calendarRefreshTickMsg:
		m.data.currentTime = time.Time(msg)
		m.clampTimedScrollOffset()
		return m, tea.Batch(loadCalendarCmd(m.currentViewDay(), time.Time(msg), m.activeCalendarID), calendarRefreshTickCmdFunc())
	case watchCalendarReadyMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.watchChanges = msg.changes
		m.watchCancel = msg.cancel
		return m, waitForCalendarChangeCmd(msg.changes)
	case watchCalendarChangedMsg:
		return m, tea.Batch(loadCalendarCmd(m.currentViewDay(), m.referenceTime(), m.activeCalendarID), waitForCalendarChangeCmd(m.watchChanges))
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

		m.showCreateDialog = false
		m.createDialog = createEventDialog{}
		m.clampTimedScrollOffset()
		return m, loadCalendarCmd(m.currentViewDay(), m.referenceTime(), m.activeCalendarID)
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
	var sections []string
	if status := renderStatusBanner(m.activeCalendarLabel(), m.statusMessage, contentWidth); status != "" {
		sections = append(sections, status)
		if contentHeight > 0 {
			contentHeight -= lipgloss.Height(status)
			if contentHeight > 0 {
				contentHeight--
			}
		}
	}

	var b strings.Builder
	switch {
	case m.loading:
		now := m.data.currentTime
		if now.IsZero() {
			now = time.Now()
		}
		b.WriteString(renderLoadingCalendarLayoutForDayWithHeightAndScrollUsingWindow(m.currentViewDay(), now, contentWidth, contentHeight, m.timedScrollOffset, m.pinnedSlotWindow))
	case m.err != nil:
		b.WriteString(friendlyError(m.err))
	default:
		b.WriteString(renderCalendarLayoutWithHeightAndScrollUsingWindow(m.data, contentWidth, contentHeight, m.timedScrollOffset, m.pinnedSlotWindow))
	}

	sections = append(sections, b.String())
	return strings.Join(sections, "\n\n")
}

func loadCalendarCmd(viewDay, now time.Time, activeCalendarID string) tea.Cmd {
	return func() tea.Msg {
		data, err := loadCalendar(viewDay, now, activeCalendarID)
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

	for attempt := 0; attempt < watchStartRetries; attempt++ {
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

func (m model) activeCalendarLabel() string {
	if strings.TrimSpace(m.activeCalendarID) == "" {
		return ""
	}
	return calendarFilterLabel(m.data.calendars, m.activeCalendarID)
}

func (m model) maxTimedScrollOffset() int {
	contentWidth := contentWidthForTerminal(m.width)
	contentHeight := contentHeightForTerminal(m.height)

	if status := renderStatusBanner(m.activeCalendarLabel(), m.statusMessage, contentWidth); status != "" {
		contentHeight -= lipgloss.Height(status)
		if contentHeight > 0 {
			contentHeight--
		}
	}
	if contentHeight <= 0 {
		return 0
	}

	switch {
	case m.loading:
		now := m.data.currentTime
		if now.IsZero() {
			now = time.Now()
		}
		return maxLoadingCalendarScroll(m.currentViewDay(), now, contentWidth, contentHeight, m.pinnedSlotWindow)
	case m.err != nil:
		return 0
	default:
		return maxCalendarLayoutScroll(m.data, contentWidth, contentHeight, m.pinnedSlotWindow)
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

func (m model) currentSlotWindow() calendarSlotWindow {
	contentWidth := contentWidthForTerminal(m.width)
	contentHeight := contentHeightForTerminal(m.height)

	if status := renderStatusBanner(m.activeCalendarLabel(), m.statusMessage, contentWidth); status != "" {
		contentHeight -= lipgloss.Height(status)
		if contentHeight > 0 {
			contentHeight--
		}
	}

	switch {
	case m.loading:
		now := m.data.currentTime
		if now.IsZero() {
			now = time.Now()
		}
		return effectiveLoadingSlotWindow(now, calendarFixedRowCount(make([][]string, 3), false), contentHeight, m.pinnedSlotWindow)
	case m.err != nil:
		return m.pinnedSlotWindow
	default:
		allDayLines := make([][]string, len(m.data.sections))
		for i, section := range m.data.sections {
			allDayEvents, _ := partitionDayEvents(section.events)
			allDayLines[i] = renderAllDayLines(allDayEvents, m.data.calendarColors, dayColumnWidth)
		}
		return effectiveCalendarSlotWindow(m.data.sections, m.data.currentTime, calendarFixedRowCount(allDayLines, len(m.data.legend) > 0), contentHeight, m.pinnedSlotWindow)
	}
}

func (m *model) pinCurrentSlotWindow() {
	if m == nil {
		return
	}
	m.pinnedSlotWindow = m.currentSlotWindow()
}

func renderStatusBanner(activeCalendarLabel, statusMessage string, width int) string {
	parts := make([]string, 0, 3)
	if activeCalendarLabel != "" {
		parts = append(parts, statusPillStyle.Render("Viewing"))
		parts = append(parts, activeCalendarLabel)
	}
	if strings.TrimSpace(statusMessage) != "" {
		parts = append(parts, statusTextStyle.Render(statusMessage))
	}
	if len(parts) == 0 {
		return ""
	}

	banner := strings.Join(parts, "  ")
	if width > 0 {
		banner = ansi.Truncate(banner, width, "…")
	}
	return banner
}
