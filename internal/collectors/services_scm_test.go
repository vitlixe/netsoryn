package collectors

import (
	"testing"
)

func TestParseSCMJSON_Array(t *testing.T) {
	input := `[{"Name":"wuauserv","DisplayName":"Windows Update","Status":"Running"},{"Name":"spooler","DisplayName":"Print Spooler","Status":"Stopped"}]`
	data, err := parseSCMJSON([]byte(input), "windows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(data.Services))
	}

	svc := data.Services[0]
	if svc.Name != "wuauserv" {
		t.Errorf("Name = %q, want %q", svc.Name, "wuauserv")
	}
	if svc.Description != "Windows Update" {
		t.Errorf("Description = %q, want %q", svc.Description, "Windows Update")
	}
	if svc.SubState != "running" {
		t.Errorf("SubState = %q, want %q", svc.SubState, "running")
	}
	if svc.ActiveState != "active" {
		t.Errorf("ActiveState = %q, want %q", svc.ActiveState, "active")
	}
	if svc.LoadState != "loaded" {
		t.Errorf("LoadState = %q, want %q", svc.LoadState, "loaded")
	}

	svc2 := data.Services[1]
	if svc2.SubState != "stopped" {
		t.Errorf("SubState = %q, want %q", svc2.SubState, "stopped")
	}
	if svc2.ActiveState != "inactive" {
		t.Errorf("ActiveState = %q, want %q", svc2.ActiveState, "inactive")
	}
}

func TestParseSCMJSON_SingleObject(t *testing.T) {
	input := `{"Name":"wsearch","DisplayName":"Windows Search","Status":"Running"}`
	data, err := parseSCMJSON([]byte(input), "windows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(data.Services))
	}
	if data.Services[0].Name != "wsearch" {
		t.Errorf("Name = %q, want %q", data.Services[0].Name, "wsearch")
	}
}

func TestParseSCMJSON_EmptyArray(t *testing.T) {
	input := `[]`
	data, err := parseSCMJSON([]byte(input), "windows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(data.Services))
	}
	if data.Platform != "windows" {
		t.Errorf("Platform = %q, want %q", data.Platform, "windows")
	}
}

func TestParseSCMJSON_EmptyInput(t *testing.T) {
	data, err := parseSCMJSON([]byte(""), "windows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(data.Services))
	}
}

func TestParseSCMJSON_Paused(t *testing.T) {
	input := `[{"Name":"foo","DisplayName":"Foo Svc","Status":"Paused"}]`
	data, err := parseSCMJSON([]byte(input), "windows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := data.Services[0]
	if svc.SubState != "paused" {
		t.Errorf("SubState = %q, want %q", svc.SubState, "paused")
	}
	if svc.ActiveState != "active" {
		t.Errorf("ActiveState = %q, want %q", svc.ActiveState, "active")
	}
}

func TestParseSCMJSON_Pending(t *testing.T) {
	cases := []string{"StartPending", "StopPending", "ContinuePending", "PausePending"}
	for _, status := range cases {
		input := `[{"Name":"bar","DisplayName":"Bar","Status":"` + status + `"}]`
		data, err := parseSCMJSON([]byte(input), "windows")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", status, err)
		}
		svc := data.Services[0]
		if svc.SubState != "pending" {
			t.Errorf("%s: SubState = %q, want %q", status, svc.SubState, "pending")
		}
		if svc.ActiveState != "active" {
			t.Errorf("%s: ActiveState = %q, want %q", status, svc.ActiveState, "active")
		}
	}
}

func TestParseSCMJSON_UnknownStatus(t *testing.T) {
	input := `[{"Name":"baz","DisplayName":"Baz","Status":"SomeWeirdState"}]`
	data, err := parseSCMJSON([]byte(input), "windows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := data.Services[0]
	if svc.SubState != "someweirdstate" {
		t.Errorf("SubState = %q, want %q", svc.SubState, "someweirdstate")
	}
	if svc.ActiveState != "unknown" {
		t.Errorf("ActiveState = %q, want %q", svc.ActiveState, "unknown")
	}
}

func TestScmStatusToStates(t *testing.T) {
	cases := []struct {
		status     string
		wantSub    string
		wantActive string
	}{
		{"Running", "running", "active"},
		{"4", "running", "active"},
		{"Stopped", "stopped", "inactive"},
		{"1", "stopped", "inactive"},
		{"Paused", "paused", "active"},
		{"7", "paused", "active"},
		{"StartPending", "pending", "active"},
		{"2", "pending", "active"},
		{"StopPending", "pending", "active"},
		{"3", "pending", "active"},
		{"ContinuePending", "pending", "active"},
		{"5", "pending", "active"},
		{"PausePending", "pending", "active"},
		{"6", "pending", "active"},
		{"Unknown", "unknown", "unknown"},
		{"", "", "unknown"},
	}
	for _, c := range cases {
		sub, active := scmStatusToStates(c.status)
		if sub != c.wantSub || active != c.wantActive {
			t.Errorf("scmStatusToStates(%q) = (%q, %q), want (%q, %q)",
				c.status, sub, active, c.wantSub, c.wantActive)
		}
	}
}
