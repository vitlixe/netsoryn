package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vitlixe/netsoryn/internal/collectors"
	"github.com/vitlixe/netsoryn/internal/config"
)

// dumpOutput is the top-level JSON document produced by `netsoryn dump`.
// Sections are added as more collectors are wired in; omitempty keeps the
// document compact when a section is unavailable.
type dumpOutput struct {
	System    *collectors.SystemData  `json:"system,omitempty"`
	Processes *collectors.ProcessData `json:"processes,omitempty"`
	Network   *collectors.NetworkData `json:"network,omitempty"`
	Ports     *collectors.PortData    `json:"ports,omitempty"`
	Services  *collectors.ServiceData `json:"services,omitempty"`
	Docker    *collectors.DockerData  `json:"docker,omitempty"`
}

func dumpCmd(cfgFile *string) *cobra.Command {
	var pretty bool
	var format string
	var sections string

	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Print a one-shot system snapshot (json, md, or text)",
		Long: `Collect a single snapshot of system metrics and print it to stdout, then
exit. Unlike the interactive UI this needs no terminal, so it works over SSH,
in scripts, and in CI.

Formats (--format): json (default, machine-readable), md (Markdown, for
tickets/wikis/chat), text (plain table for the terminal).

Sections (--sections): comma-separated names. Default: system.
Available sections: system, processes, network, ports, services, docker, all.

  netsoryn dump > snapshot.json
  netsoryn dump --sections system,ports,services --pretty
  netsoryn dump --sections all --format json
  netsoryn dump --format md > snapshot.md
  netsoryn dump --format text
  ssh host 'netsoryn dump' | jq .system`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cfg, err := config.Load(*cfgFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			selected, err := parseDumpSections(sections)
			if err != nil {
				return err
			}
			out, err := dumpSnapshot(ctx, cfg, selected)
			if err != nil {
				return err
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetEscapeHTML(false)
				if pretty {
					enc.SetIndent("", "  ")
				}
				return enc.Encode(out)
			case "md", "markdown":
				_, err := fmt.Fprint(os.Stdout, renderMarkdownDump(out))
				return err
			case "text", "txt":
				_, err := fmt.Fprint(os.Stdout, renderTextDump(out))
				return err
			default:
				return fmt.Errorf("unknown format %q (want json, md, or text)", format)
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "json", "output format: json, md, or text")
	cmd.Flags().StringVar(&sections, "sections", "system", "comma-separated sections: system, processes, network, ports, services, docker, all")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent the output (json format only)")
	return cmd
}

type dumpSections map[string]bool

var allDumpSections = []string{"system", "processes", "network", "ports", "services", "docker"}

func parseDumpSections(raw string) (dumpSections, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "system"
	}

	sections := make(dumpSections)
	for _, part := range strings.Split(raw, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		if name == "all" {
			for _, section := range allDumpSections {
				sections[section] = true
			}
			continue
		}
		normalized, ok := normalizeDumpSection(name)
		if !ok {
			return nil, fmt.Errorf("unknown dump section %q (want one of: %s)", name, strings.Join(allDumpSections, ", "))
		}
		sections[normalized] = true
	}
	if len(sections) == 0 {
		sections["system"] = true
	}
	return sections, nil
}

func normalizeDumpSection(name string) (string, bool) {
	switch name {
	case "system", "sys":
		return "system", true
	case "processes", "process", "proc":
		return "processes", true
	case "network", "net":
		return "network", true
	case "ports", "port":
		return "ports", true
	case "services", "service", "svc":
		return "services", true
	case "docker":
		return "docker", true
	default:
		return "", false
	}
}

