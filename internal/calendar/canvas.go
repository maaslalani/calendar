package calendar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

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
		if target >= len(baseLines) {
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
