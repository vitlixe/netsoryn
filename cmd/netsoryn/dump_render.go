package main

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/vitlixe/netsoryn/internal/collectors"
)

// renderMarkdown formats a system snapshot as Markdown tables — readable in
// tickets, wikis, and chat where Markdown renders.
func renderMarkdown(s collectors.SystemData) string {
	return renderMarkdownDump(dumpOutput{System: &s})
}

func renderMarkdownDump(out dumpOutput) string {
	var b strings.Builder

	host := "host"
	if out.System != nil && out.System.Hostname != "" {
		host = out.System.Hostname
	}
	uptime := "unknown"
	if out.System != nil {
		uptime = humanUptime(out.System.UptimeSeconds)
	}
	fmt.Fprintf(&b, "# 🖥 %s — netsoryn snapshot\n", host)
	fmt.Fprintf(&b, "_%s · uptime %s_\n\n", time.Now().Format("2006-01-02 15:04"), uptime)

	if out.System != nil {
		renderSystemMarkdown(&b, *out.System)
	}
	if out.Processes != nil {
		renderProcessesMarkdown(&b, *out.Processes)
	}
	if out.Network != nil {
		renderNetworkMarkdown(&b, *out.Network)
	}
	if out.Ports != nil {
		renderPortsMarkdown(&b, *out.Ports)
	}
	if out.Services != nil {
		renderServicesMarkdown(&b, *out.Services)
	}
	if out.Docker != nil {
		renderDockerMarkdown(&b, *out.Docker)
	}

	return b.String()
}

func renderSystemMarkdown(b *strings.Builder, s collectors.SystemData) {
	b.WriteString("## System\n\n")
	b.WriteString("| Metric | Value |\n|--------|-------|\n")
	fmt.Fprintf(b, "| CPU | %.1f%% |\n", s.CPUTotal)
	fmt.Fprintf(b, "| Memory | %s / %s · %.0f%% |\n", humanBytes(s.MemUsed), humanBytes(s.MemTotal), s.MemPercent)
	fmt.Fprintf(b, "| Swap | %s / %s · %.0f%% |\n", humanBytes(s.SwapUsed), humanBytes(s.SwapTotal), s.SwapPercent)
	if s.LoadAvgSupported {
		fmt.Fprintf(b, "| Load avg | %.2f · %.2f · %.2f |\n", s.LoadAvg1, s.LoadAvg5, s.LoadAvg15)
	}
	b.WriteString("\n")

	if len(s.CPUPercents) > 0 {
		b.WriteString("## CPU cores\n\n| Core | Usage |\n|------|-------|\n")
		for i, p := range s.CPUPercents {
			fmt.Fprintf(b, "| %d | %.1f%% |\n", i, p)
		}
		b.WriteString("\n")
	}

	if len(s.Disks) > 0 {
		b.WriteString("## Disks\n\n| Mount | Used | Total | % | FS |\n|-------|------|-------|---|----|\n")
		for _, d := range s.Disks {
			fmt.Fprintf(b, "| %s | %s | %s | %.1f%% | %s |\n",
				d.Mountpoint, humanBytes(d.Used), humanBytes(d.Total), d.UsedPercent, d.Fstype)
		}
		b.WriteString("\n")
	}

	if len(s.DiskIO) > 0 {
		b.WriteString("## Disk I/O (since boot)\n\n| Device | Read | Write |\n|--------|------|-------|\n")
		for _, io := range s.DiskIO {
			fmt.Fprintf(b, "| %s | %s | %s |\n", io.Name, humanBytes(io.ReadBytes), humanBytes(io.WriteBytes))
		}
		b.WriteString("\n")
	}
}

func renderProcessesMarkdown(b *strings.Builder, p collectors.ProcessData) {
	b.WriteString("## Processes\n\n| PID | Name | User | RSS | Memory % | Status |\n|-----|------|------|-----|----------|--------|\n")
	for _, proc := range p.Processes {
		fmt.Fprintf(b, "| %d | %s | %s | %s | %.1f%% | %s |\n",
			proc.PID, mdCell(proc.Name), mdCell(proc.Username), humanBytes(proc.MemRSS), proc.MemPercent, mdCell(proc.Status))
	}
	b.WriteString("\n")
}

