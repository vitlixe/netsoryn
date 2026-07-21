package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(\"\") returned nil config")
	}
	if cfg.RefreshInterval <= 0 {
		t.Errorf("RefreshInterval = %v, want > 0", cfg.RefreshInterval)
	}
	if cfg.ProcessLimit <= 0 {
		t.Errorf("ProcessLimit = %d, want > 0", cfg.ProcessLimit)
	}
}

func TestLoad_NormalizesSecondBasedDurations(t *testing.T) {
	cfg := loadConfigFromString(t, `
refresh_interval: 5
http_checks:
  - url: "https://example.com"
    timeout: 10
`)

	if cfg.RefreshInterval != 5*time.Second {
		t.Errorf("RefreshInterval = %v, want %v", cfg.RefreshInterval, 5*time.Second)
	}
	if got := cfg.HTTPChecks[0].Timeout; got != 10*time.Second {
		t.Errorf("HTTPChecks[0].Timeout = %v, want %v", got, 10*time.Second)
	}
}

func TestLoad_ParsesStringDurations(t *testing.T) {
	cfg := loadConfigFromString(t, `
refresh_interval: "500ms"
http_checks:
  - url: "https://example.com"
    timeout: "2s"
`)

	if cfg.RefreshInterval != 500*time.Millisecond {
		t.Errorf("RefreshInterval = %v, want %v", cfg.RefreshInterval, 500*time.Millisecond)
	}
	if got := cfg.HTTPChecks[0].Timeout; got != 2*time.Second {
		t.Errorf("HTTPChecks[0].Timeout = %v, want %v", got, 2*time.Second)
	}
}

func TestLoad_RejectsInvalidDuration(t *testing.T) {
	path := writeConfig(t, `refresh_interval: "soon"`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error, want invalid duration error")
	}
}

func TestLoad_RejectsZeroRefreshInterval(t *testing.T) {
	path := writeConfig(t, `refresh_interval: 0`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error, want error for zero refresh_interval")
	}
}

func TestLoad_RejectsNegativeRefreshInterval(t *testing.T) {
	cases := []string{
		`refresh_interval: -3`,
		`refresh_interval: "-3"`,
	}
	for _, yaml := range cases {
		path := writeConfig(t, yaml)
		if _, err := Load(path); err == nil {
			t.Errorf("Load returned nil error for %q, want error", yaml)
		}
	}
}

func TestLoad_RejectsPathologicalDurationValues(t *testing.T) {
	// These must all produce errors: Inf/NaN as YAML floats, scientific notation
	// strings, and quoted pseudo-values that time.ParseDuration cannot handle.
	cases := []string{
		"refresh_interval: .inf",
		"refresh_interval: -.inf",
		"refresh_interval: .nan",
		`refresh_interval: "2e3"`,
		`refresh_interval: ".nan"`,
		`refresh_interval: ".inf"`,
	}
	for _, yaml := range cases {
		path := writeConfig(t, yaml)
		if _, err := Load(path); err == nil {
			t.Errorf("Load returned nil error for %q, want error", yaml)
		}
	}
}

func TestLoad_RejectsNegativeHTTPTimeout(t *testing.T) {
	path := writeConfig(t, `
refresh_interval: 2
http_checks:
  - url: "https://example.com"
    timeout: -1
`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error, want error for negative http_checks timeout")
	}
}

func TestLoad_AllowsZeroHTTPTimeout(t *testing.T) {
	// Zero timeout is valid; the HTTP collector applies its own default.
	cfg := loadConfigFromString(t, `
refresh_interval: 2
http_checks:
  - url: "https://example.com"
    timeout: 0
`)
	if got := cfg.HTTPChecks[0].Timeout; got != 0 {
		t.Errorf("HTTPChecks[0].Timeout = %v, want 0", got)
	}
}

