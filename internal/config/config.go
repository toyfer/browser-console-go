package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const DefaultFont = `Consolas, "Cascadia Mono", "MS Gothic", "BIZ UDGothic", monospace`

var candidates = []string{"shell.json", "shell.config.json", "config.json"}

type Config struct {
	Path      string            `json:"-"`
	Shell     string            `json:"shell"`
	ShellArgs []string          `json:"shellArgs"`
	Cwd       string            `json:"cwd"`
	Env       map[string]string `json:"env"`
	Server    Server            `json:"server"`
	Pty       Pty               `json:"pty"`
	UI        UI                `json:"ui"`
}

type Server struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	OpenBrowser bool   `json:"openBrowser"`
}

type Pty struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type UI struct {
	FontFamily string  `json:"fontFamily"`
	FontSize   float64 `json:"fontSize"`
	FontWeight string  `json:"fontWeight"`
	LineHeight float64 `json:"lineHeight"`
	Theme      string  `json:"theme"`
}

func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return mustAbs(".")
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	return filepath.Dir(resolved)
}

func mustAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}

func searchDirs() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(d string) {
		if d == "" {
			return
		}
		d = mustAbs(d)
		if _, ok := seen[d]; ok {
			return
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	add(exeDir())
	add(".")
	return out
}

func FindPath() (string, error) {
	for _, dir := range searchDirs() {
		for _, name := range candidates {
			p := filepath.Join(dir, name)
			st, err := os.Stat(p)
			if err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("shell.json not found next to the executable or in the working directory")
}

func Load() (*Config, error) {
	path, err := FindPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg, err := Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	cfg.Path = path
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Parse(raw []byte) (*Config, error) {
	var parsed struct {
		Shell     any `json:"shell"`
		ShellArgs any `json:"shellArgs"`
		Cwd       any `json:"cwd"`
		Env       any `json:"env"`
		Server    any `json:"server"`
		Pty       any `json:"pty"`
		UI        any `json:"ui"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	defaultShell := `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	if runtime.GOOS != "windows" {
		defaultShell = "/bin/bash"
	}

	cfg := &Config{
		Shell:     defaultShell,
		ShellArgs: nil,
		Env:       map[string]string{},
		Server: Server{
			Host:        "127.0.0.1",
			Port:        8080,
			OpenBrowser: false,
		},
		Pty: Pty{Cols: 120, Rows: 30},
		UI: UI{
			FontFamily: DefaultFont,
			FontSize:   15,
			FontWeight: "normal",
			LineHeight: 1.0,
			Theme:      "dark",
		},
	}

	if s, ok := parsed.Shell.(string); ok && s != "" {
		cfg.Shell = s
	}
	if arr, ok := parsed.ShellArgs.([]any); ok {
		cfg.ShellArgs = make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				cfg.ShellArgs = append(cfg.ShellArgs, s)
			}
		}
	}
	if s, ok := parsed.Cwd.(string); ok && s != "" {
		cfg.Cwd = s
	}
	if m, ok := parsed.Env.(map[string]any); ok {
		for k, v := range m {
			if s, ok := v.(string); ok {
				cfg.Env[k] = s
			}
		}
	}
	if m, ok := parsed.Server.(map[string]any); ok {
		if s, ok := m["host"].(string); ok && s != "" {
			cfg.Server.Host = s
		}
		switch n := m["port"].(type) {
		case float64:
			cfg.Server.Port = int(n)
		}
		if b, ok := m["openBrowser"].(bool); ok {
			cfg.Server.OpenBrowser = b
		}
	}
	if m, ok := parsed.Pty.(map[string]any); ok {
		if n, ok := m["cols"].(float64); ok {
			cfg.Pty.Cols = int(n)
		}
		if n, ok := m["rows"].(float64); ok {
			cfg.Pty.Rows = int(n)
		}
	}
	if m, ok := parsed.UI.(map[string]any); ok {
		if s, ok := m["fontFamily"].(string); ok && s != "" {
			cfg.UI.FontFamily = s
		}
		if n, ok := m["fontSize"].(float64); ok && n > 0 {
			cfg.UI.FontSize = n
		}
		if s, ok := m["fontWeight"].(string); ok && s != "" {
			cfg.UI.FontWeight = s
		}
		if n, ok := m["lineHeight"].(float64); ok && n > 0 {
			cfg.UI.LineHeight = n
		}
		if s, ok := m["theme"].(string); ok && s == "light" {
			cfg.UI.Theme = "light"
		}
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Shell == "" {
		return fmt.Errorf("shell is empty")
	}
	if _, err := os.Stat(c.Shell); err != nil {
		return fmt.Errorf("shell not found: %s", c.Shell)
	}
	if c.Cwd != "" {
		if st, err := os.Stat(c.Cwd); err != nil || !st.IsDir() {
			return fmt.Errorf("cwd not found: %s", c.Cwd)
		}
	}
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}
	if c.Pty.Cols < 20 {
		c.Pty.Cols = 20
	}
	if c.Pty.Rows < 5 {
		c.Pty.Rows = 5
	}
	return nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) URL() string {
	return fmt.Sprintf("http://%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) PublicUI() map[string]any {
	return map[string]any{
		"fontFamily": c.UI.FontFamily,
		"fontSize":   c.UI.FontSize,
		"fontWeight": c.UI.FontWeight,
		"lineHeight": c.UI.LineHeight,
		"theme":      c.UI.Theme,
		"ptyBackend": "conpty",
	}
}

func (c *Config) BuildEnv() []string {
	out := os.Environ()
	replace := map[string]string{}
	for k, v := range c.Env {
		replace[k] = v
	}
	if _, ok := replace["TERM"]; !ok {
		replace["TERM"] = "xterm-256color"
	}
	if _, ok := replace["COLORTERM"]; !ok {
		replace["COLORTERM"] = "truecolor"
	}
	if _, ok := replace["TERM_PROGRAM"]; !ok {
		replace["TERM_PROGRAM"] = "browser-console"
	}
	seen := map[string]int{}
	for i, kv := range out {
		for j := 0; j < len(kv); j++ {
			if kv[j] == '=' {
				seen[kv[:j]] = i
				break
			}
		}
	}
	for k, v := range replace {
		entry := k + "=" + v
		if i, ok := seen[k]; ok {
			out[i] = entry
		} else {
			out = append(out, entry)
		}
	}
	return out
}
