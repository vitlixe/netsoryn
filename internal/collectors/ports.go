package collectors

import (
	"context"
	"fmt"
	"sort"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type PortCollector struct {
	listenOnly bool
}

func NewPortCollector(listenOnly bool) *PortCollector {
	return &PortCollector{listenOnly: listenOnly}
}

func (c *PortCollector) Name() string            { return "ports" }
func (c *PortCollector) Interval() time.Duration { return 3 * time.Second }

func (c *PortCollector) Collect(ctx context.Context) (interface{}, error) {
	kind := "inet"
	conns, err := psnet.ConnectionsWithContext(ctx, kind)
	if err != nil {
		return PortData{}, err
	}

	pidNameCache := make(map[int32]string)
	seen := make(map[string]struct{})
	ports := make([]PortStat, 0)

	for _, conn := range conns {
		if c.listenOnly && conn.Status != "LISTEN" {
			continue
		}
		if conn.Laddr.Port == 0 {
			continue
		}

		key := fmt.Sprintf("%d-%d-%d", conn.Type, conn.Pid, conn.Laddr.Port)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		procName := ""
		if conn.Pid > 0 {
			if name, ok := pidNameCache[conn.Pid]; ok {
				procName = name
			} else {
				if p, err := process.NewProcess(conn.Pid); err == nil {
					if name, err := p.Name(); err == nil {
						procName = name
						pidNameCache[conn.Pid] = name
					}
				}
			}
		}

		addr := conn.Laddr.IP
		if addr == "" {
			addr = "0.0.0.0"
		}

		proto := "TCP"
		if conn.Type == sockDGRAM {
			proto = "UDP"
		}

		ports = append(ports, PortStat{
			Port:     conn.Laddr.Port,
			Protocol: proto,
			Address:  addr,
			State:    conn.Status,
			PID:      conn.Pid,
			Process:  procName,
		})
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Port < ports[j].Port
	})

	return PortData{Ports: ports}, nil
}
