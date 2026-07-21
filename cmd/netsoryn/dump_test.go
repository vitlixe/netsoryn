package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vitlixe/netsoryn/internal/collectors"
	"github.com/vitlixe/netsoryn/internal/config"
)

func TestParseDumpSections_Default(t *testing.T) {
	sections, err := parseDumpSections("")
	if err != nil {
		t.Fatalf("parseDumpSections returned error: %v", err)
	}
	if len(sections) != 1 || !sections["system"] {
		t.Fatalf("sections = %#v, want only system", sections)
	}
}

func TestParseDumpSections_AllAndAliases(t *testing.T) {
	sections, err := parseDumpSections("sys,proc,net,svc,docker,ports,all")
	if err != nil {
		t.Fatalf("parseDumpSections returned error: %v", err)
	}
	for _, name := range allDumpSections {
		if !sections[name] {
			t.Fatalf("sections = %#v, missing %q", sections, name)
		}
	}
}

func TestParseDumpSections_RejectsUnknown(t *testing.T) {
	if _, err := parseDumpSections("system,secrets"); err == nil {
		t.Fatal("parseDumpSections returned nil error, want unknown section error")
	}
}

func TestDumpSnapshot_SystemOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := dumpSnapshot(ctx, &config.Config{ProcessLimit: 50, PortsListenOnly: true}, dumpSections{"system": true})
	if err != nil {
		t.Fatalf("dumpSnapshot: %v", err)
	}
	if out.System == nil {
		t.Fatal("System section is nil")
	}
	if out.Processes != nil || out.Network != nil || out.Ports != nil || out.Services != nil || out.Docker != nil {
		t.Fatalf("dumpSnapshot(system) returned extra sections: %+v", out)
	}
}

func TestDumpOutputJSONShape(t *testing.T) {
	out := dumpOutput{
		System:    &collectors.SystemData{Hostname: "host"},
		Processes: &collectors.ProcessData{Processes: []collectors.ProcessStat{{PID: 123, Name: "netsoryn"}}},
		Network:   &collectors.NetworkData{Interfaces: []collectors.NetInterface{{Name: "lo0"}}},
		Ports:     &collectors.PortData{Ports: []collectors.PortStat{{Port: 22, Protocol: "TCP"}}},
		Services:  &collectors.ServiceData{Platform: "linux"},
		Docker:    &collectors.DockerData{Available: true},
	}

	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	for _, key := range []string{`"system"`, `"processes"`, `"network"`, `"ports"`, `"services"`, `"docker"`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("dump JSON %s missing key %s", raw, key)
		}
	}
}

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
