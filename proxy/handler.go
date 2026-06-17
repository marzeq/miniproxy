package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cgi"
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

type intercept404 struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (r *intercept404) WriteHeader(code int) {
	r.status = code
}

func (r *intercept404) Write(b []byte) (int, error) {
	return r.buf.Write(b)
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

	h.serveTarget(w, r, host, target, true)
}

func (h *Handler) serveTarget(w http.ResponseWriter, r *http.Request, host string, target *config.ProxyDest, allow504Fallback bool) {
	if target == nil {
		h.log.Println("Proxy target is missing for incoming host:", host)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if target.ShellBefore != nil {
		ExecuteShellCommands(target.ShellBefore, h.log)
	}

	if target.ServeFrom != "" {
		h.log.Printf("%s -> serving static files from %s\n", host, target.ServeFrom)
		h.serveStatic(w, r, target.ServeFrom)
		if target.ShellAfter != nil {
			ExecuteShellCommands(target.ShellAfter, h.log)
		}
		return
	}

	if target.CGI != nil {
		h.log.Printf("%s -> cgi %s\n", host, target.CGI.Path)
		cgiHandler := &cgi.Handler{
			Path:       target.CGI.Path,
			Root:       "/",
			Dir:        target.CGI.Dir,
			Env:        target.CGI.Env,
			InheritEnv: target.CGI.InheritEnv,
			Logger:     h.log,
			Args:       target.CGI.Args,
			Stderr:     h.log.Writer(),
		}
		cgiHandler.ServeHTTP(w, r)
		if target.ShellAfter != nil {
			ExecuteShellCommands(target.ShellAfter, h.log)
		}
		return
	}

	if target.Host == "" {
		if target.ShellBefore != nil {
			if target.ShellAfter != nil {
				ExecuteShellCommands(target.ShellAfter, h.log)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		h.log.Println("Proxy target has empty host for incoming host:", host)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.serveReverseProxy(w, r, host, target, allow504Fallback)
	if target.ShellAfter != nil {
		ExecuteShellCommands(target.ShellAfter, h.log)
	}
}

func (h *Handler) serveStatic(w http.ResponseWriter, r *http.Request, root string) {
	fs := http.Dir(root)
	fileServer := http.StripPrefix("/", http.FileServer(fs))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &intercept404{ResponseWriter: w, status: http.StatusOK}

		fileServer.ServeHTTP(rec, r)

		if rec.status == http.StatusNotFound {
			if f, err := fs.Open("/404.html"); err == nil {
				defer f.Close()

				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				io.Copy(w, f)
				return
			}
			http.NotFound(w, r)
			return
		}

		if rec.status != 0 {
			w.WriteHeader(rec.status)
		}
		w.Write(rec.buf.Bytes())
	})

	handler.ServeHTTP(w, r)
}

func (h *Handler) serveReverseProxy(w http.ResponseWriter, r *http.Request, host string, target *config.ProxyDest, allow504Fallback bool) {
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
		if allow504Fallback && h.config.Special504 != nil && h.config.Special504 != target {
			h.serveTarget(w, r, host, h.config.Special504, false)
			return
		}

		h.log.Println("Upstream request failed:", err)
		w.WriteHeader(http.StatusGatewayTimeout)
	}

	proxy.ServeHTTP(w, r)
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
