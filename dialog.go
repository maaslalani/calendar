package main

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

type dialogNumberInput struct {
	value  int
	min    int
	max    int
	step   int
	width  int
	wrap   bool
	hasMax bool
	format func(int) string
}

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
		if m.watchCancel != nil {
			m.watchCancel()
			m.watchCancel = nil
			m.watchChanges = nil
		}
		return m, tea.Quit
	}

	switch msg.String() {
	case "esc":
		if m.createDialog.submitting {
			return m, nil
		}
		m.showCreateDialog = false
		m.createDialog = createEventDialog{}
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
		switch m.createDialog.focusIndex {
		case dialogFocusDay:
			return m, m.createDialog.setFocus(dialogFocusMonth)
		case dialogFocusYear:
			return m, m.createDialog.setFocus(dialogFocusDay)
		case dialogFocusHour:
			return m, m.createDialog.setFocus(dialogFocusYear)
		case dialogFocusMinute:
			return m, m.createDialog.setFocus(dialogFocusHour)
		case dialogFocusDuration:
			return m, m.createDialog.setFocus(dialogFocusMinute)
		case dialogFocusCancel:
			return m, m.createDialog.setFocus(dialogFocusSubmit)
		}
	case "right":
		switch m.createDialog.focusIndex {
		case dialogFocusMonth:
			return m, m.createDialog.setFocus(dialogFocusDay)
		case dialogFocusDay:
			return m, m.createDialog.setFocus(dialogFocusYear)
		case dialogFocusYear:
			return m, m.createDialog.setFocus(dialogFocusHour)
		case dialogFocusHour:
			return m, m.createDialog.setFocus(dialogFocusMinute)
		case dialogFocusMinute:
			return m, m.createDialog.setFocus(dialogFocusDuration)
		case dialogFocusSubmit:
			return m, m.createDialog.setFocus(dialogFocusCancel)
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
		m.showCreateDialog = false
		m.createDialog = createEventDialog{}
		return m, nil
	default:
		return m, m.createDialog.moveFocus(1)
	}
}

func newCreateEventDialog(now time.Time) (createEventDialog, tea.Cmd) {
	return newCreateEventDialogForDay(now, now)
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

// newCreateEventDialogAt builds a create dialog pre-filled with an explicit
// start time and duration in minutes. It backs both the keyboard ("n") and the
// mouse drag-to-create entry points.
func newCreateEventDialogAt(start time.Time, durationMinutes int) (createEventDialog, tea.Cmd) {
	if start.IsZero() {
		start = nextQuarterHour(time.Now())
	}
	if durationMinutes < dialogMinuteStep {
		durationMinutes = dialogMinuteStep
	}

	dialog := createEventDialog{
		titleInput:    newDialogTextInput("Title", "", 24),
		startDate:     beginningOfDay(start),
		hourInput:     newDialogNumberInput(start.Hour(), 0, 23, 1, 4, false, formatDialogClockValue),
		minuteInput:   newDialogNumberInput(start.Minute(), 0, 45, dialogMinuteStep, 4, false, formatDialogClockValue),
		durationInput: newDialogNumberInput(durationMinutes, dialogMinuteStep, -1, dialogMinuteStep, 4, false, nil),
		focusIndex:    dialogFocusTitle,
	}

	return dialog, dialog.syncFocus()
}

func nextDialogFocus(current, delta int) int {
	next := current + delta
	for next < 0 {
		next += dialogFocusCount
	}
	return next % dialogFocusCount
}

func newDialogTextInput(placeholder, value string, width int) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.Prompt = ""
	input.Width = width
	if value != "" {
		input.SetValue(value)
	}
	return input
}

func newDialogNumberInput(value, minValue, maxValue, step, width int, wrap bool, format func(int) string) dialogNumberInput {
	input := dialogNumberInput{
		value:  value,
		min:    minValue,
		step:   step,
		width:  width,
		wrap:   wrap,
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
		if i.wrap {
			span := i.max - i.min + 1
			if span <= 0 {
				i.value = i.min
				return
			}
			offset := (next - i.min) % span
			if offset < 0 {
				offset += span
			}
			i.value = i.min + offset
			return
		}
		i.value = min(i.max, max(i.min, next))
		return
	}
	if next < i.min {
		next = i.min
	}
	i.value = next
}

func formatDialogClockValue(value int) string {
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
	} else {
		d.titleInput.Blur()
	}
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
	return func() tea.Msg {
		client, err := ical.New()
		if err != nil {
			return createEventMsg{err: err}
		}

		event, err := client.CreateEvent(input)
		return createEventMsg{
			event: event,
			err:   err,
		}
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
		return dialogSectionStyle.Render(dialogTitleStyle.Render(summaryText))
	}
	return dialogSectionStyle.Render(dialogTitleStyle.Width(width).Align(lipgloss.Center).Render(summaryText))
}

func renderDialogHeaderSection(dialog createEventDialog, width int) string {
	titleInput := dialog.titleInput
	titleWidth := max(12, width)
	titleInput.Width = titleWidth
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
	return dialogSectionStyle.Render(renderDialogLabeledField("Date", dateControl, width))
}

