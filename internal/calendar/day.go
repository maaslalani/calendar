package calendar

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// ViewCommand returns the `cal view` subcommand for listing a day's events.
func ViewCommand() *cobra.Command {
	var color string

	cmd := &cobra.Command{
		Use:   "view [date]",
		Short: "View a day's events",
		Long: "View a day's events.\n\n" +
			`The date can be "today", "tomorrow", "yesterday", or YYYY-MM-DD, and` + "\n" +
			"defaults to today.\n\n" +
			"Output is styled when attached to a terminal and plain when piped;\n" +
			"use --color always or --color never to force either.",
		Example: `  cal view
  cal view tomorrow
  cal view 2026-08-01`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			isTTY := term.IsTerminal(os.Stdout.Fd())
			colored, err := resolveColorMode(color, isTTY)
			if err != nil {
				return err
			}

			dateArg := "today"
			if len(args) == 1 {
				dateArg = args[0]
			}

			now := time.Now()
			day, err := parseEventDate(dateArg, now)
			if err != nil {
				return err
			}

			data, err := loadCalendar(day, now)
			if err != nil {
				return errors.New(friendlyError(err))
			}

			section, ok := sectionForDay(data, day)
			if !ok {
				return fmt.Errorf("could not load events for %s", day.Format(time.DateOnly))
			}

			if !colored {
				fmt.Fprint(cmd.OutOrStdout(), renderDayPlain(section))
				return nil
			}

			if !isTTY {
				// Keep styles when colored output is forced into a pipe.
				lipgloss.SetColorProfile(termenv.TrueColor)
			}
			fmt.Fprintln(cmd.OutOrStdout(), renderDayPretty(section, data.calendarColors, data.currentTime))
			return nil
		},
	}

	cmd.Flags().StringVar(&color, "color", "auto", `colorize output: "auto", "always", or "never"`)
	return cmd
}

// resolveColorMode reports whether output should be colored for the given
// --color value: "always", "never", or "auto" (color only when on a terminal).
func resolveColorMode(mode string, isTTY bool) (bool, error) {
	switch mode {
	case "auto":
		return isTTY, nil
	case "always":
		return true, nil
	case "never":
		return false, nil
	}
	return false, fmt.Errorf("invalid --color %q (use \"auto\", \"always\", or \"never\")", mode)
}

func sectionForDay(data calendarData, day time.Time) (daySection, bool) {
	for _, section := range data.sections {
		if section.date.Equal(day) {
			return section, true
		}
	}
	return daySection{}, false
}

const plainTimeColumnWidth = len("00:00 - 00:00")

func renderDayPlain(section daySection) string {
	if len(section.events) == 0 {
		return ""
	}

	titleWidth := 0
	for _, event := range section.events {
		titleWidth = max(titleWidth, lipgloss.Width(displayEventTitle(event)))
	}

	allDay, timed := partitionDayEvents(section.events)
	lines := make([]string, 0, len(section.events))
	for _, event := range allDay {
		lines = append(lines, plainEventLine("all day", event, titleWidth))
	}
	for _, event := range timed {
		start := event.StartDate.In(section.date.Location())
		end := event.EndDate.In(section.date.Location())
		lines = append(lines, plainEventLine(start.Format("15:04")+" - "+end.Format("15:04"), event, titleWidth))
	}

	return strings.Join(lines, "\n") + "\n"
}

func plainEventLine(timeText string, event ical.Event, titleWidth int) string {
	line := padRight(timeText, plainTimeColumnWidth) + "  " + padRight(displayEventTitle(event), titleWidth)
	if event.Calendar != "" {
		line += "  [" + event.Calendar + "]"
	}
	return strings.TrimRight(line, " ")
}

type dayRow struct {
	event    ical.Event
	timeText string
	ongoing  bool
}

func renderDayPretty(section daySection, calendarColors map[string]string, now time.Time) string {
	label := relativeDayLabel(section.date, now)
	header := dialogTitleStyle.Render(label) + primaryTextStyle.Render(section.date.Format(" · Monday, January 2, 2006"))
	if len(section.events) == 0 {
		return header + "\n\n" + allDayMoreStyle.Render(fmt.Sprintf("No events %s.", lowerFirst(label)))
	}

	allDay, timed := partitionDayEvents(section.events)
	rows := make([]dayRow, 0, len(section.events))
	for _, event := range allDay {
		rows = append(rows, dayRow{event: event, timeText: "all day"})
	}
	for _, event := range timed {
		rows = append(rows, dayRow{
			event:    event,
			timeText: formatClockRange(event, section.date.Location()),
			ongoing:  eventOngoing(event, now),
		})
	}

	timeWidth := 0
	for _, row := range rows {
		timeWidth = max(timeWidth, lipgloss.Width(row.timeText))
	}

	lines := []string{header, ""}
	for _, row := range rows {
		lines = append(lines, renderDayPrettyRow(row, calendarColors, timeWidth))
	}
	if legend := renderLegend(dayLegend(section.events, calendarColors)); legend != "" {
		lines = append(lines, "", legend)
	}
	return strings.Join(lines, "\n")
}

func renderDayPrettyRow(row dayRow, calendarColors map[string]string, timeWidth int) string {
	color := eventCalendarColor(row.event, calendarColors)

	timeStyle := primaryTextStyle
	if row.ongoing {
		timeStyle = currentTimeLineStyle.Bold(true)
	}

	title := colorStyle(color).Render(displayEventTitle(row.event))
	if marker := displayEventMarker(row.event); marker != "" {
		title += strings.Repeat(" ", recurringMarkerPad) + recurringMarkerStyle(color, "").Render(marker)
	}

	return timeStyle.Render(padRight(row.timeText, timeWidth)) + "  " + title
}

func dayLegend(events []ical.Event, calendarColors map[string]string) []calendarLegendItem {
	byLabel := make(map[string]calendarLegendItem, len(events))
	for _, event := range events {
		label := fallbackCalendarLabel(event)
		byLabel[label] = calendarLegendItem{
			key:   label,
			label: label,
			color: eventCalendarColor(event, calendarColors),
		}
	}

	items := make([]calendarLegendItem, 0, len(byLabel))
	for _, item := range byLabel {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].label < items[j].label })
	return items
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func formatClockRange(event ical.Event, loc *time.Location) string {
	start := event.StartDate.In(loc)
	end := event.EndDate.In(loc)
	return start.Format("15:04") + " – " + end.Format("15:04")
}

func eventOngoing(event ical.Event, now time.Time) bool {
	return !now.Before(event.StartDate) && now.Before(event.EndDate)
}
