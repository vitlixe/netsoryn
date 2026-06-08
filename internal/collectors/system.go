package collectors

import (
	"context"
	"os"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemCollector struct{}

func NewSystemCollector() *SystemCollector {
	return &SystemCollector{}
}

func (c *SystemCollector) Name() string            { return "system" }
func (c *SystemCollector) Interval() time.Duration { return 2 * time.Second }

func (c *SystemCollector) Collect(ctx context.Context) (interface{}, error) {
	data := SystemData{}

	// hostname
	if h, err := os.Hostname(); err == nil {
		data.Hostname = h
	}

	// CPU per-core and total
	perCPU, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, true)
	if err == nil {
		data.CPUPercents = perCPU
	}
	total, err := cpu.PercentWithContext(ctx, 0, false)
	if err == nil && len(total) > 0 {
		data.CPUTotal = total[0]
	}

	// Memory
	vmStat, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		data.MemTotal = vmStat.Total
		data.MemUsed = vmStat.Used
		data.MemPercent = vmStat.UsedPercent
	}
	swapStat, err := mem.SwapMemoryWithContext(ctx)
	if err == nil {
		data.SwapTotal = swapStat.Total
		data.SwapUsed = swapStat.Used
		data.SwapPercent = swapStat.UsedPercent
	}

	// Disk partitions
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err == nil {
		for _, p := range parts {
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			data.Disks = append(data.Disks, DiskStat{
				Mountpoint:  p.Mountpoint,
				Total:       usage.Total,
				Used:        usage.Used,
				Free:        usage.Free,
				UsedPercent: usage.UsedPercent,
				Fstype:      p.Fstype,
			})
		}
	}

	// Disk I/O counters (cumulative; the dashboard derives per-second rates
	// from the delta between two samples). Sorted for stable output.
	if ioCounters, err := disk.IOCountersWithContext(ctx); err == nil {
		names := make([]string, 0, len(ioCounters))
		for name := range ioCounters {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			st := ioCounters[name]
			data.DiskIO = append(data.DiskIO, DiskIOStat{
				Name:       name,
				ReadBytes:  st.ReadBytes,
				WriteBytes: st.WriteBytes,
			})
		}
	}

	// Load average — not supported on Windows; zero values are left as-is.
	avg, err := load.AvgWithContext(ctx)
	if err == nil {
		data.LoadAvg1 = avg.Load1
		data.LoadAvg5 = avg.Load5
		data.LoadAvg15 = avg.Load15
		data.LoadAvgSupported = true
	}

	// Uptime
	uptime, err := host.UptimeWithContext(ctx)
	if err == nil {
		data.UptimeSeconds = uptime
	}

	return data, nil
}
