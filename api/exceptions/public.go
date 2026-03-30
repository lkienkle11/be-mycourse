package exceptions

import (
	"sort"
	"strings"

	"mycourse-io-be/pkg/setting"
)

// Endpoint describes a route that skips JWT and permission checks (public API).
// Path is the full request path (e.g. /api/v1/health). Method empty means any method.
// If Path ends with "/*", it matches any path with that prefix.
type Endpoint struct {
	Method string
	Path   string
}

// PublicEndpoints returns the merged allowlist under app.public_api_exceptions.
// YAML is applied first; hardcoded entries overwrite the same (method, path) and appear first in the slice
// so Match checks hardcoded rules before YAML-only rules.
func PublicEndpoints() []Endpoint {
	return mergePublicEndpoints(publicEndpointsHardcoded(), setting.AppSetting.PublicAPIExceptions)
}

func publicEndpointsHardcoded() []Endpoint {
	return []Endpoint{
		{Method: "GET", Path: "/api/v1/health"},
	}
}

// mergePublicEndpoints loads YAML into a map, then applies hardcoded on top (hardcoded wins on key clash).
// Output order: hardcoded list order first, then YAML-only keys sorted by key for stability.
func mergePublicEndpoints(hardcoded []Endpoint, fromYAML []setting.PublicAPIExceptionEntry) []Endpoint {
	byKey := make(map[string]Endpoint, len(hardcoded)+len(fromYAML))
	for _, y := range fromYAML {
		e := Endpoint{
			Method: strings.TrimSpace(y.Method),
			Path:   strings.TrimSpace(y.Path),
		}
		if e.Path == "" {
			continue
		}
		k := endpointKey(e)
		byKey[k] = e
	}

	hardOrder := make([]string, 0, len(hardcoded))
	hardSeen := make(map[string]struct{}, len(hardcoded))
	for _, e := range hardcoded {
		if strings.TrimSpace(e.Path) == "" {
			continue
		}
		k := endpointKey(e)
		if _, dup := hardSeen[k]; dup {
			continue
		}
		hardSeen[k] = struct{}{}
		byKey[k] = e // overwrites YAML for the same (method, path)
		hardOrder = append(hardOrder, k)
	}

	out := make([]Endpoint, 0, len(byKey))
	for _, k := range hardOrder {
		out = append(out, byKey[k])
	}

	var yamlOnly []string
	for k := range byKey {
		if _, fromHard := hardSeen[k]; !fromHard {
			yamlOnly = append(yamlOnly, k)
		}
	}
	sort.Strings(yamlOnly)
	for _, k := range yamlOnly {
		out = append(out, byKey[k])
	}
	return out
}

func endpointKey(e Endpoint) string {
	m := strings.ToUpper(strings.TrimSpace(e.Method))
	p := normalizePath(e.Path)
	return m + "\x00" + p
}

// Match reports whether method/path is covered by the allowlist.
func Match(method, reqPath string, rules []Endpoint) bool {
	reqPath = normalizePath(reqPath)
	for _, r := range rules {
		if r.Path == "" {
			continue
		}
		if r.Method != "" && !strings.EqualFold(strings.TrimSpace(r.Method), method) {
			continue
		}
		pat := normalizePath(r.Path)
		if strings.HasSuffix(pat, "/*") {
			prefix := strings.TrimSuffix(pat, "/*")
			if reqPath == prefix || strings.HasPrefix(reqPath, prefix+"/") {
				return true
			}
			continue
		}
		if reqPath == pat {
			return true
		}
	}
	return false
}

func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return "/"
	}
	return p
}