func TestLoad_HTTPTimeoutDefault(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") returned error: %v", err)
	}
	if cfg.HTTPTimeout != 10*time.Second {
		t.Errorf("HTTPTimeout = %v, want %v", cfg.HTTPTimeout, 10*time.Second)
	}
}

func TestLoad_ParsesHTTPTimeout(t *testing.T) {
	cfg := loadConfigFromString(t, `
refresh_interval: 2
http_timeout: "2500ms"
`)
	if cfg.HTTPTimeout != 2500*time.Millisecond {
		t.Errorf("HTTPTimeout = %v, want %v", cfg.HTTPTimeout, 2500*time.Millisecond)
	}
}

func TestLoad_RejectsNegativeHTTPTimeoutDefault(t *testing.T) {
	path := writeConfig(t, `
refresh_interval: 2
http_timeout: -1
`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error, want error for negative http_timeout")
	}
}

func TestLoad_ParsesTCPChecks(t *testing.T) {
	cfg := loadConfigFromString(t, `
refresh_interval: 2
tcp_checks:
  - target: "1.1.1.1:53"
  - target: "github.com:443"
    timeout: 3
`)
	if len(cfg.TCPChecks) != 2 {
		t.Fatalf("TCPChecks len = %d, want 2", len(cfg.TCPChecks))
	}
	if cfg.TCPChecks[0].Target != "1.1.1.1:53" {
		t.Errorf("TCPChecks[0].Target = %q, want %q", cfg.TCPChecks[0].Target, "1.1.1.1:53")
	}
	if cfg.TCPChecks[1].Timeout != 3*time.Second {
		t.Errorf("TCPChecks[1].Timeout = %v, want %v", cfg.TCPChecks[1].Timeout, 3*time.Second)
	}
}

func TestLoad_RejectsNegativeTCPTimeout(t *testing.T) {
	path := writeConfig(t, `
refresh_interval: 2
tcp_checks:
  - target: "host:1"
    timeout: -1
`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error, want error for negative tcp_checks timeout")
	}
}

func TestLoad_ParsesSSHHosts(t *testing.T) {
	cfg := loadConfigFromString(t, `
refresh_interval: 2
ssh_hosts:
  - name: "prod"
    host: "10.0.0.5"
    user: "deploy"
    port: 2222
    key: "~/.ssh/prod_ed25519"
    options:
      - "-o"
      - "StrictHostKeyChecking=accept-new"
  - name: "staging"
    host: "staging.example.com"
`)
	if len(cfg.SSHHosts) != 2 {
		t.Fatalf("SSHHosts len = %d, want 2", len(cfg.SSHHosts))
	}
	if got := cfg.SSHHosts[0]; got.Name != "prod" || got.Host != "10.0.0.5" || got.User != "deploy" || got.Port != 2222 || got.Key != "~/.ssh/prod_ed25519" {
		t.Errorf("SSHHosts[0] = %+v, want populated prod host", got)
	}
	if got := cfg.SSHHosts[0].Options; len(got) != 2 || got[0] != "-o" || got[1] != "StrictHostKeyChecking=accept-new" {
		t.Errorf("SSHHosts[0].Options = %#v, want ssh options", got)
	}
	if got := cfg.SSHHosts[1].Port; got != 22 {
		t.Errorf("SSHHosts[1].Port = %d, want default 22", got)
	}
}

func TestLoad_RecordsConfigFile(t *testing.T) {
	path := writeConfig(t, `refresh_interval: 2`)
	cfg := loadConfigFromPath(t, path)
	if cfg.ConfigFile != path {
		t.Fatalf("ConfigFile = %q, want %q", cfg.ConfigFile, path)
	}
}

