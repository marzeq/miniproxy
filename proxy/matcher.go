package proxy

import (
	"fmt"
	"strings"

	"github.com/marzeq/miniproxy/config"
)

func ProxySourceMatchesHost(ps *config.ProxySource, host string) bool {
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
