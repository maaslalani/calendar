package calendar

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	ical "github.com/BRO3886/go-eventkit/calendar"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	dialogFocusTitle = iota
	dialogFocusMonth
	dialogFocusDay
	dialogFocusYear
	dialogFocusHour
	dialogFocusMinute
	dialogFocusDuration
	dialogFocusSubmit
	dialogFocusCancel
	dialogFocusCount
)

type createEventDialog struct {
	titleInput         textinput.Model
	startDate          time.Time
	hourInput          dialogNumberInput
	minuteInput        dialogNumberInput
	durationInput      dialogNumberInput
	focusIndex         int
	numericInputBuffer string
	submitting         bool
	err                error
}

func (m model) updateCreateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		m.stopCalendarWatch()
		return m, tea.Quit
	}

	switch msg.String() {
	case "esc":
		if m.createDialog.submitting {
			return m, nil
		}
		m.closeCreateDialog()
		return m, nil
	case "tab":
		return m, m.createDialog.moveFocus(1)
	case "shift+tab":
		return m, m.createDialog.moveFocus(-1)
	case "up":
		if m.createDialog.stepFocusedInput(1) {
			return m, nil
		}
	case "down":
		if m.createDialog.stepFocusedInput(-1) {
			return m, nil
		}
	case "left":
		if focus, ok := adjacentDialogFocus(m.createDialog.focusIndex, -1); ok {
			return m, m.createDialog.setFocus(focus)
		}
	case "right":
		if focus, ok := adjacentDialogFocus(m.createDialog.focusIndex, 1); ok {
			return m, m.createDialog.setFocus(focus)
		}
	case "enter":
		return m.submitOrAdvanceCreateDialog()
	}

	if m.createDialog.submitting {
		return m, nil
	}

	return m, m.createDialog.updateFocusedInput(msg)
}

func (m model) submitOrAdvanceCreateDialog() (tea.Model, tea.Cmd) {
	switch m.createDialog.focusIndex {
	case dialogFocusSubmit:
		input, err := m.createDialog.createInput(m.referenceTime())
		if err != nil {
			m.createDialog.err = err
			return m, nil
		}

		m.createDialog.err = nil
		m.createDialog.submitting = true
		return m, createEventCmd(input)
	case dialogFocusCancel:
		m.closeCreateDialog()
		return m, nil
	default:
		return m, m.createDialog.moveFocus(1)
	}
}

func newCreateEventDialogForDay(day, reference time.Time) (createEventDialog, tea.Cmd) {
	base := reference
	if base.IsZero() {
		base = time.Now()
	}

	startDay := day
	if startDay.IsZero() {
		startDay = base
	}
	start := nextQuarterHour(wallClockTimeOnDay(beginningOfDay(startDay), base))
	return newCreateEventDialogAt(start, 30)
}

func newCreateEventDialogAt(start time.Time, durationMinutes int) (createEventDialog, tea.Cmd) {
	if start.IsZero() {
		start = nextQuarterHour(time.Now())
	}
	if durationMinutes < dialogMinuteStep {
		durationMinutes = dialogMinuteStep
	}

	dialog := createEventDialog{
		titleInput:    newDialogTextInput("Title", 24),
		startDate:     beginningOfDay(start),
		hourInput:     newDialogNumberInput(start.Hour(), 0, 23, 1, 4, formatTwoDigitNumber),
		minuteInput:   newDialogNumberInput(start.Minute(), 0, 45, dialogMinuteStep, 4, formatTwoDigitNumber),
		durationInput: newDialogNumberInput(durationMinutes, dialogMinuteStep, -1, dialogMinuteStep, 4, nil),
		focusIndex:    dialogFocusTitle,
	}

	return dialog, dialog.syncFocus()
}

func newDialogTextInput(placeholder string, width int) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.Prompt = ""
	input.Width = width
	return input
}

