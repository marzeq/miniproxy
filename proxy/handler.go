package proxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"runtime"
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

	if target.ShellBefore != nil {
		ExecuteShellCommands(target.ShellBefore, h.log)
	}

  proxy.ServeHTTP(w, r)

	if target.ShellAfter != nil {
		ExecuteShellCommands(target.ShellAfter, h.log)
	}
}

func ExecuteShellCommands(commands []string, logger *log.Logger) bool {
	for _, cmd := range commands {
		logger.Printf("Executing shell command '%s'\n", cmd)
		err := ExecuteShellCommand(cmd)
		if err != nil {
			logger.Printf("Error executing shell command '%s': %v\n", cmd, err)
			return false
		}
	}

	return true
}

func ExecuteShellCommand(command string) error {
  var shell string
  var args []string

  switch runtime.GOOS {
  case "windows":
    shell = "cmd.exe"
    args = []string{"/C", command}

  default:
    if _, err := exec.LookPath("bash"); err == nil {
      shell = "bash"
      args = []string{"-c", command}
    } else if _, err := exec.LookPath("sh"); err == nil {
      shell = "sh"
      args = []string{"-c", command}
    } else {
      return fmt.Errorf("no suitable shell found to execute command")
    }
  }

  cmd := exec.Command(shell, args...)
  cmd.Stdout = nil
  cmd.Stderr = nil
  cmd.Stdin = nil

  return cmd.Run()
}
