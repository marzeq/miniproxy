package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/marzeq/miniproxy/config"
)

func ProxySourceMatchesHost(ps *config.ProxySource, host string) bool {
	if !portMatches(ps, host) {
		return false
	}

	host = stripPort(host)
	return wildcardMatch(ps.Host, host)
}

func ProxySourceMatchesRequest(ps *config.ProxySource, r *http.Request) bool {
	if !ProxySourceMatchesHost(ps, r.Host) {
		return false
	}

	if ps.Path != "" && !wildcardMatch(ps.Path, r.URL.Path) {
		return false
	}

	return queryMatches(ps.Query, r.URL.Query())
}

func portMatches(ps *config.ProxySource, host string) bool {
	if ps.Port != 0 {
		port := 0
		hostParts := strings.Split(host, ":")
		if len(hostParts) == 2 {
			_, err := fmt.Sscanf(hostParts[1], "%d", &port)
			if err != nil {
				return false
			}
			host = hostParts[0]
		} else {
			if ps.Tls != nil {
				port = 443
			} else {
				port = 80
			}
		}
		if port != ps.Port {
			return false
		}
	}

	return true
}

func stripPort(host string) string {
	hostParts := strings.Split(host, ":")
	if len(hostParts) == 2 {
		return hostParts[0]
	}
	return host
}

func wildcardMatch(pattern string, value string) bool {
	if pattern == "" {
		return value == ""
	}
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 0 {
		return value == pattern
	}

	if !strings.HasPrefix(pattern, "*") {
		if !strings.HasPrefix(value, parts[0]) {
			return false
		}
		value = value[len(parts[0]):]
		parts = parts[1:]
	}

	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == len(parts)-1 && !strings.HasSuffix(pattern, "*") {
			return strings.HasSuffix(value, part)
		}

		idx := strings.Index(value, part)
		if idx < 0 {
			return false
		}
		value = value[idx+len(part):]
	}

	return true
}

func queryMatches(expected url.Values, actual url.Values) bool {
	if len(expected) == 0 {
		return true
	}

	for key, expectedValues := range expected {
		actualValues, ok := actual[key]
		if !ok || len(actualValues) < len(expectedValues) {
			return false
		}

		used := make([]bool, len(actualValues))
		for _, expectedValue := range expectedValues {
			matched := false
			for idx, actualValue := range actualValues {
				if used[idx] {
					continue
				}
				if wildcardMatch(expectedValue, actualValue) {
					used[idx] = true
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	return true
}
