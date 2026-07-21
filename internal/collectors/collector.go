package collectors

import (
	"time"
)

// SystemData holds CPU, memory, disk, and load metrics.
type SystemData struct {
	CPUPercents      []float64    `json:"cpu_percents"`
	CPUTotal         float64      `json:"cpu_total"`
	MemTotal         uint64       `json:"mem_total"`
	MemUsed          uint64       `json:"mem_used"`
	MemPercent       float64      `json:"mem_percent"`
	SwapTotal        uint64       `json:"swap_total"`
	SwapUsed         uint64       `json:"swap_used"`
	SwapPercent      float64      `json:"swap_percent"`
	Disks            []DiskStat   `json:"disks"`
	LoadAvg1         float64      `json:"load_avg_1"`
	LoadAvg5         float64      `json:"load_avg_5"`
	LoadAvg15        float64      `json:"load_avg_15"`
	LoadAvgSupported bool         `json:"load_avg_supported"`
	UptimeSeconds    uint64       `json:"uptime_seconds"`
	Hostname         string       `json:"hostname"`
	DiskIO           []DiskIOStat `json:"disk_io"`
}

type DiskStat struct {
	Mountpoint  string  `json:"mountpoint"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	Fstype      string  `json:"fstype"`
}

// DiskIOStat holds cumulative I/O byte counters for one block device. The
// dashboard derives read/write throughput from the delta between two samples.
type DiskIOStat struct {
	Name       string `json:"name"`
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
}

// ProcessData holds a snapshot of running processes.
type ProcessData struct {
	Processes []ProcessStat `json:"processes"`
}

type ProcessStat struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	Username   string  `json:"username"`
	CPUPercent float64 `json:"cpu_percent"`
	CPUTime    float64 `json:"cpu_time"` // cumulative CPU seconds; used to derive instantaneous CPUPercent
	MemPercent float32 `json:"mem_percent"`
	MemRSS     uint64  `json:"mem_rss"`
	Status     string  `json:"status"`
	Threads    int32   `json:"threads"`
	Command    string  `json:"command"`
}

// NetworkData holds interface stats and active connections.
type NetworkData struct {
	Interfaces  []NetInterface  `json:"interfaces"`
	Connections []NetConnection `json:"connections"`
}

type NetInterface struct {
	Name        string   `json:"name"`
	Addresses   []string `json:"addresses"`
	BytesSent   uint64   `json:"bytes_sent"`
	BytesRecv   uint64   `json:"bytes_recv"`
	PacketsSent uint64   `json:"packets_sent"`
	PacketsRecv uint64   `json:"packets_recv"`
	Flags       []string `json:"flags"`
}

type NetConnection struct {
	Protocol    string `json:"protocol"`
	LocalAddr   string `json:"local_addr"`
	RemoteAddr  string `json:"remote_addr"`
	State       string `json:"state"`
	PID         int32  `json:"pid"`
	ProcessName string `json:"process_name"`
}

// PortData holds listening ports with their owning processes.
type PortData struct {
	Ports []PortStat `json:"ports"`
}

type PortStat struct {
	Port     uint32 `json:"port"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	PID      int32  `json:"pid"`
	Process  string `json:"process"`
	State    string `json:"state"`
}

// ServiceData holds systemd (or launchd) service states.
type ServiceData struct {
	Services []ServiceStat `json:"services"`
	Platform string        `json:"platform"`
}

type ServiceStat struct {
	Name        string `json:"name"`
	LoadState   string `json:"load_state"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
	Description string `json:"description"`
}

// DockerData holds container info.
type DockerData struct {
	Containers []ContainerStat `json:"containers"`
	Available  bool            `json:"available"`
}

type ContainerStat struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	State   string `json:"state"`
	Ports   string `json:"ports"`
	Created string `json:"created"`
}

// DNSResult holds DNS resolution results for a single domain.
type DNSResult struct {
	Domain      string
	Server      string
	ARecords    []string
	AAAARecords []string
	MXRecords   []string
	NSRecords   []string
	CNAMERecord string
	TTL         uint32
	Elapsed     time.Duration
	Error       string
}

// HTTPResult holds the result of an HTTP endpoint check.
type HTTPResult struct {
	URL         string
	StatusCode  int
	StatusText  string
	Elapsed     time.Duration
	TLSValid    bool
	TLSExpiry   string
	TLSIssuer   string
	Redirect    string
	ContentType string
	Error       string
}
