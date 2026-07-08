package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

var (
	mainKeys = []key.Binding{
		key.NewBinding(key.WithKeys("h", "l"), key.WithHelp("h/l", "days")),
		key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "scroll")),
		key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "today")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}

	dialogKeys = []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "fields")),
		key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "adjust")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
)

func footerView(showDialog bool, width int) string {
	if width <= 0 {
		return ""
	}

	h := help.New()
	h.Width = width

	bindings := mainKeys
	if showDialog {
		bindings = dialogKeys
	}
	return h.ShortHelpView(bindings)
}
