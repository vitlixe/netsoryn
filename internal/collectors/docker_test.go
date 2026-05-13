package collectors

import (
	"reflect"
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
