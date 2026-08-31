package config

import (
	"encoding/json"
	"fmt"
	"log"
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
	Console   Console           `json:"console"`
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

// Console controls how the browser console panel is shown and how the shell
// process is tied to the lifetime of the page.
type Console struct {
	// Show renders the terminal panel. When Show is false and Debug is false,
	// the page runs the shell headless and the process is killed as soon as the
	// tab is closed (no auto-reconnect).
	Show bool `json:"show"`
	// Debug forces the terminal panel to be shown (overriding Show) and adds a
	// small debug banner. Handy for inspecting a headless session.
	Debug bool `json:"debug"`
}

// Default returns a Config populated with built-in defaults. It is used both
// as the basis for Parse() and as the fallback when no config file is found.
func Default() *Config {
	return &Config{
		Shell:     defaultShell(),
		ShellArgs: nil,
		Env:       map[string]string{},
		Server:    Server{Host: "127.0.0.1", Port: 8080, OpenBrowser: false},
		Pty:       Pty{Cols: 120, Rows: 30},
		UI:        UI{FontFamily: DefaultFont, FontSize: 15, FontWeight: "normal", LineHeight: 1.0, Theme: "dark"},
		Console:   Console{Show: true, Debug: false},
	}
}

// defaultShell returns the first existing shell among common Windows shells,
// or /bin/bash on non-Windows platforms.
func defaultShell() string {
	if runtime.GOOS != "windows" {
		return "/bin/bash"
	}
	candidates := []string{
		`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
		`C:\Windows\System32\WindowsPowerShell\v1.0\pwsh.exe`,
		`C:\Windows\System32\cmd.exe`,
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return candidates[0]
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

// DefaultPath returns the path where a default shell.json is created when no
// config file exists.
func DefaultPath() string {
	return filepath.Join(exeDir(), "shell.json")
}

// Save writes the config as indented JSON to path.
func (c *Config) Save(path string) error {
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// Load reads the first available config file. If none is found it falls back
// to built-in defaults and best-effort creates a shell.json next to the
// executable so the user can discover and edit it. If the file cannot be
// written, it keeps running with the in-memory default config.
func Load() (*Config, error) {
	path, err := FindPath()
	if err != nil {
		cfg := Default()
		defaultPath := DefaultPath()
		if werr := cfg.Save(defaultPath); werr != nil {
			log.Printf("[config] no shell.json found; using in-memory defaults (could not write %s: %v)", defaultPath, werr)
			cfg.Path = ""
			return cfg, nil
		}
		log.Printf("[config] no shell.json found; wrote default config to %s", defaultPath)
		cfg.Path = defaultPath
		if verr := cfg.Validate(); verr != nil {
			return cfg, verr
		}
		return cfg, nil
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
		Console   any `json:"console"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	cfg := Default()

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
	if m, ok := parsed.Console.(map[string]any); ok {
		if b, ok := m["show"].(bool); ok {
			cfg.Console.Show = b
		}
		if b, ok := m["debug"].(bool); ok {
			cfg.Console.Debug = b
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
		"fontFamily":   c.UI.FontFamily,
		"fontSize":     c.UI.FontSize,
		"fontWeight":   c.UI.FontWeight,
		"lineHeight":   c.UI.LineHeight,
		"theme":        c.UI.Theme,
		"ptyBackend":   "conpty",
		"consoleShow":  c.Console.Show,
		"consoleDebug": c.Console.Debug,
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
