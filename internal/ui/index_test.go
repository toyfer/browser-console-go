package ui

import (
	"strings"
	"testing"

	"github.com/toyfer/browser-console-go/internal/config"
)

func TestIndexHTMLLocalAssets(t *testing.T) {
	cfg, err := config.Parse([]byte(`{"ui":{"theme":"dark"}}`))
	if err != nil {
		t.Fatal(err)
	}
	html := IndexHTML(cfg)
	need := []string{
		"function ctor",
		"/vendor/xterm.js",
		"/ws",
		"windowsPty",
		"conpty",
	}
	for _, s := range need {
		if !strings.Contains(html, s) {
			t.Errorf("missing %q", s)
		}
	}
	for _, s := range []string{"jsdelivr", "cdn.", "unpkg"} {
		if strings.Contains(html, s) {
			t.Errorf("must not reference CDN %q", s)
		}
	}
}

func TestIndexHTMLEmbedConsoleOptions(t *testing.T) {
	cfg, err := config.Parse([]byte(`{"console":{"show":false,"debug":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	html := IndexHTML(cfg)
	need := []string{
		"consoleShow",
		"consoleDebug",
		"console-hidden",
		"debug-banner",
		"Console is hidden (console.show=false)",
	}
	for _, s := range need {
		if !strings.Contains(html, s) {
			t.Errorf("missing %q", s)
		}
	}
}

func TestIndexHTMLHeadlessSocketReassignable(t *testing.T) {
	cfg, err := config.Parse([]byte(`{"console":{"show":false,"debug":false}}`))
	if err != nil {
		t.Fatal(err)
	}
	html := IndexHTML(cfg)
	if !strings.Contains(html, "let socket = null;") {
		t.Error("headless socket must be declared with let so onclose can clear it")
	}
	if strings.Contains(html, "const socket = null;") {
		t.Error("headless socket must not be const (Assignment to constant variable)")
	}
	if !strings.Contains(html, "noReconnect") {
		t.Error("missing noReconnect flag for show=false sessions")
	}
}
