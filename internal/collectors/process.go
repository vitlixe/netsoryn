package collectors

import (
	"context"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type ProcessCollector struct{}

func NewProcessCollector() *ProcessCollector {
	return &ProcessCollector{}
}

func (c *ProcessCollector) Name() string            { return "process" }
func (c *ProcessCollector) Interval() time.Duration { return 2 * time.Second }

func (c *ProcessCollector) Collect(ctx context.Context) (interface{}, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	stats := make([]ProcessStat, 0, len(procs))
	for _, p := range procs {
		name, _ := p.NameWithContext(ctx)
		if name == "" {
			continue
		}

		// Cumulative CPU seconds; the view derives an instantaneous percentage
		// from the delta between samples (gopsutil's CPUPercent is a lifetime
		// average, which is misleading for a "top by CPU" listing).
		var cpuTime float64
		if t, err := p.TimesWithContext(ctx); err == nil && t != nil {
			cpuTime = t.User + t.System
		}
		memPct, _ := p.MemoryPercentWithContext(ctx)
		memInfo, _ := p.MemoryInfoWithContext(ctx)
		status, _ := p.StatusWithContext(ctx)
		threads, _ := p.NumThreadsWithContext(ctx)
		username, _ := p.UsernameWithContext(ctx)
		cmdline, _ := p.CmdlineWithContext(ctx)

		var rss uint64
		if memInfo != nil {
			rss = memInfo.RSS
		}

		// shorten username if it includes domain prefix
		if idx := strings.LastIndex(username, "\\"); idx >= 0 {
			username = username[idx+1:]
		}

		stat := ProcessStat{
			PID:        p.Pid,
			Name:       name,
			Username:   username,
			CPUTime:    cpuTime,
			MemPercent: memPct,
			MemRSS:     rss,
			Threads:    threads,
			Command:    cmdline,
		}
		if len(status) > 0 {
			stat.Status = status[0]
		}

		stats = append(stats, stat)
	}

	// Return all processes unsorted — the view applies its own sort + limit.
	return ProcessData{Processes: stats}, nil
}
