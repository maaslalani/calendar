package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
		hourInput:     newDialogNumberInput(start.Hour(), 0, 23, 1, 4, formatDialogClockValue),
		minuteInput:   newDialogNumberInput(start.Minute(), 0, 45, dialogMinuteStep, 4, formatDialogClockValue),
		durationInput: newDialogNumberInput(durationMinutes, dialogMinuteStep, -1, dialogMinuteStep, 4, nil),
		focusIndex:    dialogFocusTitle,
	}

	return dialog, dialog.syncFocus()
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

		_, err = client.CreateEvent(input)
		return createEventMsg{err: err}
	}
}
