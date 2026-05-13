package collectors

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type ServiceCollector struct{}

func NewServiceCollector() *ServiceCollector {
	return &ServiceCollector{}
}

func (c *ServiceCollector) Name() string            { return "services" }
func (c *ServiceCollector) Interval() time.Duration { return 5 * time.Second }

func (c *ServiceCollector) Collect(ctx context.Context) (interface{}, error) {
	platform := runtime.GOOS
	switch platform {
	case "linux":
		return c.collectSystemd(ctx, platform)
	case "darwin":
		return c.collectLaunchd(ctx, platform)
	case "windows":
		return c.collectSCM(ctx, platform)
	default:
		return ServiceData{Platform: platform}, nil
	}
}

func (c *ServiceCollector) collectSystemd(ctx context.Context, platform string) (ServiceData, error) {
	cmd := exec.CommandContext(ctx,
		"systemctl", "list-units",
		"--type=service",
		"--all",
		"--no-pager",
		"--plain",
		"--no-legend",
	)
	out, err := cmd.Output()
	if err != nil {
		// systemctl might not exist or no permissions
		return ServiceData{Platform: platform}, nil
	}

	var services []ServiceStat
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		if !strings.HasSuffix(name, ".service") {
			continue
		}
		name = strings.TrimSuffix(name, ".service")

		desc := ""
		if len(fields) >= 5 {
			desc = strings.Join(fields[4:], " ")
		}

		services = append(services, ServiceStat{
			Name:        name,
			LoadState:   fields[1],
			ActiveState: fields[2],
			SubState:    fields[3],
			Description: desc,
		})
	}

	return ServiceData{Services: services, Platform: platform}, nil
}

func (c *ServiceCollector) collectLaunchd(ctx context.Context, platform string) (ServiceData, error) {
	cmd := exec.CommandContext(ctx, "launchctl", "list")
	out, err := cmd.Output()
	if err != nil {
		return ServiceData{Platform: platform}, nil
	}

	var services []ServiceStat
	scanner := bufio.NewScanner(bytes.NewReader(out))
	// skip header line
	scanner.Scan()
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid := fields[0]
		name := fields[2]
		active := "inactive"
		substate := "stopped"
		if pid != "-" {
			active = "active"
			substate = "running"
		}
		services = append(services, ServiceStat{
			Name:        name,
			LoadState:   "loaded",
			ActiveState: active,
			SubState:    substate,
		})
	}

	return ServiceData{Services: services, Platform: platform}, nil
}

func (c *ServiceCollector) collectSCM(ctx context.Context, platform string) (ServiceData, error) {
	cmd := exec.CommandContext(ctx,
		"powershell",
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		`Get-Service | Select-Object Name,DisplayName,@{Name='Status';Expression={$_.Status.ToString()}} | ConvertTo-Json -Compress`,
	)
	out, err := cmd.Output()
	if err != nil {
		return ServiceData{Platform: platform}, nil
	}
	return parseSCMJSON(out, platform)
}

type scmEntry struct {
	Name        string `json:"Name"`
	DisplayName string `json:"DisplayName"`
	Status      string `json:"Status"`
}

func parseSCMJSON(out []byte, platform string) (ServiceData, error) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return ServiceData{Platform: platform}, nil
	}

	var entries []scmEntry
	if out[0] == '[' {
		if err := json.Unmarshal(out, &entries); err != nil {
			return ServiceData{Platform: platform}, nil
		}
	} else {
		var single scmEntry
		if err := json.Unmarshal(out, &single); err != nil {
			return ServiceData{Platform: platform}, nil
		}
		entries = []scmEntry{single}
	}

	services := make([]ServiceStat, 0, len(entries))
	for _, e := range entries {
		sub, active := scmStatusToStates(e.Status)
		services = append(services, ServiceStat{
			Name:        e.Name,
			LoadState:   "loaded",
			ActiveState: active,
			SubState:    sub,
			Description: e.DisplayName,
		})
	}
	return ServiceData{Services: services, Platform: platform}, nil
}

func scmStatusToStates(status string) (subState, activeState string) {
	switch status {
	case "Running", "4":
		return "running", "active"
	case "Stopped", "1":
		return "stopped", "inactive"
	case "Paused", "7":
		return "paused", "active"
	case "StartPending", "2", "StopPending", "3", "ContinuePending", "5", "PausePending", "6":
		return "pending", "active"
	default:
		return strings.ToLower(status), "unknown"
	}
}
