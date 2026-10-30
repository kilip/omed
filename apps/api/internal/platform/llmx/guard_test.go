package llmx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

type mockResolver struct {
	hosts map[string][]netip.Addr
}

func (m *mockResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	if addrs, ok := m.hosts[host]; ok {
		return addrs, nil
	}
	return net.DefaultResolver.LookupNetIP(ctx, network, host)
}

func TestValidateBaseURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		allowPrivate bool
		wantErr      bool
	}{
		{
			name:         "valid public https",
			url:          "https://api.openai.com/v1",
			allowPrivate: false,
			wantErr:      false,
		},
		{
			name:         "blocked scheme http when private disallowed",
			url:          "http://api.example.com",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "blocked userinfo",
			url:          "https://user:pw@host/",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "blocked invalid scheme ftp",
			url:          "ftp://x",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "loopback http allowed when allowPrivate true",
			url:          "http://127.0.0.1:11434",
			allowPrivate: true,
			wantErr:      false,
		},
		{
			name:         "loopback http blocked when allowPrivate false",
			url:          "http://127.0.0.1:11434",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "decimal ip blocked",
			url:          "http://2130706433/",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "mapped ipv4 blocked",
			url:          "http://[::ffff:127.0.0.1]/",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "localhost with trailing dot blocked",
			url:          "http://localhost./",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "loopback https blocked when allowPrivate false",
			url:          "https://127.0.0.1:8080",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "localhost https blocked when allowPrivate false",
			url:          "https://localhost/",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "decimal ip https blocked when allowPrivate false",
			url:          "https://2130706433/",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "mapped ipv4 https blocked when allowPrivate false",
			url:          "https://[::ffff:127.0.0.1]/",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "empty url blocked",
			url:          "",
			allowPrivate: false,
			wantErr:      true,
		},
		{
			name:         "subdomain of localhost blocked",
			url:          "https://foo.localhost/",
			allowPrivate: false,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBaseURL(tc.url, tc.allowPrivate)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.url)
				}
				if !errors.Is(err, ErrBlockedURL) {
					t.Fatalf("expected error wrapping ErrBlockedURL for %q, got %v", tc.url, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for %q: %v", tc.url, err)
				}
			}
		})
	}
}

func TestGuardBlocksDialToLoopback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// allowPrivate = false must fail with ErrBlockedURL
	clientBlocked := newHTTPClient(false)
	_, err := clientBlocked.Get(ts.URL)
	if err == nil {
		t.Fatal("expected dial to loopback to fail when allowPrivate=false, got nil error")
	}
	if !errors.Is(err, ErrBlockedURL) {
		t.Fatalf("expected error wrapping ErrBlockedURL, got %v", err)
	}

	// allowPrivate = true must succeed
	clientAllowed := newHTTPClient(true)
	resp, err := clientAllowed.Get(ts.URL)
	if err != nil {
		t.Fatalf("expected dial to loopback to succeed when allowPrivate=true, got %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestGuardBlocksHostnameResolvingToPrivate(t *testing.T) {
	oldResolver := defaultResolver
	defer func() { defaultResolver = oldResolver }()

	defaultResolver = &mockResolver{
		hosts: map[string][]netip.Addr{
			"test.private.local": {netip.MustParseAddr("10.0.0.5")},
		},
	}

	client := newHTTPClient(false)
	_, err := client.Get("http://test.private.local:8080/v1")
	if err == nil {
		t.Fatal("expected request to hostname resolving to private IP to fail, got nil error")
	}
	if !errors.Is(err, ErrBlockedURL) {
		t.Fatalf("expected error wrapping ErrBlockedURL, got %v", err)
	}
}

func TestRedirectsAreNotFollowed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := newHTTPClient(true)
	resp, err := client.Get(ts.URL + "/redirect")
	if err != nil {
		t.Fatalf("expected GET to succeed without following redirect, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected status 302, got %d", resp.StatusCode)
	}

	err = MapHTTPError(resp.StatusCode, nil)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected status 302 to map to ErrUnavailable, got %v", err)
	}
}

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		blocked bool
	}{
		{"ipv4 loopback", "127.0.0.1", true},
		{"ipv6 loopback", "::1", true},
		{"ipv4 private 10/8", "10.0.0.1", true},
		{"ipv4 private 172.16/12", "172.16.0.1", true},
		{"ipv4 private 192.168/16", "192.168.1.1", true},
		{"ipv6 unique local fc00::/7", "fc00::1", true},
		{"ipv6 unique local fd00::/8", "fd12:3456:789a::1", true},
		{"ipv4 link-local 169.254/16", "169.254.1.1", true},
		{"ipv6 link-local fe80::/10", "fe80::1", true},
		{"ipv4 unspecified", "0.0.0.0", true},
		{"ipv6 unspecified", "::", true},
		{"ipv4 multicast", "224.0.0.1", true},
		{"ipv6 multicast", "ff02::1", true},
		{"mapped ipv4 loopback", "::ffff:127.0.0.1", true},
		{"mapped ipv4 private", "::ffff:10.0.0.1", true},
		{"mapped ipv4 link-local", "::ffff:169.254.1.1", true},
		{"public ipv4 cloudflare", "1.1.1.1", false},
		{"public ipv4 google", "8.8.8.8", false},
		{"public ipv6 cloudflare", "2606:4700:4700::1111", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			addr := netip.MustParseAddr(tc.ip)
			if got := isBlockedIP(addr); got != tc.blocked {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tc.ip, got, tc.blocked)
			}
		})
	}

	if !isBlockedIP(netip.Addr{}) {
		t.Errorf("isBlockedIP(invalid) = false, want true")
	}
}
