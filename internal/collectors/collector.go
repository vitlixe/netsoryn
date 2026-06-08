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
	Processes []ProcessStat
}

type ProcessStat struct {
	PID        int32
	Name       string
	Username   string
	CPUPercent float64
	CPUTime    float64 // cumulative CPU seconds; used to derive instantaneous CPUPercent
	MemPercent float32
	MemRSS     uint64
	Status     string
	Threads    int32
	Command    string
}

// NetworkData holds interface stats and active connections.
type NetworkData struct {
	Interfaces  []NetInterface
	Connections []NetConnection
}

type NetInterface struct {
	Name        string
	Addresses   []string
	BytesSent   uint64
	BytesRecv   uint64
	PacketsSent uint64
	PacketsRecv uint64
	Flags       []string
}

type NetConnection struct {
	Protocol    string
	LocalAddr   string
	RemoteAddr  string
	State       string
	PID         int32
	ProcessName string
}

// PortData holds listening ports with their owning processes.
type PortData struct {
	Ports []PortStat
}

type PortStat struct {
	Port     uint32
	Protocol string
	Address  string
	PID      int32
	Process  string
	State    string
}

// ServiceData holds systemd (or launchd) service states.
type ServiceData struct {
	Services []ServiceStat
	Platform string
}

type ServiceStat struct {
	Name        string
	LoadState   string
	ActiveState string
	SubState    string
	Description string
}

// DockerData holds container info.
type DockerData struct {
	Containers []ContainerStat
	Available  bool
}

type ContainerStat struct {
	ID      string
	Name    string
	Image   string
	Status  string
	State   string
	Ports   string
	Created string
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
