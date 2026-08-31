package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("host = %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("port = %d", cfg.Server.Port)
	}
	if cfg.UI.Theme != "dark" {
		t.Fatalf("theme = %q", cfg.UI.Theme)
	}
	if cfg.Pty.Cols != 120 || cfg.Pty.Rows != 30 {
		t.Fatalf("pty = %dx%d", cfg.Pty.Cols, cfg.Pty.Rows)
	}
}

func TestParseFull(t *testing.T) {
	raw := []byte(`{
  "shell": "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
  "shellArgs": ["-NoLogo", "-NoExit"],
  "cwd": null,
  "env": {"TERM": "xterm-256color"},
  "server": {"host": "127.0.0.1", "port": 9090, "openBrowser": true},
  "pty": {"cols": 80, "rows": 24},
  "ui": {"fontFamily": "Consolas", "fontSize": 16, "fontWeight": "bold", "lineHeight": 1.2, "theme": "light"}
}`)
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9090 || !cfg.Server.OpenBrowser {
		t.Fatalf("server %+v", cfg.Server)
	}
	if cfg.Pty.Cols != 80 || cfg.Pty.Rows != 24 {
		t.Fatalf("pty %+v", cfg.Pty)
	}
	if cfg.UI.Theme != "light" || cfg.UI.FontSize != 16 {
		t.Fatalf("ui %+v", cfg.UI)
	}
	if len(cfg.ShellArgs) != 2 || cfg.ShellArgs[0] != "-NoLogo" {
		t.Fatalf("args %#v", cfg.ShellArgs)
	}
	ui := cfg.PublicUI()
	if _, ok := ui["shell"]; ok {
		t.Fatal("public UI must not include shell path")
	}
}

func TestParseRejectsBadJSON(t *testing.T) {
	if _, err := Parse([]byte(`{`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultConsoleVisible(t *testing.T) {
	cfg := Default()
	if !cfg.Console.Show || cfg.Console.Debug {
		t.Fatalf("default console = %+v", cfg.Console)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 8080 {
		t.Fatalf("default server = %+v", cfg.Server)
	}
	if cfg.Shell == "" {
		t.Fatal("default shell is empty")
	}
}

func TestParseConsoleOptions(t *testing.T) {
	cfg, err := Parse([]byte(`{"console":{"show":false,"debug":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Console.Show || !cfg.Console.Debug {
		t.Fatalf("console = %+v", cfg.Console)
	}

	ui := cfg.PublicUI()
	if ui["consoleShow"] != false || ui["consoleDebug"] != true {
		t.Fatalf("public console = %+v", ui)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "shell.json")
	cfg := Default()
	cfg.Console.Debug = true
	if err := cfg.Save(p); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg2, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg2.Console.Show || !cfg2.Console.Debug {
		t.Fatalf("console lost in round-trip: %+v", cfg2.Console)
	}
	if cfg2.Server.Port != cfg.Server.Port {
		t.Fatalf("server.port lost: %d != %d", cfg2.Server.Port, cfg.Server.Port)
	}
}

func TestLoadFallbackWritesDefault(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "shell.json")

	cfg, err := loadFallback(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Path != p {
		t.Fatalf("path = %q, want %q", cfg.Path, p)
	}
	if !cfg.Server.OpenBrowser {
		t.Fatal("generated config should open the browser (matches shipped example)")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("fallback config invalid: %v", err)
	}

	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, `"cwd"`) {
		t.Error("generated file should omit empty cwd")
	}
	if strings.Contains(s, `"shellArgs": null`) {
		t.Error("generated file should use [] for shellArgs, not null")
	}
	if !strings.Contains(s, `"console"`) {
		t.Error("generated file should include the console section")
	}
	if !strings.Contains(s, `"openBrowser": true`) {
		t.Error(`generated file should contain "openBrowser": true`)
	}

	cfg2, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg2.Server.OpenBrowser {
		t.Error("openBrowser lost in round-trip")
	}
	if cfg2.Console.Show != cfg.Console.Show || cfg2.Console.Debug != cfg.Console.Debug {
		t.Errorf("console lost in round-trip: %+v", cfg2.Console)
	}
	if err := cfg2.Validate(); err != nil {
		t.Fatalf("round-tripped config invalid: %v", err)
	}
}
