package ui

import (
	"fmt"
	"strings"
	"testing"
)

func TestTooSmall(t *testing.T) {
	tests := []struct {
		w, h int
		want bool
	}{
		{MinWidth, MinHeight, false},
		{MinWidth - 1, MinHeight, true},
		{MinWidth, MinHeight - 1, true},
		{MinWidth - 1, MinHeight - 1, true},
		{MinWidth + 10, MinHeight + 5, false},
		{0, 0, true},
	}
	for _, tt := range tests {
		got := tooSmall(tt.w, tt.h)
		if got != tt.want {
			t.Errorf("tooSmall(%d, %d) = %v, want %v", tt.w, tt.h, got, tt.want)
		}
	}
}

func TestRenderSizeWarning_ContainsKeyParts(t *testing.T) {
	out := renderSizeWarning(80, 20)
	for _, substr := range []string{"NETSORYN", "Terminal window too small", "Required:", "quit"} {
		if !strings.Contains(out, substr) {
			t.Errorf("renderSizeWarning output missing %q", substr)
		}
	}
}

func TestRenderSizeWarning_NoPanic(t *testing.T) {
	sizes := [][2]int{{0, 0}, {1, 1}, {50, 10}, {80, 24}, {MinWidth, MinHeight}}
	for _, s := range sizes {
		renderSizeWarning(s[0], s[1]) // must not panic
	}
}

func TestRenderSizeWarning_RequiredValues(t *testing.T) {
	out := renderSizeWarning(80, 20)
	want := fmt.Sprintf("%d x %d", MinWidth, MinHeight)
	if !strings.Contains(out, want) {
		t.Errorf("renderSizeWarning missing required size %q in output", want)
	}
}
