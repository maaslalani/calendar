package main

import (
	"errors"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
)

func beginningOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func wallClockTimeOnDay(day, t time.Time) time.Time {
	local := t.In(day.Location())
	year, month, date := day.Date()
	return time.Date(year, month, date, local.Hour(), local.Minute(), local.Second(), local.Nanosecond(), day.Location())
}

func wallClockOffset(dayStart, t time.Time) time.Duration {
	local := t.In(dayStart.Location())
	switch compareLocalDate(local, dayStart) {
	case -1:
		return 0
	case 1:
		return 24 * time.Hour
	}

	return time.Duration(local.Hour())*time.Hour +
		time.Duration(local.Minute())*time.Minute +
		time.Duration(local.Second())*time.Second +
		time.Duration(local.Nanosecond())
}

func wallClockSlot(dayStart, t time.Time, roundUp bool) int {
	offset := wallClockOffset(dayStart, t)
	slot := int(offset / slotDuration)
	if roundUp && offset%slotDuration != 0 {
		slot++
	}
	return min(slotsPerDay, max(0, slot))
}

func timeFallsOnSlotBoundary(dayStart, t time.Time, slot int) bool {
	local := t.In(dayStart.Location())
	if local.Minute()%30 != 0 || local.Second() != 0 || local.Nanosecond() != 0 {
		return false
	}

	switch compareLocalDate(local, dayStart) {
	case -1:
		return false
	case 1:
		return slot == slotsPerDay && local.Hour() == 0 && local.Minute() == 0
	default:
		return wallClockSlot(dayStart, t, false) == slot
	}
}

func compareLocalDate(left, right time.Time) int {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()

	if leftYear != rightYear {
		if leftYear < rightYear {
			return -1
		}
		return 1
	}
	if leftMonth != rightMonth {
		if leftMonth < rightMonth {
			return -1
		}
		return 1
	}
	if leftDay != rightDay {
		if leftDay < rightDay {
			return -1
		}
		return 1
	}
	return 0
}

func nextQuarterHour(now time.Time) time.Time {
	base := now
	if base.IsZero() {
		base = time.Now()
	}

	step := time.Duration(dialogMinuteStep) * time.Minute
	rounded := base.Truncate(step)
	if !rounded.Equal(base) {
		rounded = rounded.Add(step)
	}
	return rounded
}

func maxTime(left, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func minTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func friendlyError(err error) string {
	switch {
	case errors.Is(err, ical.ErrAccessDenied):
		return "Calendar access was denied.\nEnable your terminal in System Settings > Privacy & Security > Calendars, then run the app again."
	case errors.Is(err, ical.ErrUnsupported):
		return "This app only works on macOS because it uses EventKit."
	default:
		return err.Error()
	}
}