func (d createEventDialog) createInput(now time.Time) (ical.CreateEventInput, error) {
	if err := d.commitNumericInput(); err != nil {
		return ical.CreateEventInput{}, err
	}

	title := strings.TrimSpace(d.titleInput.Value())
	if title == "" {
		return ical.CreateEventInput{}, errors.New("title is required")
	}

	start, err := d.startTime(now)
	if err != nil {
		return ical.CreateEventInput{}, err
	}

	duration, err := d.eventDuration()
	if err != nil {
		return ical.CreateEventInput{}, err
	}

	return ical.CreateEventInput{
		Title:     title,
		StartDate: start,
		EndDate:   start.Add(duration),
	}, nil
}

func (d createEventDialog) startTime(reference time.Time) (time.Time, error) {
	if d.hourInput.value < 0 || d.hourInput.value > 23 {
		return time.Time{}, errors.New("hour must be between 0 and 23")
	}
	if d.minuteInput.value < 0 || d.minuteInput.value >= 60 || d.minuteInput.value%dialogMinuteStep != 0 {
		return time.Time{}, fmt.Errorf("minutes must be in %d-minute increments", dialogMinuteStep)
	}

	base := reference
	if base.IsZero() {
		base = time.Now()
	}

	startDay := d.startDate
	if startDay.IsZero() {
		startDay = beginningOfDay(base)
	}

	start := time.Date(
		startDay.Year(),
		startDay.Month(),
		startDay.Day(),
		d.hourInput.value,
		d.minuteInput.value,
		0,
		0,
		startDay.Location(),
	)
	if start.Hour() != d.hourInput.value || start.Minute() != d.minuteInput.value {
		return time.Time{}, errors.New("that start time does not exist in your local timezone on that date")
	}
	return start, nil
}

func (d createEventDialog) eventDuration() (time.Duration, error) {
	if d.durationInput.value < dialogMinuteStep {
		return 0, fmt.Errorf("duration must be at least %d minutes", dialogMinuteStep)
	}
	if d.durationInput.value%dialogMinuteStep != 0 {
		return 0, fmt.Errorf("duration must be in %d-minute increments", dialogMinuteStep)
	}
	return time.Duration(d.durationInput.value) * time.Minute, nil
}

func createEventCmd(input ical.CreateEventInput) tea.Cmd {
	if calendarDemoEnabled() {
		return func() tea.Msg {
			return createEventMsg{err: errors.New("event creation is unavailable in demo mode")}
		}
	}

	return func() tea.Msg {
		client, err := ical.New()
		if err != nil {
			return createEventMsg{err: err}
		}

		_, err = client.CreateEvent(input)
		return createEventMsg{err: err}
	}
}

var dialogHorizontalFocusRows = [][]int{
	{dialogFocusMonth, dialogFocusDay, dialogFocusYear, dialogFocusHour, dialogFocusMinute, dialogFocusDuration},
	{dialogFocusSubmit, dialogFocusCancel},
}

type dialogNumberInput struct {
	value  int
	min    int
	max    int
	step   int
	width  int
	hasMax bool
	format func(int) string
}

func nextDialogFocus(current, delta int) int {
	next := current + delta
	for next < 0 {
		next += dialogFocusCount
	}
	return next % dialogFocusCount
}

func adjacentDialogFocus(current, delta int) (int, bool) {
	for _, row := range dialogHorizontalFocusRows {
		for i, focus := range row {
			if focus != current {
				continue
			}
			next := i + delta
			if next < 0 || next >= len(row) {
				return 0, false
			}
			return row[next], true
		}
	}
	return 0, false
}

func newDialogNumberInput(value, minValue, maxValue, step, width int, format func(int) string) dialogNumberInput {
	input := dialogNumberInput{
		value:  value,
		min:    minValue,
		step:   step,
		width:  width,
		format: format,
	}
	if maxValue >= minValue {
		input.max = maxValue
		input.hasMax = true
	}
	return input
}

func (i dialogNumberInput) displayValue() string {
	if i.format != nil {
		return i.format(i.value)
	}
	return strconv.Itoa(i.value)
}

func (i *dialogNumberInput) increment(delta int) {
	next := i.value + delta*i.step
	if i.hasMax {
		i.value = min(i.max, max(i.min, next))
		return
	}
	if next < i.min {
		next = i.min
	}
	i.value = next
}

func formatTwoDigitNumber(value int) string {
	return fmt.Sprintf("%02d", value)
}

