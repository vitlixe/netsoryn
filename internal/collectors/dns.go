package collectors

import (
	"context"
	"net"
	"strings"
	"time"
)

type DNSCollector struct {
	domains []string
	servers []string
}

func NewDNSCollector(domains, servers []string) *DNSCollector {
	if len(servers) == 0 {
		servers = []string{"8.8.8.8:53", "1.1.1.1:53"}
	}
	return &DNSCollector{domains: domains, servers: servers}
}

func (c *DNSCollector) Name() string            { return "dns" }
func (c *DNSCollector) Interval() time.Duration { return 30 * time.Second }

func (c *DNSCollector) Collect(ctx context.Context) (interface{}, error) {
	results := make([]DNSResult, 0, len(c.domains))
	for _, domain := range c.domains {
		r := c.resolve(ctx, domain)
		results = append(results, r)
	}
	return results, nil
}

func (c *DNSCollector) resolve(ctx context.Context, domain string) DNSResult {
	result := DNSResult{Domain: domain}

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			server := c.servers[0]
			if !strings.Contains(server, ":") {
				server += ":53"
			}
			return d.DialContext(ctx, "udp", server)
		},
	}
	result.Server = c.servers[0]

	start := time.Now()

	ips, err := resolver.LookupIPAddr(ctx, domain)
	result.Elapsed = time.Since(start)

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

	return result
}

// ResolveOnce resolves a single domain on-demand (for interactive DNS view).
func ResolveOnce(ctx context.Context, domain string, servers []string) DNSResult {
	c := NewDNSCollector([]string{domain}, servers)
	return c.resolve(ctx, domain)
}
