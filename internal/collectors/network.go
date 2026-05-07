package collectors

import (
	"context"
	"fmt"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// sockDGRAM matches syscall.SOCK_DGRAM — used to identify UDP connections
// from gopsutil's ConnectionStat.Type field.
const sockDGRAM = 2

type NetworkCollector struct{}

func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{}
}

func (c *NetworkCollector) Name() string            { return "network" }
func (c *NetworkCollector) Interval() time.Duration { return 2 * time.Second }

func (c *NetworkCollector) Collect(ctx context.Context) (interface{}, error) {
	data := NetworkData{}

	// Interfaces + IO counters
	ifaces, err := psnet.InterfacesWithContext(ctx)
	if err == nil {
		ioCounters, _ := psnet.IOCountersWithContext(ctx, true)
		counterMap := make(map[string]psnet.IOCountersStat)
		for _, c := range ioCounters {
			counterMap[c.Name] = c
		}

		for _, iface := range ifaces {
			addrs := make([]string, 0, len(iface.Addrs))
			for _, a := range iface.Addrs {
				addrs = append(addrs, a.Addr)
			}

			ni := NetInterface{
				Name:      iface.Name,
				Addresses: addrs,
				Flags:     iface.Flags,
			}
			if cnt, ok := counterMap[iface.Name]; ok {
				ni.BytesSent = cnt.BytesSent
				ni.BytesRecv = cnt.BytesRecv
				ni.PacketsSent = cnt.PacketsSent
				ni.PacketsRecv = cnt.PacketsRecv
			}
			data.Interfaces = append(data.Interfaces, ni)
		}
	}

	// Active connections (best-effort — may need elevated perms for PIDs)
	conns, err := psnet.ConnectionsWithContext(ctx, "inet")
	if err == nil {
		pidNameCache := make(map[int32]string)
		for _, conn := range conns {
			if conn.Status == "NONE" {
				continue
			}

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

			local := fmt.Sprintf("%s:%d", conn.Laddr.IP, conn.Laddr.Port)
			remote := ""
			if conn.Raddr.Port > 0 {
				remote = fmt.Sprintf("%s:%d", conn.Raddr.IP, conn.Raddr.Port)
			}

			proto := "TCP"
			if conn.Type == sockDGRAM {
				proto = "UDP"
			}

			data.Connections = append(data.Connections, NetConnection{
				Protocol:    proto,
				LocalAddr:   local,
				RemoteAddr:  remote,
				State:       conn.Status,
				PID:         conn.Pid,
				ProcessName: procName,
			})
		}
	}

	return data, nil
}