func formatDialogMonthValue(value int) string {
	if value < int(time.January) || value > int(time.December) {
		return fmt.Sprintf("%02d", value)
	}
	return time.Month(value).String()[:3]
}

func formatDialogYearValue(value int) string {
	return fmt.Sprintf("%04d", value)
}

func isDialogNumericFocus(index int) bool {
	switch index {
	case dialogFocusMonth, dialogFocusDay, dialogFocusYear, dialogFocusHour, dialogFocusMinute, dialogFocusDuration:
		return true
	default:
		return false
	}
}

func dialogNumericInputMaxDigits(index int) int {
	switch index {
	case dialogFocusYear, dialogFocusDuration:
		return 4
	case dialogFocusMonth, dialogFocusDay, dialogFocusHour, dialogFocusMinute:
		return 2
	default:
		return 0
	}
}

func (d *createEventDialog) setFocus(index int) tea.Cmd {
	if d.focusIndex != index {
		if err := d.commitNumericInput(); err != nil {
			d.err = err
			return nil
		}
	}

	d.err = nil
	d.focusIndex = index
	return d.syncFocus()
}

func (d *createEventDialog) moveFocus(delta int) tea.Cmd {
	return d.setFocus(nextDialogFocus(d.focusIndex, delta))
}

func (d *createEventDialog) syncFocus() tea.Cmd {
	if d.focusIndex == dialogFocusTitle {
		return d.titleInput.Focus()
	}
	d.titleInput.Blur()
	return nil
}

func (d *createEventDialog) stepFocusedInput(delta int) bool {
	if err := d.commitNumericInput(); err != nil {
		d.err = err
		return true
	}

	d.err = nil

	switch d.focusIndex {
	case dialogFocusMonth:
		d.adjustStartMonths(delta)
		return true
	case dialogFocusDay:
		d.adjustStartDays(delta)
		return true
	case dialogFocusYear:
		d.adjustStartYears(delta)
		return true
	case dialogFocusHour:
		d.adjustStartMinutes(delta * 60)
		return true
	case dialogFocusMinute:
		d.adjustStartMinutes(delta * dialogMinuteStep)
		return true
	case dialogFocusDuration:
		d.durationInput.increment(delta)
		return true
	default:
		return false
	}
}

func (d *createEventDialog) adjustStartMinutes(delta int) {
	const minutesPerDay = 24 * 60

	d.startDate = d.currentStartDay(time.Now())

	total := d.hourInput.value*60 + d.minuteInput.value + delta
	dayShift := 0
	for total < 0 {
		total += minutesPerDay
		dayShift--
	}
	for total >= minutesPerDay {
		total -= minutesPerDay
		dayShift++
	}

	d.startDate = d.startDate.AddDate(0, 0, dayShift)
	d.hourInput.value = total / 60
	d.minuteInput.value = total % 60
}

func (d *createEventDialog) currentStartDay(reference time.Time) time.Time {
	if !d.startDate.IsZero() {
		return beginningOfDay(d.startDate)
	}
	if reference.IsZero() {
		reference = time.Now()
	}
	return beginningOfDay(reference)
}

func (d *createEventDialog) adjustStartDays(delta int) {
	d.startDate = d.currentStartDay(time.Now()).AddDate(0, 0, delta)
}

func (d *createEventDialog) adjustStartMonths(delta int) {
	day := d.currentStartDay(time.Now())
	year, month, date := day.Date()
	targetMonth := int(month) + delta
	for targetMonth < 1 {
		targetMonth += 12
		year--
	}
	for targetMonth > 12 {
		targetMonth -= 12
		year++
	}
	if year < 1 {
		year = 1
		targetMonth = 1
	}
	targetDay := min(date, daysInMonth(year, time.Month(targetMonth)))
	d.startDate = time.Date(year, time.Month(targetMonth), targetDay, 0, 0, 0, 0, day.Location())
}

func (d *createEventDialog) adjustStartYears(delta int) {
	day := d.currentStartDay(time.Now())
	year, month, date := day.Date()
	targetYear := max(1, year+delta)
	targetDay := min(date, daysInMonth(targetYear, month))
	d.startDate = time.Date(targetYear, month, targetDay, 0, 0, 0, 0, day.Location())
}

