package collectors

import (
	"context"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type ProcessCollector struct {
	limit int
}

func NewProcessCollector(limit int) *ProcessCollector {
	if limit <= 0 {
		limit = 50
	}
	return &ProcessCollector{limit: limit}
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

		cpuPct, _ := p.CPUPercentWithContext(ctx)
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
			CPUPercent: cpuPct,
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
