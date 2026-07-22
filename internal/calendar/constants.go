package calendar

import "time"

const (
	timeColumnWidth      = 8
	dayColumnWidth       = 30
	screenPaddingX       = 2
	screenPaddingY       = 1
	dialogCols           = 60
	dialogRows           = 15
	dialogPaddingX       = 5
	dialogPaddingY       = 1
	dialogMinuteStep     = 15
	minAllDayBufferRows  = 2
	maxAllDayEvents      = 3
	slotDuration         = 30 * time.Minute
	slotsPerDay          = int((24 * time.Hour) / slotDuration)
	defaultWindowFrom    = 16
	currentTimeTickEvery = time.Minute
	calendarRefreshEvery = 5 * time.Second
	recurringMarker      = "*"
	legendSwatch         = "●"
	recurringMarkerPad   = 1
	recurringMarkerScale = 75
	eventTitleRightPad   = 1
	eventAccent          = "┃ "
	shortEventAccent     = "│ "
	watchStartRetries    = 6
	watchRetryDelay      = 10 * time.Millisecond
)

const (
	scrollGestureGap          = 100 * time.Millisecond
	scrollAxisThreshold       = 2
	scrollAxisRatio           = 2
	horizontalScrollThreshold = 4
)
