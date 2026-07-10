package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

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
