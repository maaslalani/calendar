package main

import (
	"strings"
	"testing"

	ical "github.com/BRO3886/go-eventkit/calendar"
	"github.com/charmbracelet/lipgloss"
)

func TestRenderCurrentTimeActiveLineJoinsEventAccent(t *testing.T) {
	tests := []struct {
		name     string
		accent   string
		junction string
	}{
		{name: "full", accent: eventAccent, junction: "╂"},
		{name: "short", accent: shortEventAccent, junction: "┼"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			block := &timedEventBlock{
				event:  ical.Event{CalendarID: demoWorkCalendarID},
				accent: test.accent,
			}
			line := renderCurrentTimeActiveLine(
				[]*timedEventBlock{block},
				map[string]string{demoWorkCalendarID: "#7C3AED"},
				timedEventLayout{columnWidths: []int{8}},
				lipgloss.NewStyle(),
			)

			if !strings.Contains(line, test.junction) {
				t.Fatalf("renderCurrentTimeActiveLine() = %q, want junction %q", line, test.junction)
			}
			if width := lipgloss.Width(line); width != 8 {
				t.Fatalf("lipgloss.Width(renderCurrentTimeActiveLine()) = %d, want 8", width)
			}
		})
	}
}
