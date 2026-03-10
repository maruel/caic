// Package ipgeo provides IP geolocation and country-based allowlist enforcement
// using MaxMind MMDB files.
package ipgeo

import (
	"net/http"
	"net/netip"
	"strings"

	"github.com/oschwald/maxminddb-golang/v2"
)

// tailscalePrefix is the Tailscale CGNAT range 100.64.0.0/10.
var tailscalePrefix = netip.MustParsePrefix("100.64.0.0/10")

// GetClientIP extracts the real client IP from a request, checking
// X-Forwarded-For and X-Real-IP headers for proxied requests.
func GetClientIP(r *http.Request) string {
	// X-Forwarded-For may contain "client, proxy1, proxy2" — use the leftmost.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, found := strings.Cut(xff, ",")
		if found {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// RemoteAddr: strip port, handle IPv6 [::1]:port form.
	addr := r.RemoteAddr
	if strings.HasPrefix(addr, "[") {
		if host, _, found := strings.Cut(addr, "]:"); found {
			return host[1:]
		}
		return strings.Trim(addr, "[]")
	}
	if host, _, found := strings.Cut(addr, ":"); found {
		return host
	}
	return addr
}

// Checker resolves IP addresses to ISO 3166-1 alpha-2 country codes using a
// MaxMind MMDB file. A nil *Checker is valid and handles local/Tailscale IPs.
type Checker struct {
	reader *maxminddb.Reader
}

// Open opens an MMDB file for country lookups.
func Open(dbPath string) (*Checker, error) {
	r, err := maxminddb.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &Checker{reader: r}, nil
}

// Close releases MMDB reader resources.
func (c *Checker) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

// countryRecord is the minimal MMDB struct for country lookups.
type countryRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for the given IP
// string. Special return values:
//   - "local" for loopback, private, link-local, and unspecified IPs
//   - "tailscale" for Tailscale CGNAT IPs (100.64.0.0/10)
//   - "" on parse error, lookup error, or if c is nil with a public IP
func (c *Checker) CountryCode(ipStr string) string {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return ""
	}
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsUnspecified() || addr.IsLinkLocalUnicast() {
		return "local"
	}
	if tailscalePrefix.Contains(addr) {
		return "tailscale"
	}
	if c == nil || c.reader == nil {
		return ""
	}
	var rec countryRecord
	if err := c.reader.Lookup(addr).Decode(&rec); err != nil {
		return ""
	}
	return rec.Country.ISOCode
}

// Allowlist checks whether a country code is permitted. A nil *Allowlist
// allows everything.
type Allowlist struct {
	allowed map[string]struct{}
}

// ParseAllowlist parses a comma-separated list of allowed values. Each token
// is uppercased; "LOCAL" and "TAILSCALE" match the special return values from
// CountryCode; any other token is treated as an ISO 3166-1 alpha-2 country
// code (e.g. "CA", "US"). Returns nil if s is empty.
func ParseAllowlist(s string) *Allowlist {
	if s == "" {
		return nil
	}
	a := &Allowlist{allowed: make(map[string]struct{})}
	for token := range strings.SplitSeq(s, ",") {
		token = strings.ToUpper(strings.TrimSpace(token))
		if token != "" {
			a.allowed[token] = struct{}{}
		}
	}
	if len(a.allowed) == 0 {
		return nil
	}
	return a
}

// Allowed reports whether the given country code (as returned by CountryCode)
// is on the allowlist. Returns true if the allowlist is nil.
func (a *Allowlist) Allowed(cc string) bool {
	if a == nil {
		return true
	}
	_, ok := a.allowed[strings.ToUpper(cc)]
	return ok
}

// NeedsDB reports whether the allowlist contains any entry that requires a
// MaxMind MMDB to resolve (i.e. not just "LOCAL" or "TAILSCALE").
func (a *Allowlist) NeedsDB() bool {
	if a == nil {
		return false
	}
	for token := range a.allowed {
		if token != "LOCAL" && token != "TAILSCALE" {
			return true
		}
	}
	return false
}
