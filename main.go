package main

import (
	"fmt"
	"os"

	"github.com/maaslalani/cal/internal/calendar"
)

func main() {
	if err := calendar.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "calendar viewer failed: %v\n", err)
		os.Exit(1)
	}
}
