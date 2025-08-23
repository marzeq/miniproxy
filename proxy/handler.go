package proxy

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/marzeq/miniproxy/config"
)

type Handler struct {
  config *config.Config
  log    *log.Logger
}

func NewHandler(cfg *config.Config, l *log.Logger) http.Handler {
  return &Handler{cfg, l}
}


func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.log.Printf("Incoming request: %s %s from %s\n", r.Method, r.URL, r.RemoteAddr)

  host := r.Host

  var target *config.ProxyDest
  for _, mp := range h.config.Mappings {
		src := mp.First
		dst := mp.Second
    if ProxySourceMatchesHost(src, host) {
      target = dst
      break
    }
  }

	if target == nil {
		if h.config.Special404 != nil {
			target = h.config.Special404
		} else {
      h.log.Println("No proxy target found for incoming host:", host)
      w.WriteHeader(http.StatusBadRequest)
      return
    }
  }

  if target.ServeFrom != "" {
    http.StripPrefix("/", http.FileServer(http.Dir(target.ServeFrom))).ServeHTTP(w, r)
    h.log.Printf("%s -> serving static files from %s\n", host, target.ServeFrom)
    return
  }

  if target.Host == "" {
    h.log.Println("Proxy target has empty host for incoming host:", host)
    w.WriteHeader(http.StatusInternalServerError)
    return
  }

  h.log.Printf("%s -> %s\n", host, target.Host)

  targetHost := target.Host
  if !strings.HasPrefix(targetHost, "http://") && !strings.HasPrefix(targetHost, "https://") {
    targetHost = "http://" + targetHost
  }

  targetURL, err := url.Parse(targetHost)
  if err != nil {
    h.log.Println("Invalid target URL:", targetHost, err)
    w.WriteHeader(http.StatusBadGateway)
    return
  }

  proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
  proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
    if h.config.Special504 != nil {
			fallback := h.config.Special504
      if fallback.ServeFrom != "" {
        http.StripPrefix("/", http.FileServer(http.Dir(fallback.ServeFrom))).ServeHTTP(w, r)
        return
      }
      if fallback.Host != "" {
        fallbackURL, parseErr := url.Parse(fallback.Host)
        if parseErr != nil {
          h.log.Println("Invalid 504 fallback URL:", fallback.Host, parseErr)
          http.Error(w, "Invalid fallback URL", http.StatusInternalServerError)
          return
        }
        httputil.NewSingleHostReverseProxy(fallbackURL).ServeHTTP(w, r)
        return
      }
		}

		h.log.Println("Upstream request failed:", err)
		w.WriteHeader(http.StatusGatewayTimeout)
  }

  proxy.ServeHTTP(w, r)
}
