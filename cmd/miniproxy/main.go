package main

import (
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
  }

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	l.Printf("Listening on port %s\n", port)

  handler := proxy.NewHandler(cfg, l)
  if err := http.ListenAndServe(":" + port, handler); err != nil {
    l.Fatal("server: ", err)
  }
}
