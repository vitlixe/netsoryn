package views

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vitlixe/netsoryn/internal/config"
)

type activatableView interface {
	SetActive(bool) tea.Cmd
}

// TestViewsImplementSetActive ensures the heavy, periodically-collecting views
// can be paused by the root model when they are not in focus.
func TestViewsImplementSetActive(t *testing.T) {
	cfg := &config.Config{}
	ctx := context.Background()
	views := map[string]tea.Model{
		"dashboard": NewDashboard(cfg, ctx),
		"processes": NewProcesses(cfg, ctx),
		"network":   NewNetwork(cfg, ctx),
		"ports":     NewPorts(cfg, ctx),
		"services":  NewServices(cfg, ctx),
		"docker":    NewDocker(cfg, ctx),
	}
	for name, v := range views {
		if _, ok := v.(activatableView); !ok {
			t.Errorf("%s view (%T) does not implement SetActive", name, v)
		}
	}
}

func TestProcesses_SetActiveTogglesAndRefreshes(t *testing.T) {
	p := NewProcesses(&config.Config{}, context.Background())
	if p.active {
		t.Fatal("a new view should start inactive (no background collection)")
	}
	if cmd := p.SetActive(true); cmd == nil {
		t.Error("SetActive(true) should return a refresh command")
	}
	if !p.active {
		t.Error("SetActive(true) should set active=true")
	}
	if cmd := p.SetActive(false); cmd != nil {
		t.Error("SetActive(false) should not return a refresh command")
	}
	if p.active {
		t.Error("SetActive(false) should set active=false")
	}
}
