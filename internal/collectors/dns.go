package collectors

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// defaultDNSServers is used when a query specifies no servers of its own.
var defaultDNSServers = []string{"8.8.8.8:53", "1.1.1.1:53"}

// DNSQuery is a single domain to resolve together with the servers to use.
// An empty Servers list falls back to defaultDNSServers.
type DNSQuery struct {
	Domain  string
	Servers []string
}

type DNSCollector struct {
	queries []DNSQuery
}

func NewDNSCollector(queries []DNSQuery) *DNSCollector {
	return &DNSCollector{queries: queries}
}

func (c *DNSCollector) Name() string            { return "dns" }
func (c *DNSCollector) Interval() time.Duration { return 30 * time.Second }

func (c *DNSCollector) Collect(ctx context.Context) (interface{}, error) {
	results := make([]DNSResult, 0, len(c.queries))
	for _, q := range c.queries {
		results = append(results, resolve(ctx, q.Domain, serversOrDefault(q.Servers)))
	}
	return results, nil
}

// serversOrDefault returns servers unchanged, or the package defaults when empty.
func serversOrDefault(servers []string) []string {
	if len(servers) == 0 {
		return defaultDNSServers
	}
	return servers
}

func resolve(ctx context.Context, domain string, servers []string) DNSResult {
	result := DNSResult{Domain: domain}

	// usedServer records which configured server actually answered; the Go
	// resolver may dial A and AAAA concurrently, so guard it with a mutex.
	var mu sync.Mutex
	usedServer := servers[0]

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: 5 * time.Second}
			// Honour the network the resolver asks for ("udp", then "tcp" when a
			// response is truncated) and fall back across all configured servers.
			var lastErr error
			for _, server := range servers {
				if !strings.Contains(server, ":") {
					server += ":53"
				}
				conn, err := dialer.DialContext(ctx, network, server)
				if err == nil {
					mu.Lock()
					usedServer = server
					mu.Unlock()
					return conn, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
	}

	start := time.Now()

	ips, err := resolver.LookupIPAddr(ctx, domain)
	result.Elapsed = time.Since(start)

	// All Lookup* calls below block until their dials finish, so reading
	// usedServer here is safe without holding the lock.
	result.Server = usedServer

	if err != nil {
		result.Error = err.Error()
		return result
	}

	for _, ip := range ips {
		if ip.IP.To4() != nil {
			result.ARecords = append(result.ARecords, ip.IP.String())
		} else {
			result.AAAARecords = append(result.AAAARecords, ip.IP.String())
		}
	}

	mxs, _ := resolver.LookupMX(ctx, domain)
	for _, mx := range mxs {
		result.MXRecords = append(result.MXRecords, mx.Host)
	}

	nss, _ := resolver.LookupNS(ctx, domain)
	for _, ns := range nss {
		result.NSRecords = append(result.NSRecords, ns.Host)
	}

	cname, _ := resolver.LookupCNAME(ctx, domain)
	if cname != domain+"." {
		result.CNAMERecord = cname
	}

	// Later lookups may have switched servers; report the most recent one.
	mu.Lock()
	result.Server = usedServer
	mu.Unlock()

	return result
}

// ResolveOnce resolves a single domain on-demand (for interactive DNS view).
func ResolveOnce(ctx context.Context, domain string, servers []string) DNSResult {
	return resolve(ctx, domain, serversOrDefault(servers))
}
