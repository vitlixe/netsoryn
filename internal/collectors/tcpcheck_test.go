package collectors_test

import (
	"context"
	"net"
	"testing"
	"time"

	. "github.com/vitlixe/netsoryn/internal/collectors"
)

func TestCheckTCP_Open(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	r := CheckTCP(context.Background(), ln.Addr().String(), 2*time.Second)
	if !r.Open {
		t.Errorf("expected open, got closed (error=%q)", r.Error)
	}
	if r.Error != "" {
		t.Errorf("expected no error, got %q", r.Error)
	}
}

func TestCheckTCP_Closed(t *testing.T) {
	// Bind then immediately release a port so it is almost certainly closed.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	r := CheckTCP(context.Background(), addr, 1*time.Second)
	if r.Open {
		t.Error("expected closed, got open")
	}
	if r.Error == "" {
		t.Error("expected an error for a closed port")
	}
}
