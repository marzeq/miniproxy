package proxy

import (
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
    {"exact match", config.ProxySource{HostPath: "example.com"}, "example.com", true},
    {"mismatch", config.ProxySource{HostPath: "example.com"}, "other.com", false},
    {"wildcard subdomain", config.ProxySource{HostPath: "*.example.com"}, "foo.example.com", true},
    {"wildcard base domain", config.ProxySource{HostPath: "*.example.com"}, "example.com", false},
    {"wildcard fail", config.ProxySource{HostPath: "*.example.com"}, "evil.com", false},
    {"port match", config.ProxySource{HostPath: "example.com", Port: 8080}, "example.com:8080", true},
    {"port mismatch", config.ProxySource{HostPath: "example.com", Port: 8080}, "example.com:9090", false},
    {"port default 80", config.ProxySource{HostPath: "example.com", Port: 80}, "example.com", true},
    {"any host", config.ProxySource{HostPath: "*"}, "whatever.com", true},
    {"suffix wildcard", config.ProxySource{HostPath: "example.*"}, "example.com", true},
    {"suffix wildcard fail", config.ProxySource{HostPath: "example.*"}, "example", false},
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

