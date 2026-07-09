package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "calendar viewer failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	_, err := newProgram().Run()
	return err
}

func newProgram() *tea.Program {
	return tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
}
