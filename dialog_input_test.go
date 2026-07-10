package main

import "testing"

func TestAdjacentDialogFocus(t *testing.T) {
	tests := []struct {
		name    string
		current int
		delta   int
		want    int
		wantOK  bool
	}{
		{name: "next field", current: dialogFocusMonth, delta: 1, want: dialogFocusDay, wantOK: true},
		{name: "previous field", current: dialogFocusDuration, delta: -1, want: dialogFocusMinute, wantOK: true},
		{name: "next action", current: dialogFocusSubmit, delta: 1, want: dialogFocusCancel, wantOK: true},
		{name: "previous action", current: dialogFocusCancel, delta: -1, want: dialogFocusSubmit, wantOK: true},
		{name: "before first field", current: dialogFocusMonth, delta: -1},
		{name: "after last field", current: dialogFocusDuration, delta: 1},
		{name: "title excluded", current: dialogFocusTitle, delta: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := adjacentDialogFocus(tt.current, tt.delta)
			if ok != tt.wantOK {
				t.Fatalf("adjacentDialogFocus(%d, %d) ok = %v, want %v", tt.current, tt.delta, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Fatalf("adjacentDialogFocus(%d, %d) = %d, want %d", tt.current, tt.delta, got, tt.want)
			}
		})
	}
}
