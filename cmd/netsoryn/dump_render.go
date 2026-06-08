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
	var b strings.Builder

	host := s.Hostname
	if host == "" {
		host = "host"
	}
	fmt.Fprintf(&b, "# 🖥 %s — netsoryn snapshot\n", host)
	fmt.Fprintf(&b, "_%s · uptime %s_\n\n", time.Now().Format("2006-01-02 15:04"), humanUptime(s.UptimeSeconds))

	b.WriteString("## System\n\n")
	b.WriteString("| Metric | Value |\n|--------|-------|\n")
	fmt.Fprintf(&b, "| CPU | %.1f%% |\n", s.CPUTotal)
	fmt.Fprintf(&b, "| Memory | %s / %s · %.0f%% |\n", humanBytes(s.MemUsed), humanBytes(s.MemTotal), s.MemPercent)
	fmt.Fprintf(&b, "| Swap | %s / %s · %.0f%% |\n", humanBytes(s.SwapUsed), humanBytes(s.SwapTotal), s.SwapPercent)
	if s.LoadAvgSupported {
		fmt.Fprintf(&b, "| Load avg | %.2f · %.2f · %.2f |\n", s.LoadAvg1, s.LoadAvg5, s.LoadAvg15)
	}
	b.WriteString("\n")

	if len(s.CPUPercents) > 0 {
		b.WriteString("## CPU cores\n\n| Core | Usage |\n|------|-------|\n")
		for i, p := range s.CPUPercents {
			fmt.Fprintf(&b, "| %d | %.1f%% |\n", i, p)
		}
		b.WriteString("\n")
	}

	if len(s.Disks) > 0 {
		b.WriteString("## Disks\n\n| Mount | Used | Total | % | FS |\n|-------|------|-------|---|----|\n")
		for _, d := range s.Disks {
			fmt.Fprintf(&b, "| %s | %s | %s | %.1f%% | %s |\n",
				d.Mountpoint, humanBytes(d.Used), humanBytes(d.Total), d.UsedPercent, d.Fstype)
		}
		b.WriteString("\n")
	}

	if len(s.DiskIO) > 0 {
		b.WriteString("## Disk I/O (since boot)\n\n| Device | Read | Write |\n|--------|------|-------|\n")
		for _, io := range s.DiskIO {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", io.Name, humanBytes(io.ReadBytes), humanBytes(io.WriteBytes))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderText formats a system snapshot as a plain aligned table for reading
// directly in the terminal.
func renderText(s collectors.SystemData) string {
	var b strings.Builder

	host := s.Hostname
	if host == "" {
		host = "host"
	}
	fmt.Fprintf(&b, "%s — %s · uptime %s\n\n", host, time.Now().Format("2006-01-02 15:04"), humanUptime(s.UptimeSeconds))

	b.WriteString("System\n")
	fmt.Fprintf(&b, "  CPU      %.1f%%\n", s.CPUTotal)
	fmt.Fprintf(&b, "  Memory   %s / %s (%.0f%%)\n", humanBytes(s.MemUsed), humanBytes(s.MemTotal), s.MemPercent)
	fmt.Fprintf(&b, "  Swap     %s / %s (%.0f%%)\n", humanBytes(s.SwapUsed), humanBytes(s.SwapTotal), s.SwapPercent)
	if s.LoadAvgSupported {
		fmt.Fprintf(&b, "  Load     %.2f %.2f %.2f\n", s.LoadAvg1, s.LoadAvg5, s.LoadAvg15)
	}
	b.WriteString("\n")

	if len(s.Disks) > 0 {
		b.WriteString("Disks\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
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
		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "  DEVICE\tREAD\tWRITE")
		for _, io := range s.DiskIO {
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", io.Name, humanBytes(io.ReadBytes), humanBytes(io.WriteBytes))
		}
		_ = tw.Flush()
		b.WriteString("\n")
	}

	return b.String()
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
