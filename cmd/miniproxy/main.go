package main

import (
  "crypto/tls"
  "log"
  "net/http"
  "os"

  "github.com/marzeq/miniproxy/config"
  "github.com/marzeq/miniproxy/proxy"
)

func main() {
  l := log.New(os.Stdout, "", log.LstdFlags)

  cfgfile := os.Getenv("CONFIG_FILE")
  if cfgfile == "" {
    cfgfile = "/etc/miniproxy/config.mconf"
  }

  cfg, err := config.Parse(cfgfile)
  if err != nil {
    l.Fatal("parse config: ", err)
  }

  certs := make(map[string]config.TlsConfig)

  for cmd, dst := range cfg {
    dests := ""
    if dst.Host != "" {
      dests += dst.Host
    }
    if dst.ServeFrom != "" {
      if len(dests) > 0 {
        dests += ", "
      }
      dests += "static content from " + dst.ServeFrom
    }

    switch cmd.HostPath {
    case "404":
      l.Printf("Proxying all unmatched hosts to %s (special \"404\" host)\n", dests)
    case "504":
      l.Printf("Proxying all unresponsive hosts to %s (special \"504\" host)\n", dests)
    default:
      l.Printf("Proxying %s to %s\n", cmd.HostPath, dests)
    }

    if cmd.Tls != nil {
      certs[cmd.HostPath] = *cmd.Tls
      l.Printf(" -- with TLS %s\n", cmd.Tls)
    }
  }

  port := os.Getenv("PORT")
  if port == "" {
    port = "80"
  }
  l.Printf("Listening on HTTP port %s\n", port)

  handler := proxy.NewHandler(cfg, l)

  go func() {
    if err := http.ListenAndServe(":"+port, handler); err != nil {
      l.Fatal("HTTP server: ", err)
    }
  }()

  if len(certs) > 0 {
		https_port := os.Getenv("HTTPS_PORT")
		if https_port == "" {
			https_port = "443"
		}
		l.Printf("Listening on HTTPS port %s\n", https_port)


		tlsConfig := &tls.Config{
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				host := clientHello.ServerName

				for ps := range cfg {
					if proxy.ProxySourceMatchesHost(ps, host) && ps.Tls != nil {
						cert, err := tls.LoadX509KeyPair(ps.Tls.Cert, ps.Tls.Key)
						if err != nil {
							l.Printf("Error loading cert for %s: %v", host, err)
							return nil, err
						}
						return &cert, nil
					}
				}

				return nil, nil
			},
		}


    httpsServer := &http.Server{
			Addr:      ":" + https_port,
      Handler:   handler,
      TLSConfig: tlsConfig,
    }

    if err := httpsServer.ListenAndServeTLS("", ""); err != nil {
      l.Fatal("HTTPS server: ", err)
    }
  }

  select {}
}

