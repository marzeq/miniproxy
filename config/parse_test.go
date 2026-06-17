package config

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseCGIStringTarget(t *testing.T) {
	cfg := parseConfigFromText(t, `"example.com" = { cgi = "./cgi-bin/app" }`)

	if len(cfg.Mappings) != 1 {
		t.Fatalf("got %d mappings, want 1", len(cfg.Mappings))
	}

	target := cfg.Mappings[0].Second
	if target.CGI == nil {
		t.Fatal("expected CGI target to be set")
	}
	if target.CGI.Path != "./cgi-bin/app" {
		t.Fatalf("got CGI path %q, want %q", target.CGI.Path, "./cgi-bin/app")
	}
}

func TestParseCGIObjectTarget(t *testing.T) {
	cfg := parseConfigFromText(t, strings.Join([]string{
		`"example.com" = {`,
		`  cgi = {`,
		`    path = "./cgi-bin/app"`,
		`    dir = "./cgi-bin"`,
		`    args = ["--mode", "test"]`,
		`    env = {`,
		`      APP_ENV = "test"`,
		`      FEATURE_FLAG = "on"`,
		`    }`,
		`    inherit_env = ["PATH", "HOME"]`,
		`  }`,
		`}`,
	}, "\n"))

	target := cfg.Mappings[0].Second
	if target.CGI == nil {
		t.Fatal("expected CGI target to be set")
	}

	if target.CGI.Path != "./cgi-bin/app" {
		t.Fatalf("got CGI path %q, want %q", target.CGI.Path, "./cgi-bin/app")
	}
	if target.CGI.Dir != "./cgi-bin" {
		t.Fatalf("got CGI dir %q, want %q", target.CGI.Dir, "./cgi-bin")
	}
	if !reflect.DeepEqual(target.CGI.Args, []string{"--mode", "test"}) {
		t.Fatalf("got CGI args %v", target.CGI.Args)
	}
	if !reflect.DeepEqual(target.CGI.Env, []string{"APP_ENV=test", "FEATURE_FLAG=on"}) {
		t.Fatalf("got CGI env %v", target.CGI.Env)
	}
	if !reflect.DeepEqual(target.CGI.InheritEnv, []string{"PATH", "HOME"}) {
		t.Fatalf("got CGI inherit_env %v", target.CGI.InheritEnv)
	}
}

func TestParseRejectsMultipleDestinationKinds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.mconf")
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		`"example.com" = {`,
		`  host = "localhost:3000"`,
		`  cgi = "./cgi-bin/app"`,
		`}`,
	}, "\n")), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "only one of dest, host, serve_from or cgi") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAllowsShellBeforeOnlyTarget(t *testing.T) {
	cfg := parseConfigFromText(t, strings.Join([]string{
		`"example.com" = {`,
		`  shell_before = ["echo webhook"]`,
		`}`,
	}, "\n"))

	if len(cfg.Mappings) != 1 {
		t.Fatalf("got %d mappings, want 1", len(cfg.Mappings))
	}

	target := cfg.Mappings[0].Second
	if target.ShellBefore == nil {
		t.Fatal("expected shell_before to be set")
	}
	if !reflect.DeepEqual(target.ShellBefore, []string{"echo webhook"}) {
		t.Fatalf("got shell_before %v", target.ShellBefore)
	}
	if target.Host != "" || target.ServeFrom != "" || target.CGI != nil {
		t.Fatalf("expected shell-only target, got %#v", target)
	}
}

func TestParseSourcePathAndQuery(t *testing.T) {
	cfg := parseConfigFromText(t, `"example.com/api/users?debug=1&tag=v2" = "localhost:3000"`)

	if len(cfg.Mappings) != 1 {
		t.Fatalf("got %d mappings, want 1", len(cfg.Mappings))
	}

	source := cfg.Mappings[0].First
	if source.Host != "example.com" {
		t.Fatalf("got host %q, want %q", source.Host, "example.com")
	}
	if source.Path != "/api/users" {
		t.Fatalf("got path %q, want %q", source.Path, "/api/users")
	}
	if !reflect.DeepEqual(source.Query, url.Values{"debug": {"1"}, "tag": {"v2"}}) {
		t.Fatalf("got query %v", source.Query)
	}
}

func parseConfigFromText(t *testing.T, contents string) *Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.mconf")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	return cfg
}
