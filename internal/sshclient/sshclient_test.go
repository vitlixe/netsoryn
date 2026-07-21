package sshclient

import (
	"reflect"
	"testing"

	"github.com/vitlixe/netsoryn/internal/config"
)

func TestFindHost(t *testing.T) {
	cfg := &config.Config{
		SSHHosts: []config.SSHHost{
			{Name: "prod", Host: "prod.example.com"},
			{Name: "staging", Host: "staging.example.com"},
		},
	}

	tests := []struct {
		name     string
		lookup   string
		wantOK   bool
		wantHost string
	}{
		{"found first", "prod", true, "prod.example.com"},
		{"found second", "staging", true, "staging.example.com"},
		{"missing", "dev", false, ""},
		{"empty name", "", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, ok := FindHost(cfg, tt.lookup)
			if ok != tt.wantOK {
				t.Fatalf("FindHost(%q) ok = %v, want %v", tt.lookup, ok, tt.wantOK)
			}
			if host.Host != tt.wantHost {
				t.Fatalf("FindHost(%q) host = %q, want %q", tt.lookup, host.Host, tt.wantHost)
			}
		})
	}
}

func TestFindHost_NoHosts(t *testing.T) {
	if _, ok := FindHost(&config.Config{}, "prod"); ok {
		t.Fatalf("FindHost on empty config = true, want false")
	}
}

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name   string
		host   config.SSHHost
		remote []string
		want   []string
	}{
		{
			name: "all fields with remote command",
			host: config.SSHHost{
				Host:    "example.com",
				User:    "deploy",
				Port:    2222,
				Key:     "~/.ssh/prod_ed25519",
				Options: []string{"-o", "StrictHostKeyChecking=accept-new"},
			},
			remote: []string{"uname", "-a"},
			want: []string{
				"-o", "StrictHostKeyChecking=accept-new",
				"-i", "~/.ssh/prod_ed25519",
				"-p", "2222",
				"deploy@example.com",
				"uname", "-a",
			},
		},
		{
			name: "default port 22 is omitted",
			host: config.SSHHost{Host: "localhost", Port: 22},
			want: []string{"localhost"},
		},
		{
			name: "zero port is omitted",
			host: config.SSHHost{Host: "localhost"},
			want: []string{"localhost"},
		},
		{
			name: "no user drops the user prefix",
			host: config.SSHHost{Host: "example.com", Key: "~/.ssh/id_ed25519"},
			want: []string{"-i", "~/.ssh/id_ed25519", "example.com"},
		},
		{
			name:   "no remote command",
			host:   config.SSHHost{Host: "example.com", User: "root", Port: 22},
			remote: nil,
			want:   []string{"root@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildArgs(tt.host, tt.remote)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("BuildArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestTarget(t *testing.T) {
	tests := []struct {
		name string
		host config.SSHHost
		want string
	}{
		{"with user", config.SSHHost{Host: "example.com", User: "deploy"}, "deploy@example.com"},
		{"without user", config.SSHHost{Host: "example.com"}, "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Target(tt.host); got != tt.want {
				t.Fatalf("Target() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRemoteDumpCommand(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		sections string
		pretty   bool
		want     []string
	}{
		{
			name:   "format only",
			format: "json",
			want:   []string{"netsoryn", "dump", "--format", "json"},
		},
		{
			name:     "with sections",
			format:   "text",
			sections: "system,ports",
			want:     []string{"netsoryn", "dump", "--format", "text", "--sections", "system,ports"},
		},
		{
			name:   "with pretty",
			format: "json",
			pretty: true,
			want:   []string{"netsoryn", "dump", "--format", "json", "--pretty"},
		},
		{
			name:     "sections and pretty",
			format:   "json",
			sections: "all",
			pretty:   true,
			want:     []string{"netsoryn", "dump", "--format", "json", "--sections", "all", "--pretty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoteDumpCommand(tt.format, tt.sections, tt.pretty)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("RemoteDumpCommand() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
