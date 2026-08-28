package config

import "testing"

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
