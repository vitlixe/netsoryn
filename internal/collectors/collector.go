package collectors

import (
	"time"
)

// SystemData holds CPU, memory, disk, and load metrics.
type SystemData struct {
	CPUPercents      []float64
	CPUTotal         float64
	MemTotal         uint64
	MemUsed          uint64
	MemPercent       float64
	SwapTotal        uint64
	SwapUsed         uint64
	SwapPercent      float64
	Disks            []DiskStat
	LoadAvg1         float64
	LoadAvg5         float64
	LoadAvg15        float64
	LoadAvgSupported bool
	UptimeSeconds    uint64
	Hostname         string
}

type DiskStat struct {
	Mountpoint  string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
	Fstype      string
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
