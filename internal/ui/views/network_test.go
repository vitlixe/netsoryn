package views

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vitlixe/netsoryn/internal/config"
)

func newTestNetwork() *Network {
	return NewNetwork(&config.Config{}, context.Background())
}

// bubbles table.SetHeight(h) stores h-headersHeight in the viewport, so
// Height() returns h-headersHeight (2 for a table with header+separator).
// With ContentSizeMsg.Height=20: tableH=17, viewport=15.
func TestNetworkTableHeight_Normal(t *testing.T) {
	n := newTestNetwork()
	_, _ = n.Update(ContentSizeMsg{Width: 120, Height: 20})

	// tableH = 20-3 = 17; viewport = 17-2 = 15
	want := 15
	if got := n.ifaceTable.Height(); got != want {
		t.Errorf("ifaceTable.Height() = %d, want %d", got, want)
	}
	if got := n.connTable.Height(); got != want {
		t.Errorf("connTable.Height() = %d, want %d", got, want)
	}
}

// Very small heights: viewport must keep at least one data row (clamped tableH >= 3).
func TestNetworkTableHeight_MinimumNotNegative(t *testing.T) {
	n := newTestNetwork()
	for _, h := range []int{0, 1, 2, 3} {
		_, _ = n.Update(ContentSizeMsg{Width: 120, Height: h})
		if got := n.ifaceTable.Height(); got < 1 {
			t.Errorf("Height=%d: ifaceTable.Height() = %d, want >= 1", h, got)
		}
		if got := n.connTable.Height(); got < 1 {
			t.Errorf("Height=%d: connTable.Height() = %d, want >= 1", h, got)
		}
	}
}

// ── focus sync on tab switch ──────────────────────────────────────────────────

func TestNetworkFocus_InitialState(t *testing.T) {
	n := newTestNetwork()
	if !n.ifaceTable.Focused() {
		t.Error("ifaceTable should be focused initially")
	}
	if n.connTable.Focused() {
		t.Error("connTable should not be focused initially")
	}
}

func TestNetworkFocus_SwitchToConnections(t *testing.T) {
	n := newTestNetwork()
	_, _ = n.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})

	if n.activeTab != netTabConns {
		t.Errorf("activeTab = %d, want netTabConns (%d)", n.activeTab, netTabConns)
	}
	if !n.connTable.Focused() {
		t.Error("connTable should be focused after switching to Connections")
	}
	if n.ifaceTable.Focused() {
		t.Error("ifaceTable should not be focused after switching to Connections")
	}
}

func TestNetworkFocus_SwitchBackToInterfaces(t *testing.T) {
	n := newTestNetwork()
	// Go to Connections first.
	_, _ = n.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	// Then back to Interfaces.
	_, _ = n.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})

	if n.activeTab != netTabIfaces {
		t.Errorf("activeTab = %d, want netTabIfaces (%d)", n.activeTab, netTabIfaces)
	}
	if !n.ifaceTable.Focused() {
		t.Error("ifaceTable should be focused after switching back to Interfaces")
	}
	if n.connTable.Focused() {
		t.Error("connTable should not be focused after switching back to Interfaces")
	}
}

func TestNetworkFocus_RightArrowSwitchesToConnections(t *testing.T) {
	n := newTestNetwork()
	_, _ = n.Update(tea.KeyMsg{Type: tea.KeyRight})

	if n.activeTab != netTabConns {
		t.Errorf("activeTab = %d, want netTabConns (%d)", n.activeTab, netTabConns)
	}
	if !n.connTable.Focused() {
		t.Error("connTable should be focused after right arrow")
	}
	if n.ifaceTable.Focused() {
		t.Error("ifaceTable should not be focused after right arrow")
	}
}

func TestNetworkFocus_LeftArrowSwitchesBackToInterfaces(t *testing.T) {
	n := newTestNetwork()
	// Move to Connections first.
	_, _ = n.Update(tea.KeyMsg{Type: tea.KeyRight})
	// Move back with left arrow.
	_, _ = n.Update(tea.KeyMsg{Type: tea.KeyLeft})

	if n.activeTab != netTabIfaces {
		t.Errorf("activeTab = %d, want netTabIfaces (%d)", n.activeTab, netTabIfaces)
	}
	if !n.ifaceTable.Focused() {
		t.Error("ifaceTable should be focused after left arrow")
	}
	if n.connTable.Focused() {
		t.Error("connTable should not be focused after left arrow")
	}
}
