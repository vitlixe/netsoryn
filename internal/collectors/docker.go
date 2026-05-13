package collectors

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

type DockerCollector struct {
	socketPath string
}

func NewDockerCollector(socketPath string) *DockerCollector {
	return &DockerCollector{socketPath: socketPath}
}

func (c *DockerCollector) Name() string            { return "docker" }
func (c *DockerCollector) Interval() time.Duration { return 3 * time.Second }

type dockerPsEntry struct {
	ID      string `json:"ID"`
	Names   string `json:"Names"`
	Image   string `json:"Image"`
	Status  string `json:"Status"`
	State   string `json:"State"`
	Ports   string `json:"Ports"`
	Created string `json:"CreatedAt"`
}

// dockerArgs prepends --host <socketPath> to args when socketPath is non-empty.
// An empty socketPath leaves args unchanged, preserving default Docker CLI behaviour
// (DOCKER_HOST env or platform default socket).
func dockerArgs(socketPath string, args ...string) []string {
	if socketPath == "" {
		return args
	}
	out := make([]string, 0, len(args)+2)
	out = append(out, "--host", socketPath)
	out = append(out, args...)
	return out
}

func (c *DockerCollector) Collect(ctx context.Context) (interface{}, error) {
	// Use docker CLI — works without SDK dependency
	args := dockerArgs(c.socketPath,
		"ps", "--all",
		"--format", `{"ID":"{{.ID}}","Names":"{{.Names}}","Image":"{{.Image}}","Status":"{{.Status}}","State":"{{.State}}","Ports":"{{.Ports}}","CreatedAt":"{{.CreatedAt}}"}`,
	)
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.Output()
	if err != nil {
		// Docker not available or not running
		return DockerData{Available: false}, nil
	}

	var containers []ContainerStat
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry dockerPsEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		containers = append(containers, ContainerStat{
			ID:      entry.ID,
			Name:    strings.TrimPrefix(entry.Names, "/"),
			Image:   entry.Image,
			Status:  entry.Status,
			State:   entry.State,
			Ports:   entry.Ports,
			Created: entry.Created,
		})
	}

	return DockerData{Containers: containers, Available: true}, nil
}
