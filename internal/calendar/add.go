package calendar

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

// AddCommand returns the `cal add` subcommand for creating events.
func AddCommand() *cobra.Command {
	var (
		date         string
		startTime    string
		duration     string
		allDay       bool
		calendarName string
		location     string
		notes        string
	)

	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a calendar event",
		Example: `  cal add "Lunch with Will" --time 12:30
  cal add Standup --date tomorrow --time 9:15 --duration 15m
  cal add Launch --date 2026-08-01 --all-day`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if calendarDemoEnabled() {
				return errors.New("event creation is unavailable in demo mode")
			}

			input, err := buildCreateEventInput(createEventOptions{
				title:        strings.Join(args, " "),
				date:         date,
				startTime:    startTime,
				timeSet:      cmd.Flags().Changed("time"),
				duration:     duration,
				durationSet:  cmd.Flags().Changed("duration"),
				allDay:       allDay,
				calendarName: calendarName,
				location:     location,
				notes:        notes,
			}, time.Now())
			if err != nil {
				return err
			}

			client, err := ical.New()
			if err != nil {
				return errors.New(friendlyError(err))
			}

			event, err := client.CreateEvent(input)
			if err != nil {
				return err
			}

			if !term.IsTerminal(os.Stdout.Fd()) {
				fmt.Fprintln(cmd.OutOrStdout(), formatCreatedEvent(*event))
				return nil
			}

			var colors map[string]string
			if calendars, err := client.Calendars(); err == nil {
				colors, _ = buildCalendarDecorations(calendars, []ical.Event{*event})
			}
			fmt.Fprintln(cmd.OutOrStdout(), renderCreatedEventPretty(*event, colors, time.Now()))
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&date, "date", "d", "today", `event date: "today", "tomorrow", "yesterday", or YYYY-MM-DD`)
	flags.StringVarP(&startTime, "time", "t", "", `start time, e.g. "14:30" or "2:30pm" (default: next quarter hour)`)
	flags.StringVar(&duration, "duration", "30m", `event duration, e.g. "45m", "1h30m", or minutes`)
	flags.BoolVar(&allDay, "all-day", false, "create an all-day event")
	flags.StringVarP(&calendarName, "calendar", "c", "", "calendar to add the event to (default: system default)")
	flags.StringVarP(&location, "location", "l", "", "event location")
	flags.StringVar(&notes, "notes", "", "event notes")
	return cmd
}

type createEventOptions struct {
	title        string
	date         string
	startTime    string
	timeSet      bool
	duration     string
	durationSet  bool
	allDay       bool
	calendarName string
	location     string
	notes        string
}

func buildCreateEventInput(opts createEventOptions, now time.Time) (ical.CreateEventInput, error) {
	title := strings.TrimSpace(opts.title)
	if title == "" {
		return ical.CreateEventInput{}, errors.New("title is required")
	}

	day, err := parseEventDate(opts.date, now)
	if err != nil {
		return ical.CreateEventInput{}, err
	}

	input := ical.CreateEventInput{
		Title:    title,
		Calendar: opts.calendarName,
		Location: opts.location,
		Notes:    opts.notes,
	}

	if opts.allDay {
		if opts.timeSet || opts.durationSet {
			return ical.CreateEventInput{}, errors.New("cannot combine --all-day with --time or --duration")
		}
		input.AllDay = true
		input.StartDate = day
		input.EndDate = day.AddDate(0, 0, 1)
		return input, nil
	}

	start, err := eventStartTime(day, opts.startTime, now)
	if err != nil {
		return ical.CreateEventInput{}, err
	}

	eventDuration, err := parseEventDuration(opts.duration)
	if err != nil {
		return ical.CreateEventInput{}, err
	}

	input.StartDate = start
	input.EndDate = start.Add(eventDuration)
	return input, nil
}

func parseEventDate(value string, now time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", "today":
		return beginningOfDay(now), nil
	case "tomorrow":
		return beginningOfDay(now).AddDate(0, 0, 1), nil
	case "yesterday":
		return beginningOfDay(now).AddDate(0, 0, -1), nil
	}

	day, err := time.ParseInLocation(time.DateOnly, value, now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (use \"today\", \"tomorrow\", or YYYY-MM-DD)", value)
	}
	return day, nil
}

func eventStartTime(day time.Time, value string, now time.Time) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nextQuarterHour(wallClockTimeOnDay(day, now)), nil
	}

	hour, minute, err := parseEventTime(value)
	if err != nil {
		return time.Time{}, err
	}

	start := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
	if start.Hour() != hour || start.Minute() != minute {
		return time.Time{}, errors.New("that start time does not exist in your local timezone on that date")
	}
	return start, nil
}

