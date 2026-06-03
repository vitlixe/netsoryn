package views

import "testing"

func TestCPUPercent(t *testing.T) {
	cases := []struct {
		name             string
		deltaCPU, deltaW float64
		numCPU           int
		want             float64
	}{
		{"one core fully busy on 4 cores", 1.0, 1.0, 4, 25},
		{"half a core on 2 cores", 0.5, 1.0, 2, 25},
		{"idle", 0, 1.0, 8, 0},
		{"zero wall-clock delta", 1.0, 0, 4, 0},
		{"zero cpus", 1.0, 1.0, 0, 0},
		{"negative delta (pid reused)", -2.0, 1.0, 4, 0},
		{"clamped over 100", 100, 1.0, 1, 100},
	}
	for _, c := range cases {
		if got := cpuPercent(c.deltaCPU, c.deltaW, c.numCPU); got != c.want {
			t.Errorf("%s: cpuPercent(%v, %v, %d) = %v, want %v",
				c.name, c.deltaCPU, c.deltaW, c.numCPU, got, c.want)
		}
	}
}

func TestPerSecond(t *testing.T) {
	if got := perSecond(1000, 2); got != 500 {
		t.Errorf("perSecond(1000, 2) = %v, want 500", got)
	}
	if got := perSecond(1000, 0); got != 0 {
		t.Errorf("perSecond(1000, 0) = %v, want 0 (guard against div by zero)", got)
	}
}

func TestCounterDelta(t *testing.T) {
	if got := counterDelta(500, 100); got != 400 {
		t.Errorf("counterDelta(500, 100) = %d, want 400", got)
	}
	if got := counterDelta(100, 500); got != 0 {
		t.Errorf("counterDelta(100, 500) = %d, want 0 (counter reset)", got)
	}
}
