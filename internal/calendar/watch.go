package calendar

import (
	"context"
	"errors"
	"strings"
	"time"

	ical "github.com/BRO3886/go-eventkit/calendar"
	tea "github.com/charmbracelet/bubbletea"
)

type watchCalendarReadyMsg struct {
	changes <-chan struct{}
	cancel  context.CancelFunc
	err     error
}

type watchCalendarChangedMsg struct{}

type watchCalendarStoppedMsg struct{}

func startCalendarWatchCmd() tea.Cmd {
	return func() tea.Msg {
		client, err := ical.New()
		if err != nil {
			return watchCalendarReadyMsg{err: err}
		}

		changes, cancel, err := openCalendarWatch(client.WatchChanges, time.Sleep)
		if err != nil {
			return watchCalendarReadyMsg{err: err}
		}

		return watchCalendarReadyMsg{
			changes: changes,
			cancel:  cancel,
		}
	}
}

func openCalendarWatch(start func(context.Context) (<-chan struct{}, error), sleep func(time.Duration)) (<-chan struct{}, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())

	for attempt := range watchStartRetries {
		changes, err := start(ctx)
		if err == nil {
			return changes, cancel, nil
		}
		if !isWatchAlreadyActiveError(err) || attempt == watchStartRetries-1 {
			cancel()
			return nil, nil, err
		}

		sleep(watchRetryDelay * time.Duration(1<<attempt))
	}

	cancel()
	return nil, nil, errors.New("calendar: failed to start watcher")
}

func isWatchAlreadyActiveError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "watcher already active")
}

func waitForCalendarChangeCmd(changes <-chan struct{}) tea.Cmd {
	if changes == nil {
		return nil
	}

	return func() tea.Msg {
		if _, ok := <-changes; !ok {
			return watchCalendarStoppedMsg{}
		}
		return watchCalendarChangedMsg{}
	}
}

func (m *model) stopCalendarWatch() {
	if m.watchCancel != nil {
		m.watchCancel()
	}
	m.watchCancel = nil
	m.watchChanges = nil
}
