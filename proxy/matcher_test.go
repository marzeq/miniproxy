package proxy

import "testing"

func TestDoesProxySourceMatchHost(t *testing.T) {
  tests := []struct {
    name   string
    ps     ProxySource
    host   string
    expect bool
  }{
    {"exact match", ProxySource{HostPath: "example.com"}, "example.com", true},
    {"mismatch", ProxySource{HostPath: "example.com"}, "other.com", false},
    {"wildcard subdomain", ProxySource{HostPath: "*.example.com"}, "foo.example.com", true},
    {"wildcard base domain", ProxySource{HostPath: "*.example.com"}, "example.com", false},
    {"wildcard fail", ProxySource{HostPath: "*.example.com"}, "evil.com", false},
    {"port match", ProxySource{HostPath: "example.com", Port: 8080}, "example.com:8080", true},
    {"port mismatch", ProxySource{HostPath: "example.com", Port: 8080}, "example.com:9090", false},
    {"port default 80", ProxySource{HostPath: "example.com", Port: 80}, "example.com", true},
    {"any host", ProxySource{HostPath: "*"}, "whatever.com", true},
    {"suffix wildcard", ProxySource{HostPath: "example.*"}, "example.com", true},
    {"suffix wildcard fail", ProxySource{HostPath: "example.*"}, "example", false},
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

