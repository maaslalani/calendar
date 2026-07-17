package main

import (
	"testing"
	"time"
)

func TestRelativeDayLabel(t *testing.T) {
	reference := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		days int
		want string
	}{
		{-730, "2 years ago"},
		{-365, "1 year ago"},
		{-364, "12 months ago"},
		{-45, "2 months ago"},
		{-30, "1 month ago"},
		{-29, "29 days ago"},
		{-14, "14 days ago"},
		{-7, "7 days ago"},
		{-6, "6 days ago"},
		{-2, "2 days ago"},
		{-1, "Yesterday"},
		{0, "Today"},
		{1, "Tomorrow"},
		{2, "In 2 days"},
		{6, "In 6 days"},
		{7, "In 7 days"},
		{14, "In 14 days"},
		{29, "In 29 days"},
		{30, "In 1 month"},
		{45, "In 2 months"},
		{90, "In 3 months"},
		{364, "In 12 months"},
		{365, "In 1 year"},
		{730, "In 2 years"},
	}

	for _, test := range tests {
		day := reference.AddDate(0, 0, test.days)
		if got := relativeDayLabel(day, reference); got != test.want {
			t.Errorf("relativeDayLabel(%d days) = %q, want %q", test.days, got, test.want)
		}
	}
}
