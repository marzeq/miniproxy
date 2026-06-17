package config

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type Pair[T any, V any] struct {
	First  T
	Second V
}

func (p Pair[T, V]) String() string {
	return fmt.Sprintf("(%v,%v)", p.First, p.Second)
}

type Config struct {
	Mappings   []Pair[*ProxySource, *ProxyDest]
	Special404 *ProxyDest
	Special504 *ProxyDest
}

type TlsConfig struct {
	Cert string
	Key  string
}

func (t TlsConfig) String() string {
	return fmt.Sprintf("cert=%s,key=%s", t.Cert, t.Key)
}

type ProxySource struct {
	Host  string
	Path  string
	Query url.Values
	Port  int // 0 = any port
	Tls   *TlsConfig
}

func (p ProxySource) String() string {
	var source strings.Builder
	source.WriteString(p.Host)
	if p.Port == 0 {
		source.WriteString(p.Path)
		source.WriteString(formatQueryValues(p.Query))
		return source.String()
	}
	source.WriteString(fmt.Sprintf(":%d", p.Port))
	source.WriteString(p.Path)
	source.WriteString(formatQueryValues(p.Query))
	return source.String()
}

type ProxyDest struct {
	Host        string
	ServeFrom   string // serve static files from this path
	CGI         *CgiConfig
	ShellAfter  []string
	ShellBefore []string
}

type CgiConfig struct {
	Path       string
	Dir        string
	Args       []string
	Env        []string
	InheritEnv []string
}

func (p ProxyDest) String() string {
	switch {
	case p.Host != "":
		return p.Host
	case p.ServeFrom != "":
		return "static content from " + p.ServeFrom
	case p.CGI != nil:
		return "cgi " + p.CGI.Path
	default:
		return ""
	}
}

func formatQueryValues(values url.Values) string {
	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		for _, value := range values[key] {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}

	return "?" + strings.Join(parts, "&")
}
