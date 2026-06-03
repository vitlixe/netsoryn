package collectors

import (
	"reflect"
	"strings"
	"testing"
)

func TestDockerArgs(t *testing.T) {
	base := []string{"ps", "--all", "--format", "json"}

	cases := []struct {
		socketPath string
		want       []string
	}{
		{
			socketPath: "",
			want:       []string{"ps", "--all", "--format", "json"},
		},
		{
			socketPath: "/var/run/docker.sock",
			want:       []string{"--host", "/var/run/docker.sock", "ps", "--all", "--format", "json"},
		},
		{
			socketPath: "unix:///var/run/docker.sock",
			want:       []string{"--host", "unix:///var/run/docker.sock", "ps", "--all", "--format", "json"},
		},
		{
			socketPath: "npipe:////./pipe/docker_engine",
			want:       []string{"--host", "npipe:////./pipe/docker_engine", "ps", "--all", "--format", "json"},
		},
		{
			socketPath: "tcp://localhost:2375",
			want:       []string{"--host", "tcp://localhost:2375", "ps", "--all", "--format", "json"},
		},
	}

	for _, c := range cases {
		got := dockerArgs(c.socketPath, base...)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("dockerArgs(%q) = %v, want %v", c.socketPath, got, c.want)
		}
	}
}

func TestDockerArgs_DoesNotMutateInput(t *testing.T) {
	base := []string{"ps", "--all"}
	original := []string{"ps", "--all"}

	dockerArgs("/var/run/docker.sock", base...)

	if !reflect.DeepEqual(base, original) {
		t.Errorf("dockerArgs mutated input slice: got %v, want %v", base, original)
	}
}

func TestParseDockerPS(t *testing.T) {
	out := []byte(`{"ID":"abc123","Names":"/web","Image":"nginx:latest","Status":"Up 2 hours","State":"running","Ports":"0.0.0.0:80->80/tcp","CreatedAt":"2024-01-01"}
{"ID":"def456","Names":"db","Image":"postgres:16","Status":"Exited (0) 1 hour ago","State":"exited","Ports":"","CreatedAt":"2024-01-01"}

this-line-is-not-json-and-must-be-skipped
{"ID":"ghi789","Names":"cache","Image":"redis","Status":"Up","State":"running","Ports":"","CreatedAt":"2024-01-02"}
`)

	got, err := parseDockerPS(out)
	if err != nil {
		t.Fatalf("parseDockerPS error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(containers) = %d, want 3 (blank and malformed lines skipped)", len(got))
	}
	if got[0].Name != "web" {
		t.Errorf("Name[0] = %q, want %q (leading slash trimmed)", got[0].Name, "web")
	}
	if got[1].State != "exited" {
		t.Errorf("State[1] = %q, want %q", got[1].State, "exited")
	}
}

func TestParseDockerPS_LongPortLineNotTruncated(t *testing.T) {
	// A container with many published ports yields a line larger than the
	// default 64 KB scanner token limit; it must still parse.
	longPorts := strings.Repeat("0.0.0.0:80->80/tcp, ", 5000) // ~100 KB
	line := `{"ID":"x","Names":"big","Image":"img","Status":"Up","State":"running","Ports":"` +
		longPorts + `","CreatedAt":"t"}`

	got, err := parseDockerPS([]byte(line))
	if err != nil {
		t.Fatalf("parseDockerPS error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(containers) = %d, want 1 (long line must not be dropped)", len(got))
	}
}
