package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestMean(t *testing.T) {
	cases := []struct {
		in   []float64
		want float64
	}{
		{nil, 0},
		{[]float64{10}, 10},
		{[]float64{0, 100}, 50},
		{[]float64{25, 50, 75}, 50},
	}
	for _, c := range cases {
		if got := mean(c.in); got != c.want {
			t.Errorf("mean(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSystemSnapshot(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := systemSnapshot(ctx)
	if err != nil {
		t.Fatalf("systemSnapshot: %v", err)
	}
	if sys.MemTotal == 0 {
		t.Error("MemTotal should be non-zero")
	}
	if sys.CPUTotal < 0 || sys.CPUTotal > 100 {
		t.Errorf("CPUTotal out of range: %v", sys.CPUTotal)
	}

	// The snapshot must round-trip through JSON for the dump command.
	if _, err := json.Marshal(dumpOutput{System: &sys}); err != nil {
		t.Errorf("snapshot not JSON-serializable: %v", err)
	}
}
