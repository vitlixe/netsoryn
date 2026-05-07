package collectors

import (
	"bufio"
	"bytes"
	"context"
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
