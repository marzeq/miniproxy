package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/marzeq/miniproxy/config"
)

func TestDoesProxySourceMatchHost(t *testing.T) {
	tests := []struct {
		name   string
		ps     config.ProxySource
		host   string
		expect bool
	}{
		{"exact match", config.ProxySource{Host: "example.com"}, "example.com", true},
		{"mismatch", config.ProxySource{Host: "example.com"}, "other.com", false},
		{"wildcard subdomain", config.ProxySource{Host: "*.example.com"}, "foo.example.com", true},
		{"wildcard base domain", config.ProxySource{Host: "*.example.com"}, "example.com", false},
		{"wildcard fail", config.ProxySource{Host: "*.example.com"}, "evil.com", false},
		{"port match", config.ProxySource{Host: "example.com", Port: 8080}, "example.com:8080", true},
		{"port mismatch", config.ProxySource{Host: "example.com", Port: 8080}, "example.com:9090", false},
		{"port default 80", config.ProxySource{Host: "example.com", Port: 80}, "example.com", true},
		{"any host", config.ProxySource{Host: "*"}, "whatever.com", true},
		{"suffix wildcard", config.ProxySource{Host: "example.*"}, "example.com", true},
		{"suffix wildcard fail", config.ProxySource{Host: "example.*"}, "example", false},
		{"middle wildcard", config.ProxySource{Host: "api.*.example.com"}, "api.v1.example.com", true},
		{"mixed wildcard", config.ProxySource{Host: "*.example.*"}, "foo.example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProxySourceMatchesHost(&tt.ps, tt.host)
			if got != tt.expect {
				t.Errorf("DoesProxySourceMatchHost(%v, %q) = %v; want %v",
					tt.ps, tt.host, got, tt.expect)
			}
		})
	}
}

func TestProxySourceMatchesRequest(t *testing.T) {
	tests := []struct {
		name   string
		ps     config.ProxySource
		url    string
		host   string
		expect bool
	}{
		{
			name: "path and query match",
			ps: config.ProxySource{
				Host:  "example.com",
				Path:  "/api/*",
				Query: url.Values{"debug": {"1"}, "tag": {"v*"}},
			},
			url:    "http://example.com/api/users?tag=v2&debug=1&extra=yes",
			host:   "example.com",
			expect: true,
		},
		{
			name: "query order does not matter",
			ps: config.ProxySource{
				Host:  "example.com",
				Query: url.Values{"a": {"1"}, "b": {"2"}},
			},
			url:    "http://example.com/?b=2&a=1",
			host:   "example.com",
			expect: true,
		},
		{
			name: "path mismatch",
			ps: config.ProxySource{
				Host: "example.com",
				Path: "/admin/*",
			},
			url:    "http://example.com/api/users",
			host:   "example.com",
			expect: false,
		},
		{
			name: "missing query param",
			ps: config.ProxySource{
				Host:  "example.com",
				Query: url.Values{"debug": {"1"}},
			},
			url:    "http://example.com/?mode=prod",
			host:   "example.com",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Host = tt.host

			got := ProxySourceMatchesRequest(&tt.ps, req)
			if got != tt.expect {
				t.Fatalf("ProxySourceMatchesRequest(%v, %q) = %v; want %v", tt.ps, tt.url, got, tt.expect)
			}
		})
	}
}