func TestAddSSHHost_CreatesConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "netsoryn", "config.yaml")
	cfg := &Config{ConfigFile: path, RefreshInterval: 2 * time.Second, ProcessLimit: 50, PortsListenOnly: true}

	written, err := AddSSHHost(cfg, SSHHost{Name: "prod", Host: "prod.example.com", User: "deploy"})
	if err != nil {
		t.Fatalf("AddSSHHost returned error: %v", err)
	}
	if written != path {
		t.Fatalf("written path = %q, want %q", written, path)
	}

	reloaded := loadConfigFromPath(t, path)
	if len(reloaded.SSHHosts) != 1 {
		t.Fatalf("SSHHosts len = %d, want 1", len(reloaded.SSHHosts))
	}
	if got := reloaded.SSHHosts[0]; got.Name != "prod" || got.Host != "prod.example.com" || got.User != "deploy" || got.Port != 22 {
		t.Fatalf("reloaded SSH host = %+v, want saved prod host with default port", got)
	}
}

func TestAddSSHHost_AppendsAndPreservesExistingKeys(t *testing.T) {
	path := writeConfig(t, `
refresh_interval: 7
ssh_hosts:
  - name: "alpha"
    host: "alpha.example.com"
`)
	cfg := loadConfigFromPath(t, path)

	if _, err := AddSSHHost(cfg, SSHHost{Name: "beta", Host: "beta.example.com", Port: 2222, Options: []string{"-o", "BatchMode=yes"}}); err != nil {
		t.Fatalf("AddSSHHost returned error: %v", err)
	}

	reloaded := loadConfigFromPath(t, path)
	if reloaded.RefreshInterval != 7*time.Second {
		t.Fatalf("RefreshInterval = %v, want 7s", reloaded.RefreshInterval)
	}
	if len(reloaded.SSHHosts) != 2 {
		t.Fatalf("SSHHosts len = %d, want 2", len(reloaded.SSHHosts))
	}
	if got := reloaded.SSHHosts[1]; got.Name != "beta" || got.Port != 2222 || len(got.Options) != 2 {
		t.Fatalf("second SSH host = %+v, want saved beta host", got)
	}
}

func TestAddSSHHost_RejectsDuplicate(t *testing.T) {
	path := writeConfig(t, `
refresh_interval: 2
ssh_hosts:
  - name: "prod"
    host: "prod.example.com"
`)
	cfg := loadConfigFromPath(t, path)

	if _, err := AddSSHHost(cfg, SSHHost{Name: "prod", Host: "other.example.com"}); err == nil {
		t.Fatal("AddSSHHost returned nil error, want duplicate host error")
	}
}

func TestLoad_RejectsInvalidSSHHosts(t *testing.T) {
	cases := []string{
		`
refresh_interval: 2
ssh_hosts:
  - host: "example.com"
`,
		`
refresh_interval: 2
ssh_hosts:
  - name: "prod"
`,
		`
refresh_interval: 2
ssh_hosts:
  - name: "prod"
    host: "one.example.com"
  - name: "prod"
    host: "two.example.com"
`,
		`
refresh_interval: 2
ssh_hosts:
  - name: "prod"
    host: "example.com"
    port: -1
`,
		`
refresh_interval: 2
ssh_hosts:
  - name: "prod"
    host: "example.com"
    port: 70000
`,
	}
	for _, yaml := range cases {
		path := writeConfig(t, yaml)
		if _, err := Load(path); err == nil {
			t.Errorf("Load returned nil error for %q, want invalid ssh_hosts error", yaml)
		}
	}
}

func TestLoad_EnvVarDurationString(t *testing.T) {
	t.Setenv("NETSORYN_REFRESH_INTERVAL", "500ms")
	cfg := loadConfigFromString(t, ``)
	if cfg.RefreshInterval != 500*time.Millisecond {
		t.Errorf("RefreshInterval = %v, want 500ms", cfg.RefreshInterval)
	}
}

func loadConfigFromString(t *testing.T, content string) *Config {
	t.Helper()

	path := writeConfig(t, content)
	return loadConfigFromPath(t, path)
}

func loadConfigFromPath(t *testing.T, path string) *Config {
	t.Helper()

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) returned error: %v", path, err)
	}
	return cfg
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return path
}
