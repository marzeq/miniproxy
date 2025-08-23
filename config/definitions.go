package config

import "fmt"

type TlsConfig struct {
	Cert string
	Key  string
}

func (t TlsConfig) String() string {
	return fmt.Sprintf("cert=%s,key=%s", t.Cert, t.Key)
}

type ProxySource struct {
  HostPath string
  Port     int // 0 = any port
	Tls      *TlsConfig
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
