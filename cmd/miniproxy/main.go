package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/marzeq/miniproxy/config"
	"github.com/marzeq/miniproxy/proxy"
)

const name = "miniproxy"
const version = "1.0.0"

type args struct {
	check bool
	print bool
}

func parseArgs() (args, []error) {
	args := args{}
	errors := []error{}
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-check":
			args.check = true
		case "-print":
			args.print = true
		default:
			errors = append(errors, fmt.Errorf("unknown argument '%s'", arg))
		}
	}

	return args, errors
}

func main() {
	args, errors := parseArgs()
	for _, err := range errors {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	l := log.New(os.Stdout, "", log.LstdFlags)

	cfgfile := os.Getenv("CONFIG_FILE")
	if cfgfile == "" {
		cfgfile = "/etc/miniproxy/config.mconf"
	}

	if args.print {
		content, err := os.ReadFile(cfgfile)
		if err != nil {
			l.Fatal("read config: ", err)
		}
		fmt.Printf("%s\n", content)
		if !args.check {
			os.Exit(0)
		}
	}

	cfg, err := config.Parse(cfgfile)
	if err != nil {
		l.Fatal("parse config: ", err)
	}

	if args.check {
		fmt.Printf("Configuration file %s is valid\n", cfgfile)
		os.Exit(0)
	}

	certs := make(map[string]config.TlsConfig)

	for _, mp := range cfg.Mappings {
		src := mp.First
		dst := mp.Second

		l.Printf("Proxying %s to %s\n", src, dst)

		if src.Tls != nil {
			certs[src.Host] = *src.Tls
			l.Printf(" -- with TLS %s\n", src.Tls)
		}
	}
	if cfg.Special404 != nil {
		l.Printf("Special 404 page to %s\n", cfg.Special404)
	}
	if cfg.Special504 != nil {
		l.Printf("Special 504 page to %s\n", cfg.Special504)
	}

	l.Printf("%s version %s\n", name, version)

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
