package config

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/marzeq/mconf/v2"
	"github.com/marzeq/mconf/v2/mconf_values"
)

func Parse(path string) (*Config, error) {
	config := &Config{
		Mappings:   []Pair[*ProxySource, *ProxyDest]{},
		Special404: nil,
		Special504: nil,
	}

	confFile, _, err := mconf.ParseFromFile(path)
	if err != nil {
		return nil, err
	}

	for _, source := range confFile.KeysOrder {
		destVal := confFile.Value[source]
		destObj, ok := destVal.(*mconf_values.MconfObject)
		if !ok {
			destStr, ok := destVal.(*mconf_values.MconfString)
			if !ok {
				return nil, fmt.Errorf("must be object or string %s", source)
			}
			destObj = &mconf_values.MconfObject{
				Value: map[string]mconf_values.MconfValue{},
			}
			destObj.Value["dest"] = destStr
		}

		proxyDest := &ProxyDest{}
		picked := []string{}
		if _, hasDest := destObj.Value["dest"]; hasDest {
			if _, hasHost := destObj.Value["host"]; hasHost {
				return nil, fmt.Errorf("only one of dest or host can be set %s", source)
			}
		}
		if destValue, ok := destObj.Value["dest"]; ok {
			destStr, ok := destValue.(*mconf_values.MconfString)
			if !ok {
				return nil, fmt.Errorf("dest must be string %s", source)
			}
			if len(destStr.Value) == 0 {
				return nil, fmt.Errorf("empty dest in source %s", source)
			}
			proxyDest.Host = destStr.Value
			picked = append(picked, "dest")
		}
		if hostValue, ok := destObj.Value["host"]; ok {
			hostStr, ok := hostValue.(*mconf_values.MconfString)
			if !ok {
				return nil, fmt.Errorf("host must be string %s", source)
			}
			if len(hostStr.Value) == 0 {
				return nil, fmt.Errorf("empty host in source %s", source)
			}
			proxyDest.Host = hostStr.Value
			picked = append(picked, "host")
		}
		if _, ok := destObj.Value["serve_from"]; ok {
			serveFromStr, ok := destObj.Value["serve_from"].(*mconf_values.MconfString)
			if !ok {
				return nil, fmt.Errorf("serve_from must be string %s", source)
			}
			if len(serveFromStr.Value) == 0 {
				return nil, fmt.Errorf("empty serve_from in source %s", source)
			}
			proxyDest.ServeFrom = serveFromStr.Value
			picked = append(picked, "serve_from")
		}
		if _, ok := destObj.Value["cgi"]; ok {
			cgiConfig, err := parseCGIConfig(destObj.Value["cgi"], source)
			if err != nil {
				return nil, err
			}
			proxyDest.CGI = cgiConfig
			picked = append(picked, "cgi")
		}
		if _, ok := destObj.Value["shell_after"]; ok {
			shellAfter, err := parseStringList(destObj.Value["shell_after"], "shell_after", source)
			if err != nil {
				return nil, err
			}
			proxyDest.ShellAfter = shellAfter
		}
		if _, ok := destObj.Value["shell_before"]; ok {
			shellBefore, err := parseStringList(destObj.Value["shell_before"], "shell_before", source)
			if err != nil {
				return nil, err
			}
			proxyDest.ShellBefore = shellBefore
		}

		if len(picked) > 1 {
			return nil, fmt.Errorf("%s - only one of dest, host, serve_from or cgi can be set (has: %s)", source, strings.Join(picked, ", "))
		} else if len(picked) == 0 && proxyDest.ShellBefore == nil {
			return nil, fmt.Errorf("dest, host, serve_from, cgi or shell_before must be set in source %s", source)
		}

		host, path, query, sourcePort, err := parseProxySource(source)
		if err != nil {
			return nil, err
		}

		tls := &TlsConfig{}
		if _, ok := destObj.Value["tls"]; ok {
			tlsObj, ok := destObj.Value["tls"].(*mconf_values.MconfObject)
			if !ok {
				return nil, fmt.Errorf("tls must be object %s", source)
			}

			if _, ok := tlsObj.Value["cert"]; !ok {
				return nil, fmt.Errorf("tls.cert must be set if tls is set %s", source)
			}
			certStr, ok := tlsObj.Value["cert"].(*mconf_values.MconfString)
			if !ok {
				return nil, fmt.Errorf("tls.cert must be string %s", source)
			}
			if len(certStr.Value) == 0 {
				return nil, fmt.Errorf("empty tls.cert in source %s", source)
			}
			tls.Cert = certStr.Value

			if _, ok := tlsObj.Value["key"]; !ok {
				return nil, fmt.Errorf("tls.key must be set if tls is set %s", source)
			}
			keyStr, ok := tlsObj.Value["key"].(*mconf_values.MconfString)
			if !ok {
				return nil, fmt.Errorf("tls.key must be string %s", source)
			}
			if len(keyStr.Value) == 0 {
				return nil, fmt.Errorf("empty tls.key in source %s", source)
			}
			tls.Key = keyStr.Value
		} else {
			tls = nil
		}

		proxySource := &ProxySource{
			Host:  host,
			Path:  path,
			Query: query,
			Port:  sourcePort,
			Tls:   tls,
		}

		switch host {
		case "404":
			config.Special404 = proxyDest
		case "504":
			config.Special504 = proxyDest
		default:
			config.Mappings = append(config.Mappings, Pair[*ProxySource, *ProxyDest]{First: proxySource, Second: proxyDest})
		}
	}

	return config, nil
}

