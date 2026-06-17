package proxy

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/marzeq/miniproxy/config"
)

func TestCGITargetServesResponse(t *testing.T) {
	handler := NewHandler(&config.Config{
		Mappings: []config.Pair[*config.ProxySource, *config.ProxyDest]{
			{
				First: &config.ProxySource{HostPath: "example.com"},
				Second: &config.ProxyDest{
					CGI: helperCGIConfig("echo"),
				},
			},
		},
	}, log.New(io.Discard, "", 0))

	req := httptest.NewRequest(http.MethodPost, "http://example.com/upload?debug=1", strings.NewReader("payload"))
	req.Host = "example.com"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("got status %d, want %d", res.StatusCode, http.StatusCreated)
	}
	if got := res.Header.Get("X-Path-Info"); got != "/upload" {
		t.Fatalf("got X-Path-Info %q, want %q", got, "/upload")
	}
	if got := res.Header.Get("X-Query-String"); got != "debug=1" {
		t.Fatalf("got X-Query-String %q, want %q", got, "debug=1")
	}
	if got := res.Header.Get("X-Request-Method"); got != http.MethodPost {
		t.Fatalf("got X-Request-Method %q, want %q", got, http.MethodPost)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "stdin=payload" {
		t.Fatalf("got body %q, want %q", string(body), "stdin=payload")
	}
}

func TestSpecial504CanUseCGI(t *testing.T) {
	handler := NewHandler(&config.Config{
		Mappings: []config.Pair[*config.ProxySource, *config.ProxyDest]{
			{
				First: &config.ProxySource{HostPath: "example.com"},
				Second: &config.ProxyDest{
					Host: "http://127.0.0.1:1",
				},
			},
		},
		Special504: &config.ProxyDest{
			CGI: helperCGIConfig("gateway-timeout"),
		},
	}, log.New(io.Discard, "", 0))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("got status %d, want %d", res.StatusCode, http.StatusGatewayTimeout)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "fallback=1" {
		t.Fatalf("got body %q, want %q", string(body), "fallback=1")
	}
}

func TestShellBeforeOnlyTargetReturnsBlankOK(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "shell-before.out")

	handler := NewHandler(&config.Config{
		Mappings: []config.Pair[*config.ProxySource, *config.ProxyDest]{
			{
				First: &config.ProxySource{HostPath: "example.com"},
				Second: &config.ProxyDest{
					ShellBefore: []string{helperShellWriteFileCommand(outputPath)},
				},
			},
		},
	}, log.New(io.Discard, "", 0))

	req := httptest.NewRequest(http.MethodPost, "http://example.com/webhook", strings.NewReader("payload"))
	req.Host = "example.com"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", res.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) != 0 {
		t.Fatalf("got body %q, want empty body", string(body))
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read shell output: %v", err)
	}
	if strings.TrimSpace(string(output)) != "shell-before" {
		t.Fatalf("got shell output %q, want %q", string(output), "shell-before")
	}
}

func TestCGIHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	switch os.Getenv("CGI_HELPER_MODE") {
	case "echo":
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		os.Stdout.WriteString("Status: 201 Created\r\n")
		os.Stdout.WriteString("Content-Type: text/plain\r\n")
		os.Stdout.WriteString("X-Path-Info: " + os.Getenv("PATH_INFO") + "\r\n")
		os.Stdout.WriteString("X-Query-String: " + os.Getenv("QUERY_STRING") + "\r\n")
		os.Stdout.WriteString("X-Request-Method: " + os.Getenv("REQUEST_METHOD") + "\r\n")
		os.Stdout.WriteString("\r\n")
		os.Stdout.WriteString("stdin=" + string(body))
	case "gateway-timeout":
		os.Stdout.WriteString("Status: 504 Gateway Timeout\r\n")
		os.Stdout.WriteString("Content-Type: text/plain\r\n")
		os.Stdout.WriteString("\r\n")
		os.Stdout.WriteString("fallback=1")
	default:
		panic("unknown CGI helper mode")
	}

	os.Exit(0)
}

func helperCGIConfig(mode string) *config.CgiConfig {
	return &config.CgiConfig{
		Path: os.Args[0],
		Args: []string{"-test.run=TestCGIHelperProcess"},
		Env: []string{
			"GO_WANT_HELPER_PROCESS=1",
			"CGI_HELPER_MODE=" + mode,
		},
		InheritEnv: []string{"PATH"},
	}
}

func helperShellWriteFileCommand(path string) string {
	switch runtime.GOOS {
	case "windows":
		return `echo shell-before>"` + path + `"`
	default:
		return `printf '%s' 'shell-before' > '` + path + `'`
	}
}