func (d *createEventDialog) commitNumericInput() error {
	if !isDialogNumericFocus(d.focusIndex) || d.numericInputBuffer == "" {
		return nil
	}

	value, err := strconv.Atoi(d.numericInputBuffer)
	if err != nil {
		return fmt.Errorf("invalid numeric value %q", d.numericInputBuffer)
	}

	switch d.focusIndex {
	case dialogFocusMonth:
		err = d.setStartMonth(value)
	case dialogFocusDay:
		err = d.setStartDay(value)
	case dialogFocusYear:
		err = d.setStartYear(value)
	case dialogFocusHour:
		err = d.setStartHour(value)
	case dialogFocusMinute:
		err = d.setStartMinute(value)
	case dialogFocusDuration:
		err = d.setDurationMinutes(value)
	}
	if err != nil {
		return err
	}

	d.numericInputBuffer = ""
	return nil
}

func (d *createEventDialog) setStartMonth(value int) error {
	if value < 1 || value > 12 {
		return errors.New("month must be between 1 and 12")
	}

	day := d.currentStartDay(time.Now())
	year, _, date := day.Date()
	targetDay := min(date, daysInMonth(year, time.Month(value)))
	d.startDate = time.Date(year, time.Month(value), targetDay, 0, 0, 0, 0, day.Location())
	return nil
}

func (d *createEventDialog) setStartDay(value int) error {
	day := d.currentStartDay(time.Now())
	maxDay := daysInMonth(day.Year(), day.Month())
	if value < 1 || value > maxDay {
		return fmt.Errorf("day must be between 1 and %d", maxDay)
	}

	d.startDate = time.Date(day.Year(), day.Month(), value, 0, 0, 0, 0, day.Location())
	return nil
}

func (d *createEventDialog) setStartYear(value int) error {
	if value < 1 {
		return errors.New("year must be at least 1")
	}

	day := d.currentStartDay(time.Now())
	targetDay := min(day.Day(), daysInMonth(value, day.Month()))
	d.startDate = time.Date(value, day.Month(), targetDay, 0, 0, 0, 0, day.Location())
	return nil
}

func (d *createEventDialog) setStartHour(value int) error {
	if value < 0 || value > 23 {
		return errors.New("hour must be between 0 and 23")
	}

	d.hourInput.value = value
	return nil
}

func (d *createEventDialog) setStartMinute(value int) error {
	if value < 0 || value >= 60 || value%dialogMinuteStep != 0 {
		return fmt.Errorf("minutes must be in %d-minute increments", dialogMinuteStep)
	}

	d.minuteInput.value = value
	return nil
}

