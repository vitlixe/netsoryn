package collectors

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

type HTTPCollector struct {
	urls    []string
	timeout time.Duration
}

func NewHTTPCollector(urls []string, timeout time.Duration) *HTTPCollector {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &HTTPCollector{urls: urls, timeout: timeout}
}

func (c *HTTPCollector) Name() string            { return "http" }
func (c *HTTPCollector) Interval() time.Duration { return 30 * time.Second }

func (c *HTTPCollector) Collect(ctx context.Context) (interface{}, error) {
	results := make([]HTTPResult, 0, len(c.urls))
	for _, u := range c.urls {
		r := c.check(ctx, u)
		results = append(results, r)
	}
	return results, nil
}

func (c *HTTPCollector) check(ctx context.Context, rawURL string) HTTPResult {
	result := HTTPResult{URL: rawURL}

	client := &http.Client{
		Timeout: c.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 {
				result.Redirect = req.URL.String()
			}
			return nil
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Header.Set("User-Agent", "netsoryn/1.0 diagnostic-tool")

	start := time.Now()
	resp, err := client.Do(req)
	result.Elapsed = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.StatusText = resp.Status
	result.ContentType = resp.Header.Get("Content-Type")

	if resp.TLS != nil {
		result.TLSValid = true
		if len(resp.TLS.PeerCertificates) > 0 {
			cert := resp.TLS.PeerCertificates[0]
			result.TLSExpiry = cert.NotAfter.Format("2006-01-02")
			if len(cert.Issuer.Organization) > 0 {
				result.TLSIssuer = cert.Issuer.Organization[0]
			} else {
				result.TLSIssuer = cert.Issuer.CommonName
			}
			if time.Now().After(cert.NotAfter) {
				result.TLSValid = false
			}
		}
	}

	return result
}

// CheckOnce checks a single URL on-demand (for interactive HTTP view).
func CheckOnce(ctx context.Context, url string, timeout time.Duration) HTTPResult {
	c := NewHTTPCollector([]string{url}, timeout)
	return c.check(ctx, url)
}

// StatusColor returns a category string based on HTTP status code.
func StatusColor(code int) string {
	switch {
	case code >= 500:
		return "error"
	case code >= 400:
		return "warn"
	case code >= 300:
		return "redirect"
	case code >= 200:
		return "ok"
	default:
		return "unknown"
	}
}

// FormatElapsed formats a duration as a short string.
func FormatElapsed(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
