package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

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
	monthInput := newDialogNumberInput(int(startDay.Month()), 1, 12, 1, 4, formatDialogMonthValue)
	dayInput := newDialogNumberInput(startDay.Day(), 1, daysInMonth(startDay.Year(), startDay.Month()), 1, 4, formatDialogClockValue)
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
