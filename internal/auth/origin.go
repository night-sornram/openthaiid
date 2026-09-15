package auth

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/night-sornram/openthaiid/internal/card"
)

// NormalizeOrigin parses and canonicalizes a browser Origin header.
// Rules from docs/security.md:
//   - require a value (missing/null rejected by caller)
//   - parse as absolute URL with scheme http/https
//   - no userinfo
//   - host must be present
//   - no path, query, or fragment
//   - port must be numeric 1–65535 when present
//   - default ports dropped from the canonical form
//   - IPv6 hosts keep bracketed serialization
func NormalizeOrigin(raw string) (Origin, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || raw == "*" {
		return "", card.NewError(card.ErrApprovalDenied, "invalid origin")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", card.NewError(card.ErrApprovalDenied, "invalid origin")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", card.NewError(card.ErrApprovalDenied, "invalid origin scheme")
	}
	if u.User != nil {
		return "", card.NewError(card.ErrApprovalDenied, "invalid origin userinfo")
	}
	if u.Path != "" && u.Path != "/" {
		return "", card.NewError(card.ErrApprovalDenied, "invalid origin path")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", card.NewError(card.ErrApprovalDenied, "invalid origin query or fragment")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", card.NewError(card.ErrApprovalDenied, "invalid origin host")
	}
	if host == "*" || strings.Contains(host, "*") {
		return "", card.NewError(card.ErrApprovalDenied, "wildcard origin forbidden")
	}

	port := u.Port()
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", card.NewError(card.ErrApprovalDenied, "invalid origin port")
		}
		switch {
		case scheme == "http" && port == "80":
			port = ""
		case scheme == "https" && port == "443":
			port = ""
		}
	}

	out := scheme + "://"
	if strings.Contains(host, ":") {
		// IPv6 literal keeps brackets in the serialized origin.
		out += "[" + host + "]"
	} else {
		out += host
	}
	if port != "" {
		out += ":" + port
	}
	return Origin(out), nil
}

// Eligible reports whether an origin may request approval (not yet trusted).
func Eligible(origin Origin, allowHTTPLoopback bool, allowList []Origin) bool {
	u, err := url.Parse(string(origin))
	if err != nil {
		return false
	}
	scheme := u.Scheme
	host := u.Hostname()

	if len(allowList) > 0 {
		for _, a := range allowList {
			if a == origin {
				return allowHTTPLoopbackOrHTTPS(scheme, host, allowHTTPLoopback)
			}
		}
		return false
	}
	return allowHTTPLoopbackOrHTTPS(scheme, host, allowHTTPLoopback)
}

func allowHTTPLoopbackOrHTTPS(scheme, host string, allowHTTPLoopback bool) bool {
	if scheme == "https" {
		return true
	}
	if scheme == "http" && allowHTTPLoopback && isLoopbackHost(host) {
		return true
	}
	return false
}

func isLoopbackHost(host string) bool {
	h := strings.ToLower(host)
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// IsLoopbackPeer reports whether the remote address is a loopback client.
func IsLoopbackPeer(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ValidHostAuthority checks the HTTP Host header against the expected
// literal authority (host:port). Rebinding attempts that present another
// host are rejected.
func ValidHostAuthority(got, want string) bool {
	g := strings.ToLower(strings.TrimSpace(got))
	w := strings.ToLower(strings.TrimSpace(want))
	return g != "" && g == w
}

// FormatOrigin is a helper for tests and terminal display.
func FormatOrigin(o Origin) string { return string(o) }

// MustOrigin is a test helper.
func MustOrigin(raw string) Origin {
	o, err := NormalizeOrigin(raw)
	if err != nil {
		panic(fmt.Sprintf("origin %q: %v", raw, err))
	}
	return o
}
