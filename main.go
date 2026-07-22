package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/maaslalani/cal/internal/calendar"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version string

func main() {
	root := &cobra.Command{
		Use:   "cal",
		Short: "Calendar in the terminal",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return calendar.Run()
		},
	}
	root.AddCommand(calendar.AddCommand(), calendar.ViewCommand())

	var opts []fang.Option
	if version != "" {
		opts = append(opts, fang.WithVersion(version))
	}

	if err := fang.Execute(context.Background(), root, opts...); err != nil {
		os.Exit(1)
	}
}
