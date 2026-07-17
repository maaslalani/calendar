package calendar

import tea "github.com/charmbracelet/bubbletea"

func Run() error {
	_, err := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}