func renderDialogScheduleSection(dialog createEventDialog, width int) string {
	timeControl := lipgloss.JoinHorizontal(
		lipgloss.Center,
		renderDialogNumberInput(dialog.hourInput, dialog.numericInputDisplay(dialogFocusHour, dialog.hourInput.displayValue(), ""), dialog.focusIndex == dialogFocusHour),
		dialogInlineTextStyle.Render(":"),
		renderDialogNumberInput(dialog.minuteInput, dialog.numericInputDisplay(dialogFocusMinute, dialog.minuteInput.displayValue(), ""), dialog.focusIndex == dialogFocusMinute),
	)
	durationControl := renderDialogDurationInput(dialog.durationInput, dialog.numericInputDisplay(dialogFocusDuration, fmt.Sprintf("%dm", dialog.durationInput.value), "m"), dialog.focusIndex == dialogFocusDuration)
	return dialogSectionStyle.Render(strings.Join([]string{
		renderDialogLabeledField("Time", timeControl, width),
		renderDialogLabeledField("Duration", durationControl, width),
	}, "\n\n"))
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
	monthInput := newDialogNumberInput(int(startDay.Month()), 1, 12, 1, 4, false, formatDialogMonthValue)
	dayInput := newDialogNumberInput(startDay.Day(), 1, daysInMonth(startDay.Year(), startDay.Month()), 1, 4, false, formatDialogClockValue)
	yearInput := newDialogNumberInput(startDay.Year(), 1, -1, 1, 6, false, formatDialogYearValue)
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

func renderDialogFieldLabel(label string) string {
	return dialogFieldLabelStyle.Render(label)
}

func renderDialogLabeledField(label, control string, width int) string {
	if width <= 0 {
		width = max(lipgloss.Width(renderDialogFieldLabel(label)), lipgloss.Width(control))
	}
	return renderDialogSplitRow(renderDialogFieldLabel(label), control, width)
}

func renderDialogNumberInput(input dialogNumberInput, display string, focused bool) string {
	style := dialogInputStyle
	if focused {
		style = dialogFocusedInputStyle
	}

	if display == "" {
		display = input.displayValue()
	}
	return style.Width(max(input.width, lipgloss.Width(display)+style.GetHorizontalFrameSize())).Align(lipgloss.Center).Render(display)
}

func renderDialogDurationInput(input dialogNumberInput, display string, focused bool) string {
	style := dialogInputStyle
	if focused {
		style = dialogFocusedInputStyle
	}

	if display == "" {
		display = fmt.Sprintf("%dm", input.value)
	}
	return style.Width(max(input.width+1, lipgloss.Width(display)+style.GetHorizontalFrameSize())).Align(lipgloss.Center).Render(display)
}

func renderDialogButtons(dialog createEventDialog, width int) string {
	submitLabel := "Create"
	if dialog.submitting {
		submitLabel = "Creating…"
	}

	submitStyle := dialogPrimaryButtonStyle
	cancelStyle := dialogSecondaryButtonStyle
	if dialog.focusIndex == dialogFocusSubmit {
		submitStyle = dialogFocusedPrimaryButtonStyle
	}
	if dialog.focusIndex == dialogFocusCancel {
		cancelStyle = dialogFocusedSecondaryButtonStyle
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

func compositeLayer(base, overlay string, width, height int) string {
	if overlay == "" {
		return base
	}

	if width <= 0 {
		width = max(ansi.StringWidth(base), lipgloss.Width(overlay))
	}
	if height <= 0 {
		height = max(strings.Count(base, "\n")+1, lipgloss.Height(overlay))
	}

	baseLines := canvasLines(base, width, height)
	overlayWidth := lipgloss.Width(overlay)
	overlayHeight := lipgloss.Height(overlay)
	x := max(0, (width-overlayWidth)/2)
	y := max(0, (height-overlayHeight)/2)
	overlayLines := canvasLines(overlay, overlayWidth, overlayHeight)

	for i, overlayLine := range overlayLines {
		target := y + i
		if target < 0 || target >= len(baseLines) {
			continue
		}

		baseLine := baseLines[target]
		left := ansi.Cut(baseLine, 0, x)
		right := ansi.Cut(baseLine, x+overlayWidth, width)
		baseLines[target] = left + overlayLine + right
	}

	return strings.Join(baseLines, "\n")
}

func canvasLines(content string, width, height int) []string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	if height > 0 {
		if len(lines) > height {
			lines = lines[:height]
		}
		for len(lines) < height {
			lines = append(lines, "")
		}
	}

	for i, line := range lines {
		if width > 0 {
			line = ansi.Truncate(line, width, "")
		}
		lines[i] = padAnsiRight(line, width)
	}

	return lines
}

func padAnsiRight(text string, width int) string {
	if width <= 0 {
		return text
	}
	if padding := width - ansi.StringWidth(text); padding > 0 {
		return text + strings.Repeat(" ", padding)
	}
	return text
}