func parseProxySource(source string) (string, string, url.Values, int, error) {
	hostPort := source
	pathAndQuery := ""
	if idx := strings.IndexAny(source, "/?"); idx >= 0 {
		hostPort = source[:idx]
		pathAndQuery = source[idx:]
	}

	host := hostPort
	port := 0
	if hostPart, portPart, ok := strings.Cut(hostPort, ":"); ok {
		if strings.Contains(portPart, ":") {
			return "", "", nil, 0, fmt.Errorf("invalid source format %s", source)
		}
		host = hostPart
		if _, err := fmt.Sscanf(portPart, "%d", &port); err != nil {
			return "", "", nil, 0, fmt.Errorf("invalid port in source %s", source)
		}
	}

	if host == "" {
		return "", "", nil, 0, fmt.Errorf("empty host in source %s", source)
	}

	path := ""
	query := url.Values{}
	if pathAndQuery != "" {
		path = pathAndQuery
		if idx := strings.Index(pathAndQuery, "?"); idx >= 0 {
			path = pathAndQuery[:idx]
			queryString := pathAndQuery[idx+1:]
			parsedQuery, err := url.ParseQuery(queryString)
			if err != nil {
				return "", "", nil, 0, fmt.Errorf("invalid query params in source %s: %w", source, err)
			}
			query = parsedQuery
		}
	}

	return host, path, query, port, nil
}

func parseStringList(value mconf_values.MconfValue, field string, source string) ([]string, error) {
	list, ok := value.(*mconf_values.MconfList)
	if !ok {
		return nil, fmt.Errorf("%s must be array %s", field, source)
	}

	values := make([]string, 0, len(list.Value))
	for _, v := range list.Value {
		valueStr, ok := v.(*mconf_values.MconfString)
		if !ok {
			return nil, fmt.Errorf("%s must be array of strings %s", field, source)
		}
		values = append(values, valueStr.Value)
	}

	return values, nil
}

func parseCGIConfig(value mconf_values.MconfValue, source string) (*CgiConfig, error) {
	switch cgiValue := value.(type) {
	case *mconf_values.MconfString:
		if cgiValue.Value == "" {
			return nil, fmt.Errorf("empty cgi in source %s", source)
		}
		return &CgiConfig{Path: cgiValue.Value}, nil
	case *mconf_values.MconfObject:
		pathValue, ok := cgiValue.Value["path"]
		if !ok {
			return nil, fmt.Errorf("cgi.path must be set %s", source)
		}
		pathStr, ok := pathValue.(*mconf_values.MconfString)
		if !ok {
			return nil, fmt.Errorf("cgi.path must be string %s", source)
		}
		if pathStr.Value == "" {
			return nil, fmt.Errorf("empty cgi.path in source %s", source)
		}

		cfg := &CgiConfig{Path: pathStr.Value}
		if dirValue, ok := cgiValue.Value["dir"]; ok {
			dirStr, ok := dirValue.(*mconf_values.MconfString)
			if !ok {
				return nil, fmt.Errorf("cgi.dir must be string %s", source)
			}
			cfg.Dir = dirStr.Value
		}
		if argsValue, ok := cgiValue.Value["args"]; ok {
			args, err := parseStringList(argsValue, "cgi.args", source)
			if err != nil {
				return nil, err
			}
			cfg.Args = args
		}
		if inheritEnvValue, ok := cgiValue.Value["inherit_env"]; ok {
			inheritEnv, err := parseStringList(inheritEnvValue, "cgi.inherit_env", source)
			if err != nil {
				return nil, err
			}
			cfg.InheritEnv = inheritEnv
		}
		if envValue, ok := cgiValue.Value["env"]; ok {
			envObj, ok := envValue.(*mconf_values.MconfObject)
			if !ok {
				return nil, fmt.Errorf("cgi.env must be object %s", source)
			}

			keys := make([]string, 0, len(envObj.Value))
			for key := range envObj.Value {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			cfg.Env = make([]string, 0, len(keys))
			for _, key := range keys {
				valueStr, ok := envObj.Value[key].(*mconf_values.MconfString)
				if !ok {
					return nil, fmt.Errorf("cgi.env values must be strings %s", source)
				}
				cfg.Env = append(cfg.Env, key+"="+valueStr.Value)
			}
		}

		return cfg, nil
	default:
		return nil, fmt.Errorf("cgi must be string or object %s", source)
	}
}
