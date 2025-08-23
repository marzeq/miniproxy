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

  for _, mp := range cfg.Mappings {
    src := mp.First
    dst := mp.Second

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

    l.Printf("Proxying %s to %s\n", src.HostPath, dests)

    if src.Tls != nil {
      certs[src.HostPath] = *src.Tls
      l.Printf(" -- with TLS %s\n", src.Tls)
    }
  }
  if cfg.Special404 != nil {
    l.Printf("Special 404 page to %s\n", cfg.Special404)
  }
  if cfg.Special504 != nil {
    l.Printf("Special 504 page to %s\n", cfg.Special504)
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

        for _, mp := range cfg.Mappings {
          src := mp.First
          if proxy.ProxySourceMatchesHost(src, host) && src.Tls != nil {
            cert, err := tls.LoadX509KeyPair(src.Tls.Cert, src.Tls.Key)
            if err != nil {
              l.Printf("Error loading cert for %s: %v", host, err)
              return nil, err
            }
            return &cert, nil
          }
        }

        if tlsConf, ok := certs["404"]; ok {
          cert, err := tls.LoadX509KeyPair(tlsConf.Cert, tlsConf.Key)
          if err != nil {
            l.Printf("Error loading cert for 404: %v", err)
            return nil, err
          }
          return &cert, nil
        }
        l.Printf("No certificate found for host %s", host)
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

