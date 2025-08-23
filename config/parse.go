package config

import (
  "fmt"
  "strings"

  "github.com/marzeq/mconf"
  "github.com/marzeq/mconf/mconf_values"
)

func Parse(path string) (map[*ProxySource]*ProxyDest, error) {
  config := make(map[*ProxySource]*ProxyDest)

  confFile, _, err := mconf.ParseFromFile(path)
  if err != nil {
    return nil, err
  }

  for source, destVal := range confFile {
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


		var proxyDest *ProxyDest
		picked := []string{}
		if _, ok := destObj.Value["dest"]; ok {
			destStr, ok := destObj.Value["dest"].(*mconf_values.MconfString)
			if !ok {
				return nil, fmt.Errorf("dest must be string %s", source)
			}
			if len(destStr.Value) == 0 {
				return nil, fmt.Errorf("empty dest in source %s", source)
			}
			proxyDest = &ProxyDest{
				Host: destStr.Value,
			}
			picked = append(picked, "dest")
		}
		if _, ok := destObj.Value["serve_from"]; ok {
			serveFromStr, ok := destObj.Value["serve_from"].(*mconf_values.MconfString)
			if !ok {
				return nil, fmt.Errorf("serve_from must be string %s", source)
			}
			if len(serveFromStr.Value) == 0 {
				return nil, fmt.Errorf("empty serve_from in source %s", source)
			}
			proxyDest = &ProxyDest{
				ServeFrom: serveFromStr.Value,
			}
			picked = append(picked, "serve_from")
		}

		if len(picked) > 1 {
			return nil, fmt.Errorf("%s - only one of dest or serve_from can be set (has: %s)", source, strings.Join(picked, ", "))
		} else if len(picked) == 0 {
			return nil, fmt.Errorf("dest or serve_from must be set in source %s", source)
		}

    sourcePort := 0
    sourceParts := strings.Split(source, ":")
    if len(sourceParts) == 2 {
      _, err := fmt.Sscanf(sourceParts[1], "%d", &sourcePort)
      if err != nil {
        return nil, fmt.Errorf("invalid port in source %s", source)
      }
    } else if len(sourceParts) > 2 {
      return nil, fmt.Errorf("invalid source format %s", source)
    }

    host := sourceParts[0]
    if len(host) == 0 {
      return nil, fmt.Errorf("empty host in source %s", source)
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
      HostPath: host,
      Port:     sourcePort,
			Tls:      tls,
    }

    config[proxySource] = proxyDest
  }

  return config, nil
}

