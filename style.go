package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	timelineMutedColor   = lipgloss.AdaptiveColor{Light: "#C4C4C4", Dark: "#454545"}
	timelineActiveColor  = lipgloss.AdaptiveColor{Light: "#A8A8A8", Dark: "#6A6A6A"}
	currentTimeColor     = lipgloss.Color("#FF5F5F")
	currentTimeEdgeColor = lipgloss.Color(scaleHexColor(string(currentTimeColor), 40, 0, "#662626"))
	dialogMutedColor     = lipgloss.AdaptiveColor{Light: "#64748B", Dark: "#8A94A6"}
	dialogBorderColor    = lipgloss.AdaptiveColor{Light: "#94A3B8", Dark: "#3F4A59"}
	dialogTitleColor     = lipgloss.AdaptiveColor{Light: "#0F172A", Dark: "#F8FAFC"}
	dialogButtonColor    = lipgloss.AdaptiveColor{Light: "#4B5563", Dark: "#374151"}
	dialogButtonText     = lipgloss.AdaptiveColor{Light: "#F8FAFC", Dark: "#F8FAFC"}
	timeAxisStyle        = lipgloss.NewStyle().Foreground(timelineMutedColor).Width(timeColumnWidth).Align(lipgloss.Right)
	activeTimeAxisStyle  = lipgloss.NewStyle().Foreground(timelineActiveColor).Width(timeColumnWidth).Align(lipgloss.Right)
	currentTimeAxisStyle = lipgloss.NewStyle().Foreground(currentTimeColor).Bold(true).Width(timeColumnWidth).Align(lipgloss.Right)
	dayCellStyle         = lipgloss.NewStyle().Width(dayColumnWidth)
	allDayMoreStyle      = lipgloss.NewStyle().Foreground(timelineMutedColor)
	headerCellStyle      = lipgloss.NewStyle().Bold(true).Width(dayColumnWidth).Align(lipgloss.Center)
	dateCellStyle        = lipgloss.NewStyle().Faint(true).Width(dayColumnWidth).Align(lipgloss.Center)
	columnSeparator      = lipgloss.NewStyle().Render("  ")
	currentTimeLineStyle = lipgloss.NewStyle().Foreground(currentTimeColor)
	currentTimeEdgeStyle = lipgloss.NewStyle().Foreground(currentTimeEdgeColor)
	screenStyle          = lipgloss.NewStyle().Padding(screenPaddingY, screenPaddingX)
	dialogCardStyle      = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(dialogBorderColor).
				Padding(dialogPaddingY, dialogPaddingX)
	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(dialogTitleColor)
	dialogFieldLabelStyle = lipgloss.NewStyle().
				Foreground(dialogMutedColor)
	dialogInlineTextStyle = lipgloss.NewStyle().
				Foreground(dialogMutedColor)
	dialogInputStyle = lipgloss.NewStyle().
				Foreground(dialogTitleColor).
				Padding(0, 1)
	dialogFocusedInputStyle = lipgloss.NewStyle().
				Foreground(currentTimeColor).
				Bold(true).
				Padding(0, 1)
	dialogButtonStyle = lipgloss.NewStyle().
				Background(dialogButtonColor).
				Foreground(dialogButtonText).
				Bold(true).
				Padding(0, 2)
	dialogFocusedButtonStyle = lipgloss.NewStyle().
					Background(currentTimeColor).
					Foreground(dialogButtonText).
					Underline(true).
					Bold(true).
					Padding(0, 2)
	dialogErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FCA5A5")).
				BorderLeft(true).
				BorderForeground(lipgloss.Color("#F87171")).
				PaddingLeft(1)

	fallbackCalendarColors = []string{
		"#60A5FA",
		"#F97316",
		"#A78BFA",
		"#34D399",
		"#F472B6",
		"#FBBF24",
		"#22D3EE",
		"#FB7185",
	}
)

func colorStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

func eventBackgroundStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(color))
}

func renderLegendSwatch(color string) string {
	return colorStyle(color).Render("●")
}

func eventBackgroundColor(color string) string {
	return scaleHexColor(color, 28, 8, "#080808")
}

func recurringMarkerColor(color string) string {
	if scaled := scaleHexColor(color, recurringMarkerScale, 8, ""); scaled != "" {
		return scaled
	}
	return color
}

func recurringMarkerStyle(color, backgroundColor string) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(recurringMarkerColor(color)))
	if backgroundColor != "" {
		style = style.Background(lipgloss.Color(backgroundColor))
	}
	return style
}

func scaleHexColor(color string, scale, minChannel int, fallback string) string {
	hex := strings.TrimPrefix(strings.TrimSpace(color), "#")
	if len(hex) != 6 {
		return fallback
	}

	channels := [3]int{}
	for i := range channels {
		value, err := strconv.ParseInt(hex[i*2:i*2+2], 16, 0)
		if err != nil {
			return fallback
		}
		channels[i] = max(minChannel, int(value)*scale/100)
	}

	return fmt.Sprintf("#%02X%02X%02X", channels[0], channels[1], channels[2])
}
