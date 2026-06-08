package main

import (
	"strings"
	"testing"

	"github.com/vitlixe/netsoryn/internal/collectors"
)

func sampleSystem() collectors.SystemData {
	return collectors.SystemData{
		Hostname:         "testhost",
		CPUTotal:         25,
		CPUPercents:      []float64{40, 10},
		MemUsed:          1 << 30,
		MemTotal:         2 << 30,
		MemPercent:       50,
		LoadAvg1:         1.5,
		LoadAvgSupported: true,
		UptimeSeconds:    90000, // 1d 1h 0m
		Disks:            []collectors.DiskStat{{Mountpoint: "/", Used: 1 << 30, Total: 2 << 30, UsedPercent: 50, Fstype: "apfs"}},
		DiskIO:           []collectors.DiskIOStat{{Name: "disk0", ReadBytes: 1 << 30, WriteBytes: 0}},
	}
}

func TestRenderMarkdown(t *testing.T) {
	out := renderMarkdown(sampleSystem())
	for _, want := range []string{
		"# 🖥 testhost", "## System", "| CPU | 25.0% |",
		"## CPU cores", "## Disks", "| / |", "## Disk I/O", "disk0",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderText(t *testing.T) {
	out := renderText(sampleSystem())
	for _, want := range []string{"testhost", "System", "CPU", "Disks", "disk0"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q:\n%s", want, out)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1 << 20, "1.0 MB"},
		{1 << 30, "1.0 GB"},
	}
	for _, c := range cases {
		if got := humanBytes(c.in); got != c.want {
			t.Errorf("humanBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHumanUptime(t *testing.T) {
	if got := humanUptime(90000); got != "1d 1h 0m" {
		t.Errorf("humanUptime(90000) = %q, want %q", got, "1d 1h 0m")
	}
	if got := humanUptime(3700); got != "1h 1m" {
		t.Errorf("humanUptime(3700) = %q, want %q", got, "1h 1m")
	}
}