var eventTimeLayouts = []string{"15:04", "15", "3:04PM", "3PM"}

func parseEventTime(value string) (int, int, error) {
	cleaned := strings.ToUpper(strings.Join(strings.Fields(value), ""))
	for _, layout := range eventTimeLayouts {
		if parsed, err := time.Parse(layout, cleaned); err == nil {
			return parsed.Hour(), parsed.Minute(), nil
		}
	}
	return 0, 0, fmt.Errorf("invalid time %q (use e.g. \"14:30\" or \"2:30pm\")", value)
}

func parseEventDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	duration, err := time.ParseDuration(value)
	if err != nil {
		minutes, convErr := strconv.Atoi(value)
		if convErr != nil {
			return 0, fmt.Errorf("invalid duration %q (use e.g. \"45m\", \"1h30m\", or minutes)", value)
		}
		duration = time.Duration(minutes) * time.Minute
	}

	if duration <= 0 {
		return 0, errors.New("duration must be positive")
	}
	return duration, nil
}

func formatCreatedEvent(event ical.Event) string {
	line := fmt.Sprintf("Created %q: %s", displayEventTitle(event), formatEventSchedule(event))
	if event.Calendar != "" {
		line += " (" + event.Calendar + ")"
	}
	return line
}

func renderCreatedEventPretty(event ical.Event, calendarColors map[string]string, now time.Time) string {
	color := eventCalendarColor(event, calendarColors)
	accent := colorStyle(color).Render(eventAccent)
	bar := colorStyle(color).Render(strings.TrimRight(eventAccent, " "))

	title := dialogTitleStyle.Render(displayEventTitle(event))
	if marker := displayEventMarker(event); marker != "" {
		title += strings.Repeat(" ", recurringMarkerPad) + recurringMarkerStyle(color, "").Render(marker)
	}

	lines := []string{
		successMarkStyle.Render("✓ ") + dialogTitleStyle.Render("Created"),
		"",
		accent + title,
		bar,
	}
	for _, field := range createdEventFields(event, color, now) {
		lines = append(lines, accent+field)
	}
	return strings.Join(lines, "\n")
}

const createdNotesMaxWidth = 48

func createdEventFields(event ical.Event, color string, now time.Time) []string {
	start := event.StartDate.Local()

	timeStyle := lipgloss.NewStyle()
	timeValue := "all day"
	if !event.AllDay {
		end := event.EndDate.Local()
		timeValue = formatClockRange(event, time.Local)
		if !beginningOfDay(start).Equal(beginningOfDay(end)) {
			timeValue += end.Format(" (Jan 2)")
		}
		if eventOngoing(event, now) {
			timeStyle = currentTimeLineStyle.Bold(true)
		}
	}

	fields := [][2]string{
		{"Date", start.Format("Monday, January 2, 2006")},
		{"Time", timeStyle.Render(timeValue)},
	}
	if event.Location != "" {
		fields = append(fields, [2]string{"Location", event.Location})
	}
	if notes := strings.Join(strings.Fields(event.Notes), " "); notes != "" {
		fields = append(fields, [2]string{"Notes", ansi.Truncate(notes, createdNotesMaxWidth, "…")})
	}
	if event.Calendar != "" {
		fields = append(fields, [2]string{"Calendar", colorStyle(color).Render(event.Calendar)})
	}

	labelWidth := 0
	for _, field := range fields {
		labelWidth = max(labelWidth, lipgloss.Width(field[0]))
	}

	lines := make([]string, 0, len(fields))
	for _, field := range fields {
		lines = append(lines, dialogFieldLabelStyle.Render(padRight(field[0], labelWidth))+"  "+field[1])
	}
	return lines
}

func formatEventSchedule(event ical.Event) string {
	const dayFormat = "Mon, Jan 2 2006"
	start := event.StartDate.Local()
	end := event.EndDate.Local()

	if event.AllDay {
		return start.Format(dayFormat) + ", all day"
	}
	if beginningOfDay(start).Equal(beginningOfDay(end)) {
		return fmt.Sprintf("%s, %s – %s", start.Format(dayFormat), start.Format("3:04 PM"), end.Format("3:04 PM"))
	}
	return fmt.Sprintf("%s %s – %s %s", start.Format(dayFormat), start.Format("3:04 PM"), end.Format(dayFormat), end.Format("3:04 PM"))
}
