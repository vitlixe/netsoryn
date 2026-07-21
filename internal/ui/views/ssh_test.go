package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vitlixe/netsoryn/internal/config"
)

func TestSSHView_SortsHosts(t *testing.T) {
	view := NewSSH(&config.Config{SSHHosts: []config.SSHHost{
		{Name: "zeta", Host: "zeta.example.com"},
		{Name: "alpha", Host: "alpha.example.com"},
	}})

	if got := view.hosts[0].Name; got != "alpha" {
		t.Fatalf("first host = %q, want alpha", got)
	}
}

func TestSSHView_Navigation(t *testing.T) {
	view := NewSSH(&config.Config{SSHHosts: []config.SSHHost{
		{Name: "alpha", Host: "alpha.example.com"},
		{Name: "zeta", Host: "zeta.example.com"},
	}})

	updated, _ := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	view = updated.(*SSH)
	if view.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", view.cursor)
	}

	updated, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	view = updated.(*SSH)
	if view.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", view.cursor)
	}
}

func TestSSHView_EnterAndDumpReturnCommands(t *testing.T) {
	view := NewSSH(&config.Config{SSHHosts: []config.SSHHost{{Name: "local", Host: "localhost"}}})

	if _, cmd := view.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
		t.Fatal("enter should return an ssh exec command")
	}
	if _, cmd := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}); cmd == nil {
		t.Fatal("d should return an ssh dump exec command")
	}
}

func TestSSHView_EmptyConfig(t *testing.T) {
	view := NewSSH(&config.Config{})
	out := view.View()
	if !strings.Contains(out, "No SSH hosts configured") {
		t.Fatalf("empty SSH view missing help text:\n%s", out)
	}
}

func TestSSHView_AddFormCapturesInput(t *testing.T) {
	view := NewSSH(&config.Config{})
	updated, _ := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	view = updated.(*SSH)

	if !view.CapturingInput() {
		t.Fatal("SSH view should capture input after pressing a")
	}
	if out := view.View(); !strings.Contains(out, "Add SSH Host") {
		t.Fatalf("add form not rendered:\n%s", out)
	}
}

func TestSSHView_AddHostSavesToConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("refresh_interval: 2\n"), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	view := NewSSH(cfg)
	view.startAdd()
	view.inputs[0].SetValue("prod")
	view.inputs[1].SetValue("prod.example.com")
	view.inputs[2].SetValue("deploy")
	view.inputs[3].SetValue("2222")
	view.inputs[4].SetValue("~/.ssh/prod")
	view.inputs[5].SetValue("-o BatchMode=yes")

	view.saveAddForm()
	if view.adding {
		t.Fatal("form should close after saving")
	}
	if len(view.hosts) != 1 {
		t.Fatalf("hosts len = %d, want 1", len(view.hosts))
	}
	if view.hosts[0].Name != "prod" || view.hosts[0].Port != 2222 {
		t.Fatalf("saved host = %+v, want prod:2222", view.hosts[0])
	}

	reloaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}
	if len(reloaded.SSHHosts) != 1 || reloaded.SSHHosts[0].Name != "prod" {
		t.Fatalf("reloaded SSH hosts = %+v, want prod", reloaded.SSHHosts)
	}
}

func TestSSHView_AddHostShowsDuplicateError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
refresh_interval: 2
ssh_hosts:
  - name: "prod"
    host: "prod.example.com"
`), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	view := NewSSH(cfg)
	view.startAdd()
	view.inputs[0].SetValue("prod")
	view.inputs[1].SetValue("other.example.com")

	view.saveAddForm()
	if !view.adding {
		t.Fatal("form should stay open after duplicate error")
	}
	if !strings.Contains(view.formErr, "duplicates") {
		t.Fatalf("formErr = %q, want duplicate error", view.formErr)
	}
}
