package llmx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	linkLocalIPv4Prefix   = netip.MustParsePrefix("169.254.0.0/16")
	uniqueLocalIPv6Prefix = netip.MustParsePrefix("fc00::/7")
)

type netResolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

var defaultResolver netResolver = net.DefaultResolver

func isBlockedIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() {
		return true
	}
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() {
		return true
	}
	if linkLocalIPv4Prefix.Contains(ip) || uniqueLocalIPv6Prefix.Contains(ip) {
		return true
	}
	return false
}

// ValidateBaseURL validates a base URL against SSRF rules.
// If allowPrivate is false, the URL scheme must be https and the host cannot resolve
// to or specify private/loopback/link-local/multicast IP addresses or localhost.
func ValidateBaseURL(raw string, allowPrivate bool) error {
	if raw == "" {
		return fmt.Errorf("%w: empty url", ErrBlockedURL)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: invalid url: %w", ErrBlockedURL, err)
	}

	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%w: missing scheme or host", ErrBlockedURL)
	}

	if u.User != nil {
		return fmt.Errorf("%w: userinfo not allowed", ErrBlockedURL)
	}

	scheme := strings.ToLower(u.Scheme)
	if allowPrivate {
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("%w: scheme must be http or https", ErrBlockedURL)
		}
	} else {
		if scheme != "https" {
			return fmt.Errorf("%w: scheme must be https", ErrBlockedURL)
		}
	}

	if !allowPrivate {
		host := u.Hostname()
		if host == "" {
			return fmt.Errorf("%w: empty host", ErrBlockedURL)
		}

		h := strings.ToLower(host)
		if h == "localhost" || h == "localhost." || strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".localhost.") {
			return fmt.Errorf("%w: localhost host not allowed", ErrBlockedURL)
		}

		if val, err := strconv.ParseUint(host, 10, 32); err == nil {
			b := [4]byte{byte(val >> 24), byte(val >> 16), byte(val >> 8), byte(val)}
			addr := netip.AddrFrom4(b)
			if isBlockedIP(addr) {
				return fmt.Errorf("%w: blocked ip: %s", ErrBlockedURL, addr)
			}
		}

		cleanHost := strings.Trim(host, "[]")
		if addr, err := netip.ParseAddr(cleanHost); err == nil {
			if isBlockedIP(addr) {
				return fmt.Errorf("%w: blocked ip: %s", ErrBlockedURL, addr)
			}
		}
	}

	return nil
}

func newHTTPClient(allowPrivate bool) *http.Client {
	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid address %s", ErrBlockedURL, addr)
			}

			if !allowPrivate {
				ips, err := defaultResolver.LookupNetIP(ctx, "ip", host)
				if err != nil {
					return nil, err
				}
				for _, ip := range ips {
					if isBlockedIP(ip) {
						return nil, fmt.Errorf("%w: resolved to blocked ip %s", ErrBlockedURL, ip)
					}
				}
			}

			dialer := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			if !allowPrivate {
				dialer.Control = func(network, address string, c syscall.RawConn) error {
					h, _, err := net.SplitHostPort(address)
					if err != nil {
						h = address
					}
					ip, err := netip.ParseAddr(h)
					if err != nil {
						return fmt.Errorf("%w: invalid socket address %s", ErrBlockedURL, address)
					}
					if isBlockedIP(ip) {
						return fmt.Errorf("%w: connection to blocked ip %s", ErrBlockedURL, ip)
					}
					return nil
				}
			}

			return dialer.DialContext(ctx, network, addr)
		},
	}

	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