func renderNetworkMarkdown(b *strings.Builder, n collectors.NetworkData) {
	if len(n.Interfaces) > 0 {
		b.WriteString("## Network interfaces\n\n| Name | Addresses | Sent | Received | Flags |\n|------|-----------|------|----------|-------|\n")
		for _, iface := range n.Interfaces {
			fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n",
				mdCell(iface.Name), mdCell(strings.Join(iface.Addresses, ", ")), humanBytes(iface.BytesSent),
				humanBytes(iface.BytesRecv), mdCell(strings.Join(iface.Flags, ", ")))
		}
		b.WriteString("\n")
	}
	if len(n.Connections) > 0 {
		b.WriteString("## Network connections\n\n| Proto | Local | Remote | State | PID | Process |\n|-------|-------|--------|-------|-----|---------|\n")
		for _, conn := range n.Connections {
			fmt.Fprintf(b, "| %s | %s | %s | %s | %d | %s |\n",
				conn.Protocol, mdCell(conn.LocalAddr), mdCell(conn.RemoteAddr), mdCell(conn.State), conn.PID, mdCell(conn.ProcessName))
		}
		b.WriteString("\n")
	}
}

func renderPortsMarkdown(b *strings.Builder, p collectors.PortData) {
	b.WriteString("## Ports\n\n| Proto | Address | Port | State | PID | Process |\n|-------|---------|------|-------|-----|---------|\n")
	for _, port := range p.Ports {
		fmt.Fprintf(b, "| %s | %s | %d | %s | %d | %s |\n",
			port.Protocol, mdCell(port.Address), port.Port, mdCell(port.State), port.PID, mdCell(port.Process))
	}
	b.WriteString("\n")
}

func renderServicesMarkdown(b *strings.Builder, s collectors.ServiceData) {
	b.WriteString("## Services\n\n")
	if s.Platform != "" {
		fmt.Fprintf(b, "_Platform: %s_\n\n", s.Platform)
	}
	b.WriteString("| Name | Active | Sub | Description |\n|------|--------|-----|-------------|\n")
	for _, svc := range s.Services {
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n",
			mdCell(svc.Name), mdCell(svc.ActiveState), mdCell(svc.SubState), mdCell(svc.Description))
	}
	b.WriteString("\n")
}

func renderDockerMarkdown(b *strings.Builder, d collectors.DockerData) {
	b.WriteString("## Docker\n\n")
	if !d.Available {
		b.WriteString("Docker is not available.\n\n")
		return
	}
	b.WriteString("| Name | Image | State | Status | Ports |\n|------|-------|-------|--------|-------|\n")
	for _, c := range d.Containers {
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n",
			mdCell(c.Name), mdCell(c.Image), mdCell(c.State), mdCell(c.Status), mdCell(c.Ports))
	}
	b.WriteString("\n")
}

// renderText formats a system snapshot as a plain aligned table for reading
// directly in the terminal.
func renderText(s collectors.SystemData) string {
	return renderTextDump(dumpOutput{System: &s})
}

func renderTextDump(out dumpOutput) string {
	var b strings.Builder

	host := "host"
	if out.System != nil && out.System.Hostname != "" {
		host = out.System.Hostname
	}
	uptime := "unknown"
	if out.System != nil {
		uptime = humanUptime(out.System.UptimeSeconds)
	}
	fmt.Fprintf(&b, "%s — %s · uptime %s\n\n", host, time.Now().Format("2006-01-02 15:04"), uptime)

	if out.System != nil {
		renderSystemText(&b, *out.System)
	}
	if out.Processes != nil {
		renderProcessesText(&b, *out.Processes)
	}
	if out.Network != nil {
		renderNetworkText(&b, *out.Network)
	}
	if out.Ports != nil {
		renderPortsText(&b, *out.Ports)
	}
	if out.Services != nil {
		renderServicesText(&b, *out.Services)
	}
	if out.Docker != nil {
		renderDockerText(&b, *out.Docker)
	}

	return b.String()
}