func dumpSnapshot(ctx context.Context, cfg *config.Config, sections dumpSections) (dumpOutput, error) {
	var out dumpOutput

	if sections["system"] {
		sys, err := systemSnapshot(ctx)
		if err != nil {
			return out, err
		}
		out.System = &sys
	}
	if sections["processes"] {
		procs, err := processSnapshot(ctx, cfg.ProcessLimit)
		if err != nil {
			return out, err
		}
		out.Processes = &procs
	}
	if sections["network"] {
		netData, err := networkSnapshot(ctx)
		if err != nil {
			return out, err
		}
		out.Network = &netData
	}
	if sections["ports"] {
		ports, err := portSnapshot(ctx, cfg.PortsListenOnly)
		if err != nil {
			return out, err
		}
		out.Ports = &ports
	}
	if sections["services"] {
		services, err := serviceSnapshot(ctx)
		if err != nil {
			return out, err
		}
		out.Services = &services
	}
	if sections["docker"] {
		docker, err := dockerSnapshot(ctx, cfg.DockerSocket)
		if err != nil {
			return out, err
		}
		out.Docker = &docker
	}

	return out, nil
}

// systemSnapshot collects a single SystemData sample. It overrides CPUTotal with
// the mean of the per-core percentages: the collector's aggregate total uses a
// non-blocking sample that is only meaningful across repeated UI refreshes,
// whereas the per-core values come from a real 200ms sample and so give an
// accurate instantaneous reading for a one-shot dump.
func systemSnapshot(ctx context.Context) (collectors.SystemData, error) {
	raw, err := collectors.NewSystemCollector().Collect(ctx)
	if err != nil {
		return collectors.SystemData{}, fmt.Errorf("collecting system data: %w", err)
	}
	sys, ok := raw.(collectors.SystemData)
	if !ok {
		return collectors.SystemData{}, fmt.Errorf("unexpected system collector result type %T", raw)
	}
	if len(sys.CPUPercents) > 0 {
		sys.CPUTotal = mean(sys.CPUPercents)
	}
	return sys, nil
}

func processSnapshot(ctx context.Context, limit int) (collectors.ProcessData, error) {
	raw, err := collectors.NewProcessCollector().Collect(ctx)
	if err != nil {
		return collectors.ProcessData{}, fmt.Errorf("collecting process data: %w", err)
	}
	procs, ok := raw.(collectors.ProcessData)
	if !ok {
		return collectors.ProcessData{}, fmt.Errorf("unexpected process collector result type %T", raw)
	}
	sort.Slice(procs.Processes, func(i, j int) bool {
		return procs.Processes[i].MemRSS > procs.Processes[j].MemRSS
	})
	if limit > 0 && len(procs.Processes) > limit {
		procs.Processes = procs.Processes[:limit]
	}
	return procs, nil
}

func networkSnapshot(ctx context.Context) (collectors.NetworkData, error) {
	raw, err := collectors.NewNetworkCollector().Collect(ctx)
	if err != nil {
		return collectors.NetworkData{}, fmt.Errorf("collecting network data: %w", err)
	}
	data, ok := raw.(collectors.NetworkData)
	if !ok {
		return collectors.NetworkData{}, fmt.Errorf("unexpected network collector result type %T", raw)
	}
	return data, nil
}

func portSnapshot(ctx context.Context, listenOnly bool) (collectors.PortData, error) {
	raw, err := collectors.NewPortCollector(listenOnly).Collect(ctx)
	if err != nil {
		return collectors.PortData{}, fmt.Errorf("collecting port data: %w", err)
	}
	data, ok := raw.(collectors.PortData)
	if !ok {
		return collectors.PortData{}, fmt.Errorf("unexpected port collector result type %T", raw)
	}
	return data, nil
}

func serviceSnapshot(ctx context.Context) (collectors.ServiceData, error) {
	raw, err := collectors.NewServiceCollector().Collect(ctx)
	if err != nil {
		return collectors.ServiceData{}, fmt.Errorf("collecting service data: %w", err)
	}
	data, ok := raw.(collectors.ServiceData)
	if !ok {
		return collectors.ServiceData{}, fmt.Errorf("unexpected service collector result type %T", raw)
	}
	return data, nil
}

func dockerSnapshot(ctx context.Context, socketPath string) (collectors.DockerData, error) {
	raw, err := collectors.NewDockerCollector(socketPath).Collect(ctx)
	if err != nil {
		return collectors.DockerData{}, fmt.Errorf("collecting docker data: %w", err)
	}
	data, ok := raw.(collectors.DockerData)
	if !ok {
		return collectors.DockerData{}, fmt.Errorf("unexpected docker collector result type %T", raw)
	}
	return data, nil
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}
