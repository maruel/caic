package ipgeo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		want          string
	}{
		{name: "remote addr ipv4", remoteAddr: "1.2.3.4:5678", want: "1.2.3.4"},
		{name: "remote addr ipv6", remoteAddr: "[::1]:8080", want: "::1"},
		{name: "x-forwarded-for single", xForwardedFor: "1.2.3.4", remoteAddr: "10.0.0.1:80", want: "1.2.3.4"},
		{name: "x-forwarded-for chain", xForwardedFor: "1.2.3.4, 10.0.0.1", remoteAddr: "10.0.0.2:80", want: "1.2.3.4"},
		{name: "x-real-ip", xRealIP: "5.6.7.8", remoteAddr: "10.0.0.1:80", want: "5.6.7.8"},
		{name: "x-forwarded-for beats x-real-ip", xForwardedFor: "1.2.3.4", xRealIP: "5.6.7.8", remoteAddr: "10.0.0.1:80", want: "1.2.3.4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			r.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				r.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				r.Header.Set("X-Real-IP", tt.xRealIP)
			}
			if got := GetClientIP(r); got != tt.want {
				t.Errorf("GetClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCountryCode(t *testing.T) {
	// A nil-reader Checker handles all special cases; public IPs return "".
	c := &Checker{}
	tests := []struct {
		ip   string
		want string
	}{
		{"127.0.0.1", "local"},
		{"::1", "local"},
		{"10.0.0.1", "local"},
		{"192.168.1.1", "local"},
		{"172.16.0.1", "local"},
		{"0.0.0.0", "local"},
		{"::", "local"},
		{"169.254.1.1", "local"},
		{"fe80::1", "local"},
		{"100.64.0.1", "tailscale"},
		{"100.100.100.100", "tailscale"},
		{"100.127.255.254", "tailscale"},
		{"100.63.255.255", ""}, // just outside Tailscale range
		{"100.128.0.0", ""},    // just outside Tailscale range
		{"8.8.8.8", ""},        // public IP, no MMDB
		{"not-an-ip", ""},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := c.CountryCode(tt.ip); got != tt.want {
				t.Errorf("CountryCode(%q) = %q, want %q", tt.ip, got, tt.want)
			}
		})
	}
	t.Run("nil checker", func(t *testing.T) {
		var nilChecker *Checker
		if got := nilChecker.CountryCode("127.0.0.1"); got != "local" {
			t.Errorf("nil checker CountryCode(loopback) = %q, want %q", got, "local")
		}
		if got := nilChecker.CountryCode("8.8.8.8"); got != "" {
			t.Errorf("nil checker CountryCode(public) = %q, want %q", got, "")
		}
	})
}

func TestParseAllowlist(t *testing.T) {
	t.Run("nil on empty", func(t *testing.T) {
		if ParseAllowlist("") != nil {
			t.Error("expected nil for empty string")
		}
	})
	t.Run("nil on whitespace only", func(t *testing.T) {
		if ParseAllowlist("  ,  ") != nil {
			t.Error("expected nil for whitespace-only")
		}
	})
	t.Run("allows listed", func(t *testing.T) {
		a := ParseAllowlist("CA,US,tailscale")
		if a == nil {
			t.Fatal("expected non-nil allowlist")
		}
		for _, cc := range []string{"CA", "US", "TAILSCALE", "tailscale", "ca"} {
			if !a.Allowed(cc) {
				t.Errorf("Allowed(%q) = false, want true", cc)
			}
		}
	})
	t.Run("blocks unlisted", func(t *testing.T) {
		a := ParseAllowlist("CA")
		for _, cc := range []string{"US", "GB", "local", "tailscale", ""} {
			if a.Allowed(cc) {
				t.Errorf("Allowed(%q) = true, want false", cc)
			}
		}
	})
	t.Run("nil allowlist allows all", func(t *testing.T) {
		var a *Allowlist
		if !a.Allowed("CA") || !a.Allowed("") || !a.Allowed("tailscale") {
			t.Error("nil allowlist should allow everything")
		}
	})
}

func TestNeedsDB(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"local", false},
		{"tailscale", false},
		{"local,tailscale", false},
		{"CA", true},
		{"local,CA", true},
		{"tailscale,US", true},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			a := ParseAllowlist(tt.s)
			if got := a.NeedsDB(); got != tt.want {
				t.Errorf("NeedsDB() = %v, want %v", got, tt.want)
			}
		})
	}
}
