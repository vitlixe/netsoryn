package ui

import (
	"strings"
	"testing"

	"github.com/vitlixe/netsoryn/internal/config"
)

func newTestModel(active ViewID) RootModel {
	cfg := &config.Config{}
	m := New(cfg, "test")
	m.active = active
	m.width = 120
	m.height = 30
	return m
}

func TestFooterDashboard_NoNavFilterSortRefresh(t *testing.T) {
	footer := newTestModel(ViewDashboard).renderFooter()
	for _, unwanted := range []string{"navigate", "filter", "sort", "refresh"} {
		if strings.Contains(footer, unwanted) {
			t.Errorf("Dashboard footer should not contain %q, got: %s", unwanted, footer)
		}
	}
}

func TestFooterProcesses_HasNavFilterSort(t *testing.T) {
	footer := newTestModel(ViewProcesses).renderFooter()
	for _, want := range []string{"navigate", "filter", "sort"} {
		if !strings.Contains(footer, want) {
			t.Errorf("Processes footer should contain %q, got: %s", want, footer)
		}
	}
}

func TestFooterTableViews_HasNavFilterNoSort(t *testing.T) {
	tableViews := []ViewID{ViewNetwork, ViewPorts, ViewServices, ViewDocker}
	for _, id := range tableViews {
		footer := newTestModel(id).renderFooter()
		if !strings.Contains(footer, "navigate") {
			t.Errorf("view %d footer should contain navigate", id)
		}
		if !strings.Contains(footer, "filter") {
			t.Errorf("view %d footer should contain filter", id)
		}
		if strings.Contains(footer, "sort") {
			t.Errorf("view %d footer should not contain sort", id)
		}
	}
}

func TestFooterNetwork_HasTabsHint(t *testing.T) {
	footer := newTestModel(ViewNetwork).renderFooter()
	if !strings.Contains(footer, "h/l") {
		t.Errorf("Network footer should contain h/l tabs hint, got: %s", footer)
	}
	if !strings.Contains(footer, "tabs") {
		t.Errorf("Network footer should contain tabs hint, got: %s", footer)
	}
}

func TestFooterDNS_HasNewQueryHint(t *testing.T) {
	footer := newTestModel(ViewDNS).renderFooter()
	if !strings.Contains(footer, "n") {
		t.Errorf("DNS footer should contain n hint, got: %s", footer)
	}
	if !strings.Contains(footer, "query") {
		t.Errorf("DNS footer should contain query hint, got: %s", footer)
	}
}

func TestFooterHTTP_HasNewCheckHint(t *testing.T) {
	footer := newTestModel(ViewHTTP).renderFooter()
	if !strings.Contains(footer, "n") {
		t.Errorf("HTTP footer should contain n hint, got: %s", footer)
	}
	if !strings.Contains(footer, "check") {
		t.Errorf("HTTP footer should contain check hint, got: %s", footer)
	}
}

func TestFooterDNSHTTP_NoNavFilterSort(t *testing.T) {
	for _, id := range []ViewID{ViewDNS, ViewHTTP} {
		footer := newTestModel(id).renderFooter()
		for _, unwanted := range []string{"navigate", "filter", "sort", "refresh"} {
			if strings.Contains(footer, unwanted) {
				t.Errorf("view %d footer should not contain %q", id, unwanted)
			}
		}
	}
}

func TestFooterNoRefreshAnywhere(t *testing.T) {
	for id := ViewID(0); id < numViews; id++ {
		footer := newTestModel(id).renderFooter()
		if strings.Contains(footer, "refresh") {
			t.Errorf("view %d footer should not contain refresh (not implemented)", id)
		}
	}
}

func TestFooterSortHint_OnlyInProcesses(t *testing.T) {
	views := []struct {
		id       ViewID
		wantSort bool
	}{
		{ViewDashboard, false},
		{ViewProcesses, true},
		{ViewNetwork, false},
		{ViewPorts, false},
		{ViewServices, false},
		{ViewDocker, false},
		{ViewDNS, false},
		{ViewHTTP, false},
	}

	for _, tc := range views {
		footer := newTestModel(tc.id).renderFooter()
		hasSortHint := strings.Contains(footer, "sort")
		if hasSortHint != tc.wantSort {
			t.Errorf("view %d: footer sort hint present=%v, want=%v", tc.id, hasSortHint, tc.wantSort)
		}
	}
}
