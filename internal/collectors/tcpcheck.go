package collectors

import (
	"context"
	"net"
	"time"
)

// TCPResult holds the outcome of a single TCP connect attempt.
type TCPResult struct {
	Target  string        `json:"target"`
	Open    bool          `json:"open"`
	Elapsed time.Duration `json:"elapsed"`
	Error   string        `json:"error,omitempty"`
}

// CheckTCP attempts a TCP connection to target ("host:port") within timeout and
// reports whether the port accepted the connection, plus how long it took. The
// connection is closed immediately; this is a reachability probe, not a scan.
func CheckTCP(ctx context.Context, target string, timeout time.Duration) TCPResult {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	result := TCPResult{Target: target}

	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := net.Dialer{}
	start := time.Now()
	conn, err := dialer.DialContext(dctx, "tcp", target)
	result.Elapsed = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}
	_ = conn.Close()
	result.Open = true
	return result
}
