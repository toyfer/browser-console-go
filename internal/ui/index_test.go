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