func renderSystemText(b *strings.Builder, s collectors.SystemData) {
	b.WriteString("System\n")
	fmt.Fprintf(b, "  CPU      %.1f%%\n", s.CPUTotal)
	fmt.Fprintf(b, "  Memory   %s / %s (%.0f%%)\n", humanBytes(s.MemUsed), humanBytes(s.MemTotal), s.MemPercent)
	fmt.Fprintf(b, "  Swap     %s / %s (%.0f%%)\n", humanBytes(s.SwapUsed), humanBytes(s.SwapTotal), s.SwapPercent)
	if s.LoadAvgSupported {
		fmt.Fprintf(b, "  Load     %.2f %.2f %.2f\n", s.LoadAvg1, s.LoadAvg5, s.LoadAvg15)
	}
	b.WriteString("\n")

	if len(s.Disks) > 0 {
		b.WriteString("Disks\n")
		tw := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "  MOUNT\tUSED\tTOTAL\t%\tFS")
		for _, d := range s.Disks {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%.1f%%\t%s\n",
				d.Mountpoint, humanBytes(d.Used), humanBytes(d.Total), d.UsedPercent, d.Fstype)
		}
		_ = tw.Flush()
		b.WriteString("\n")
	}

	if len(s.DiskIO) > 0 {
		b.WriteString("Disk I/O (since boot)\n")
		tw := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "  DEVICE\tREAD\tWRITE")
		for _, io := range s.DiskIO {
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", io.Name, humanBytes(io.ReadBytes), humanBytes(io.WriteBytes))
		}
		_ = tw.Flush()
		b.WriteString("\n")
	}
}

func renderProcessesText(b *strings.Builder, p collectors.ProcessData) {
	b.WriteString("Processes\n")
	tw := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "  PID\tNAME\tUSER\tRSS\tMEM%\tSTATUS")
	for _, proc := range p.Processes {
		fmt.Fprintf(tw, "  %d\t%s\t%s\t%s\t%.1f%%\t%s\n",
			proc.PID, proc.Name, proc.Username, humanBytes(proc.MemRSS), proc.MemPercent, proc.Status)
	}
	_ = tw.Flush()
	b.WriteString("\n")
}

func renderNetworkText(b *strings.Builder, n collectors.NetworkData) {
	if len(n.Interfaces) > 0 {
		b.WriteString("Network interfaces\n")
		tw := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "  NAME\tADDRESSES\tSENT\tRECV\tFLAGS")
		for _, iface := range n.Interfaces {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n",
				iface.Name, strings.Join(iface.Addresses, ", "), humanBytes(iface.BytesSent),
				humanBytes(iface.BytesRecv), strings.Join(iface.Flags, ","))
		}
		_ = tw.Flush()
		b.WriteString("\n")
	}
	if len(n.Connections) > 0 {
		b.WriteString("Network connections\n")
		tw := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "  PROTO\tLOCAL\tREMOTE\tSTATE\tPID\tPROCESS")
		for _, conn := range n.Connections {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%d\t%s\n",
				conn.Protocol, conn.LocalAddr, conn.RemoteAddr, conn.State, conn.PID, conn.ProcessName)
		}
		_ = tw.Flush()
		b.WriteString("\n")
	}
}

func renderPortsText(b *strings.Builder, p collectors.PortData) {
	b.WriteString("Ports\n")
	tw := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "  PROTO\tADDRESS\tPORT\tSTATE\tPID\tPROCESS")
	for _, port := range p.Ports {
		fmt.Fprintf(tw, "  %s\t%s\t%d\t%s\t%d\t%s\n",
			port.Protocol, port.Address, port.Port, port.State, port.PID, port.Process)
	}
	_ = tw.Flush()
	b.WriteString("\n")
}

func renderServicesText(b *strings.Builder, s collectors.ServiceData) {
	if s.Platform != "" {
		fmt.Fprintf(b, "Services (%s)\n", s.Platform)
	} else {
		b.WriteString("Services\n")
	}
	tw := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tACTIVE\tSUB\tDESCRIPTION")
	for _, svc := range s.Services {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", svc.Name, svc.ActiveState, svc.SubState, svc.Description)
	}
	_ = tw.Flush()
	b.WriteString("\n")
}

func renderDockerText(b *strings.Builder, d collectors.DockerData) {
	b.WriteString("Docker\n")
	if !d.Available {
		b.WriteString("  unavailable\n\n")
		return
	}
	tw := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tIMAGE\tSTATE\tSTATUS\tPORTS")
	for _, c := range d.Containers {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n", c.Name, c.Image, c.State, c.Status, c.Ports)
	}
	_ = tw.Flush()
	b.WriteString("\n")
}

func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// humanBytes formats a byte count as a short human-readable string (KB, MB, …).
func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit && exp < len(units)-1; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", float64(b)/float64(div), units[exp])
}

// humanUptime formats a second count as "5d 3h 22m".
func humanUptime(s uint64) string {
	d := s / 86400
	h := (s % 86400) / 3600
	m := (s % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh %dm", d, h, m)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
