package proxy

import (
  "fmt"
  "strings"
)

type ProxySource struct {
  HostPath string
  Port     int // 0 = any port
}

func (p ProxySource) String() string {
  if p.Port == 0 {
    return p.HostPath
  }
  return fmt.Sprintf("%s:%d", p.HostPath, p.Port)
}

type ProxyDest struct {
  Host string
	ServeFrom string // serve static files from this path, mutually exclusive with Host
}

func (p ProxyDest) String() string {
  return p.Host
}

func ProxySourceMatchesHost(ps *ProxySource, host string) bool {
  if ps.Port != 0 {
    port := 80
    hostParts := strings.Split(host, ":")
    if len(hostParts) == 2 {
      _, err := fmt.Sscanf(hostParts[1], "%d", &port)
      if err != nil {
        return false
      }
      host = hostParts[0]
    }
    if port != ps.Port {
      return false
    }
  }

  if ps.HostPath == "*" {
    return true
  }

  if domain, ok := strings.CutPrefix(ps.HostPath, "*."); ok {
    return strings.HasSuffix(host, "." + domain)
  }
	if domain, ok := strings.CutSuffix(ps.HostPath, ".*"); ok {
		return strings.HasPrefix(host, domain + ".")
	}

  return ps.HostPath == host
}
