package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vitlixe/netsoryn/internal/config"
	"github.com/vitlixe/netsoryn/internal/sshclient"
)

func TestBuildSSHArgs(t *testing.T) {
	host := config.SSHHost{
		Name:    "prod",
		Host:    "example.com",
		User:    "deploy",
		Port:    2222,
		Key:     "~/.ssh/prod_ed25519",
		Options: []string{"-o", "StrictHostKeyChecking=accept-new"},
	}

	got := sshclient.BuildArgs(host, []string{"uname", "-a"})
	want := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-i", "~/.ssh/prod_ed25519",
		"-p", "2222",
		"deploy@example.com",
		"uname", "-a",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSSHArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildSSHArgs_DefaultPortIsOmitted(t *testing.T) {
	host := config.SSHHost{Name: "local", Host: "localhost", Port: 22}

	got := sshclient.BuildArgs(host, nil)
	want := []string{"localhost"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSSHArgs() = %#v, want %#v", got, want)
	}
}

func TestSSHListCommand(t *testing.T) {
	path := writeMainTestConfig(t, `
refresh_interval: 2
ssh_hosts:
  - name: "zeta"
    host: "zeta.example.com"
  - name: "alpha"
    host: "10.0.0.2"
    user: "deploy"
    port: 2222
    key: "~/.ssh/alpha"
`)

	out, err := executeRoot(t, "--config", path, "ssh", "list")
	if err != nil {
		t.Fatalf("ssh list returned error: %v", err)
	}
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "alpha") || !strings.Contains(out, "deploy@10.0.0.2:2222") {
		t.Fatalf("ssh list output missing expected host data:\n%s", out)
	}
	if strings.Index(out, "alpha") > strings.Index(out, "zeta") {
		t.Fatalf("ssh list output should be sorted by name:\n%s", out)
	}
}

func TestSSHCommandRunsConfiguredHost(t *testing.T) {
	path := writeMainTestConfig(t, `
refresh_interval: 2
ssh_hosts:
  - name: "prod"
    host: "example.com"
    user: "deploy"
    port: 2222
`)

	got := captureSSHRun(t, func() error {
		_, err := executeRoot(t, "--config", path, "ssh", "prod", "--", "uname", "-a")
		return err
	})
	want := []string{"-p", "2222", "deploy@example.com", "uname", "-a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ssh runner args = %#v, want %#v", got, want)
	}
}

func TestSSHDumpCommandRunsRemoteDump(t *testing.T) {
	path := writeMainTestConfig(t, `
refresh_interval: 2
ssh_hosts:
  - name: "prod"
    host: "example.com"
`)

	got := captureSSHRun(t, func() error {
		_, err := executeRoot(t, "--config", path, "ssh", "dump", "prod", "-f", "text", "--sections", "system,ports")
		return err
	})
	want := []string{"example.com", "netsoryn", "dump", "--format", "text", "--sections", "system,ports"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ssh dump runner args = %#v, want %#v", got, want)
	}
}

func TestSSHUnknownHost(t *testing.T) {
	path := writeMainTestConfig(t, `
refresh_interval: 2
ssh_hosts:
  - name: "prod"
    host: "example.com"
`)

	called := false
	old := runSSHCommand
	runSSHCommand = func(args []string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { runSSHCommand = old })

	_, err := executeRoot(t, "--config", path, "ssh", "missing")
	if err == nil {
		t.Fatal("ssh missing returned nil error, want unknown host error")
	}
	if called {
		t.Fatal("ssh runner was called for an unknown host")
	}
}

func captureSSHRun(t *testing.T, run func() error) []string {
	t.Helper()

	var got []string
	old := runSSHCommand
	runSSHCommand = func(args []string) error {
		got = append([]string(nil), args...)
		return nil
	}
	t.Cleanup(func() { runSSHCommand = old })

	if err := run(); err != nil {
		t.Fatalf("command returned error: %v", err)
	}
	return got
}

func executeRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := rootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func writeMainTestConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return path
}