func (d *createEventDialog) setDurationMinutes(value int) error {
	if value < dialogMinuteStep {
		return fmt.Errorf("duration must be at least %d minutes", dialogMinuteStep)
	}
	if value%dialogMinuteStep != 0 {
		return fmt.Errorf("duration must be in %d-minute increments", dialogMinuteStep)
	}

	d.durationInput.value = value
	return nil
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func (d *createEventDialog) updateFocusedInput(msg tea.Msg) tea.Cmd {
	d.err = nil

	switch d.focusIndex {
	case dialogFocusTitle:
		var cmd tea.Cmd
		d.titleInput, cmd = d.titleInput.Update(msg)
		return cmd
	case dialogFocusMonth, dialogFocusDay, dialogFocusYear, dialogFocusHour, dialogFocusMinute, dialogFocusDuration:
		keyMsg, ok := msg.(tea.KeyMsg)
		if !ok {
			return nil
		}

		switch keyMsg.Type {
		case tea.KeyRunes:
			if len(keyMsg.Runes) != 1 || !unicode.IsDigit(keyMsg.Runes[0]) {
				return nil
			}
			if maxDigits := dialogNumericInputMaxDigits(d.focusIndex); maxDigits > 0 && len(d.numericInputBuffer) >= maxDigits {
				d.numericInputBuffer = ""
			}
			d.numericInputBuffer += string(keyMsg.Runes[0])
			return nil
		case tea.KeyBackspace, tea.KeyDelete:
			if d.numericInputBuffer != "" {
				d.numericInputBuffer = d.numericInputBuffer[:len(d.numericInputBuffer)-1]
			}
			return nil
		default:
			return nil
		}
	default:
		return nil
	}
}

func renderCreateEventDialog(dialog createEventDialog, width, height int) string {
	dialogWidth := dialogCols
	if width > 0 {
		dialogWidth = max(0, min(dialogWidth, width))
	}
	dialogHeight := dialogRows
	if height > 0 {
		dialogHeight = max(0, min(dialogHeight, height))
	}
	cardWidth := max(0, dialogWidth-2)
	cardHeight := max(0, dialogHeight-2)
	sectionWidth := max(0, cardWidth-(dialogPaddingX*2))
	content := []string{
		renderDialogSummarySection(dialog, sectionWidth),
		renderDialogHeaderSection(dialog, sectionWidth),
		renderDialogDateSection(dialog, sectionWidth),
		renderDialogScheduleSection(dialog, sectionWidth),
	}
	if dialog.err != nil {
		content = append(content, dialogErrorStyle.Width(sectionWidth).Render(dialog.err.Error()))
	}
	content = append(content, renderDialogButtons(dialog, sectionWidth))
	return dialogCardStyle.Width(cardWidth).Height(cardHeight).Render(strings.Join(content, "\n\n"))
}

func renderDialogSummarySection(dialog createEventDialog, width int) string {
	summaryText := dialogScheduleSummary(dialog)
	if width <= 0 {
		return dialogTitleStyle.Render(summaryText)
	}
	return dialogTitleStyle.Width(width).Align(lipgloss.Center).Render(summaryText)
}

func renderDialogHeaderSection(dialog createEventDialog, width int) string {
	titleInput := dialog.titleInput
	titleWidth := max(12, width)
	titleInput.Width = titleWidth - 1
	titleInput.SetCursor(titleInput.Position())
	return padRight(renderDialogTextInput(titleInput, titleWidth, dialog.focusIndex == dialogFocusTitle), width)
}

func renderDialogDateSection(dialog createEventDialog, width int) string {
	monthInput, dayInput, yearInput := dialog.dateInputs()
	dateControl := lipgloss.JoinHorizontal(
		lipgloss.Center,
		renderDialogNumberInput(monthInput, dialog.numericInputDisplay(dialogFocusMonth, monthInput.displayValue(), ""), dialog.focusIndex == dialogFocusMonth),
		dialogInlineTextStyle.Render("/"),
		renderDialogNumberInput(dayInput, dialog.numericInputDisplay(dialogFocusDay, dayInput.displayValue(), ""), dialog.focusIndex == dialogFocusDay),
		dialogInlineTextStyle.Render("/"),
		renderDialogNumberInput(yearInput, dialog.numericInputDisplay(dialogFocusYear, yearInput.displayValue(), ""), dialog.focusIndex == dialogFocusYear),
	)
	return renderDialogLabeledField("Date", dateControl, width)
}

func renderDialogScheduleSection(dialog createEventDialog, width int) string {
	timeControl := lipgloss.JoinHorizontal(
		lipgloss.Center,
		renderDialogNumberInput(dialog.hourInput, dialog.numericInputDisplay(dialogFocusHour, dialog.hourInput.displayValue(), ""), dialog.focusIndex == dialogFocusHour),
		dialogInlineTextStyle.Render(":"),
		renderDialogNumberInput(dialog.minuteInput, dialog.numericInputDisplay(dialogFocusMinute, dialog.minuteInput.displayValue(), ""), dialog.focusIndex == dialogFocusMinute),
	)
	durationControl := renderDialogDurationInput(dialog.durationInput, dialog.numericInputDisplay(dialogFocusDuration, fmt.Sprintf("%dm", dialog.durationInput.value), "m"), dialog.focusIndex == dialogFocusDuration)
	return strings.Join([]string{
		renderDialogLabeledField("Time", timeControl, width),
		renderDialogLabeledField("Duration", durationControl, width),
	}, "\n\n")
}

func renderDialogSplitRow(left, right string, width int) string {
	if right == "" {
		return padRight(ansi.Truncate(left, width, ""), width)
	}

	const gap = "  "
	if width > 0 && lipgloss.Width(left)+lipgloss.Width(gap)+lipgloss.Width(right) > width {
		return strings.Join([]string{
			padAnsiRight(ansi.Truncate(left, width, ""), width),
			padAnsiRight(ansi.Truncate(right, width, ""), width),
		}, "\n")
	}

	leftWidth := max(0, width-lipgloss.Width(right)-lipgloss.Width(gap))
	return padRight(ansi.Truncate(left, leftWidth, "…"), leftWidth) + gap + right
}

func dialogScheduleSummary(dialog createEventDialog) string {
	start, end := dialog.previewTimes()
	return start.Format("Jan 2, 2006 3:04PM") + " – " + end.Format("3:04PM")
}

func (d createEventDialog) dateInputs() (dialogNumberInput, dialogNumberInput, dialogNumberInput) {
	startDay := d.startDate
	if startDay.IsZero() {
		startDay = beginningOfDay(time.Now())
	}
	monthInput := newDialogNumberInput(int(startDay.Month()), 1, 12, 1, 4, formatDialogMonthValue)
	dayInput := newDialogNumberInput(startDay.Day(), 1, daysInMonth(startDay.Year(), startDay.Month()), 1, 4, formatTwoDigitNumber)
	yearInput := newDialogNumberInput(startDay.Year(), 1, -1, 1, 6, formatDialogYearValue)
	return monthInput, dayInput, yearInput
}

func (d createEventDialog) previewTimes() (time.Time, time.Time) {
	startDay := d.startDate
	if startDay.IsZero() {
		startDay = beginningOfDay(time.Now())
	}
	start := time.Date(
		startDay.Year(),
		startDay.Month(),
		startDay.Day(),
		d.hourInput.value,
		d.minuteInput.value,
		0,
		0,
		startDay.Location(),
	)
	return start, start.Add(time.Duration(d.durationInput.value) * time.Minute)
}

func (d createEventDialog) numericInputDisplay(focusIndex int, fallback, suffix string) string {
	if d.focusIndex == focusIndex && d.numericInputBuffer != "" {
		return d.numericInputBuffer + suffix
	}
	return fallback
}

func renderDialogTextInput(input textinput.Model, width int, focused bool) string {
	if focused {
		return padAnsiRight(ansi.Truncate(input.View(), width, ""), width)
	}

	display := strings.TrimRight(ansi.Strip(input.View()), " ")
	if input.Value() == "" {
		display = input.Placeholder
	}
	return padRight(ansi.Truncate(display, width, ""), width)
}

func renderDialogLabeledField(label, control string, width int) string {
	label = dialogFieldLabelStyle.Render(label)
	if width <= 0 {
		width = max(lipgloss.Width(label), lipgloss.Width(control))
	}
	return renderDialogSplitRow(label, control, width)
}

func renderDialogNumberInput(input dialogNumberInput, display string, focused bool) string {
	style := dialogInputStyle
	if focused {
		style = dialogFocusedInputStyle
	}

	return style.Width(max(input.width, lipgloss.Width(display)+style.GetHorizontalFrameSize())).Align(lipgloss.Center).Render(display)
}

func renderDialogDurationInput(input dialogNumberInput, display string, focused bool) string {
	style := dialogInputStyle
	if focused {
		style = dialogFocusedInputStyle
	}

	return style.Width(max(input.width+1, lipgloss.Width(display)+style.GetHorizontalFrameSize())).Align(lipgloss.Center).Render(display)
}

func renderDialogButtons(dialog createEventDialog, width int) string {
	submitLabel := "Create"
	if dialog.submitting {
		submitLabel = "Creating…"
	}

	submitStyle := dialogButtonStyle
	cancelStyle := dialogButtonStyle
	if dialog.focusIndex == dialogFocusSubmit {
		submitStyle = dialogFocusedButtonStyle
	}
	if dialog.focusIndex == dialogFocusCancel {
		cancelStyle = dialogFocusedButtonStyle
	}

	buttonRowStyle := lipgloss.NewStyle().Width(width).Align(lipgloss.Right)
	return buttonRowStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			submitStyle.Render(submitLabel),
			"  ",
			cancelStyle.Render("Cancel"),
		),
	)
}
