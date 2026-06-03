package collectors

import (
	"testing"

	psnet "github.com/shirou/gopsutil/v3/net"
)

func TestIsListening(t *testing.T) {
	const sockStream = 1 // syscall.SOCK_STREAM

	tcp := func(status string) psnet.ConnectionStat {
		return psnet.ConnectionStat{Type: sockStream, Status: status}
	}
	udp := func(remotePort uint32) psnet.ConnectionStat {
		return psnet.ConnectionStat{
			Type:   sockDGRAM,
			Status: "NONE", // UDP has no TCP-style state
			Raddr:  psnet.Addr{Port: remotePort},
		}
	}

	cases := []struct {
		name string
		conn psnet.ConnectionStat
		want bool
	}{
		{"tcp listening", tcp("LISTEN"), true},
		{"tcp established", tcp("ESTABLISHED"), false},
		{"tcp time_wait", tcp("TIME_WAIT"), false},
		{"udp bound without peer", udp(0), true},
		{"udp connected to peer", udp(53), false},
	}

	for _, c := range cases {
		if got := isListening(c.conn); got != c.want {
			t.Errorf("%s: isListening = %v, want %v", c.name, got, c.want)
		}
	}
}
