package utils

import (
	"net/url"
	"strings"
)

func parseHTTPURL(raw string) (*url.URL, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, false
	}
	if strings.TrimSpace(u.Hostname()) == "" {
		return nil, false
	}
	return u, true
}

// IsOptionalHTTPURL returns true when raw is empty or a valid http(s) URL.
func IsOptionalHTTPURL(raw string) bool {
	_, ok := parseHTTPURL(raw)
	return ok
}

func hostMatchesDomain(hostname, domain string) bool {
	host := strings.ToLower(strings.TrimSpace(hostname))
	host = strings.TrimPrefix(host, "www.")
	domain = strings.ToLower(domain)
	return host == domain || strings.HasSuffix(host, "."+domain)
}

// IsOptionalLinkedInURL returns true when raw is empty or a valid LinkedIn http(s) URL.
func IsOptionalLinkedInURL(raw string) bool {
	u, ok := parseHTTPURL(raw)
	if !ok || u == nil {
		return ok
	}
	return hostMatchesDomain(u.Hostname(), "linkedin.com")
}

// IsOptionalGitHubURL returns true when raw is empty or a valid GitHub http(s) URL.
func IsOptionalGitHubURL(raw string) bool {
	u, ok := parseHTTPURL(raw)
	if !ok || u == nil {
		return ok
	}
	return hostMatchesDomain(u.Hostname(), "github.com")
}
