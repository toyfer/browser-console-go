package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/toyfer/browser-console-go/internal/config"
	"github.com/toyfer/browser-console-go/internal/pty"
	"github.com/toyfer/browser-console-go/internal/ui"
)

const maxSessions = 4

type Server struct {
	cfg      *config.Config
	web      fs.FS
	http     *http.Server
	upgrader websocket.Upgrader
	mu       sync.Mutex
	sessions map[*session]struct{}
	active   atomic.Int32
}

type session struct {
	pty pty.Session
}

func New(cfg *config.Config, web fs.FS) *Server {
	s := &Server{
		cfg:      cfg,
		web:      web,
		sessions: map[*session]struct{}{},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return originOK(r, cfg)
			},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/ui", s.handleUI)
	mux.Handle("/vendor/", http.StripPrefix("/vendor/", http.FileServer(http.FS(mustSub(web, "vendor")))))
	mux.HandleFunc("/ws", s.handleWS)
	s.http = &http.Server{
		Addr:              cfg.Addr(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		return fsys
	}
	return sub
}

func originOK(r *http.Request, cfg *config.Config) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	allow := []string{
		fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port),
		fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.Port),
		fmt.Sprintf("http://localhost:%d", cfg.Server.Port),
	}
	for _, a := range allow {
		if o == a {
			return true
		}
	}
	log.Printf("[ws] reject origin %q", o)
	return false
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.cfg.Addr())
	if err != nil {
		return err
	}
	log.Printf("")
	log.Printf("========================================")
	log.Printf("  Browser Console (Go / ConPTY)")
	log.Printf("  URL:   %s", s.cfg.URL())
	log.Printf("  Shell: %s %s", s.cfg.Shell, strings.Join(s.cfg.ShellArgs, " "))
	log.Printf("  Font:  %s", s.cfg.UI.FontFamily)
	log.Printf("========================================")
	log.Printf("open the URL above. Ctrl+C to stop.")
	if s.cfg.Server.OpenBrowser {
		go openBrowser(s.cfg.URL())
	}
	return s.http.Serve(ln)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	for sess := range s.sessions {
		_ = sess.pty.Close()
	}
	s.sessions = map[*session]struct{}{}
	s.mu.Unlock()
	return s.http.Shutdown(ctx)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, ui.IndexHTML(s.cfg))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":      "ok",
		"pid":         os.Getpid(),
		"pty":         true,
		"backend":     "conpty",
		"sessions":    s.active.Load(),
		"maxSessions": maxSessions,
		"goos":        runtime.GOOS,
	})
}

func (s *Server) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s.cfg.PublicUI())
}

type wsConn struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (c *wsConn) json(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.conn.WriteJSON(v)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if s.active.Load() >= maxSessions {
		http.Error(w, "too many sessions", http.StatusTooManyRequests)
		return
	}
	rawConn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade: %v", err)
		return
	}
	defer rawConn.Close()
	conn := &wsConn{conn: rawConn}

	p, err := pty.Start(s.cfg)
	if err != nil {
		log.Printf("[pty] start: %v", err)
		_ = conn.json(map[string]any{"type": "exit", "code": 1, "error": err.Error()})
		return
	}
	sess := &session{pty: p}
	s.mu.Lock()
	s.sessions[sess] = struct{}{}
	s.mu.Unlock()
	s.active.Add(1)
	defer func() {
		_ = p.Close()
		s.mu.Lock()
		delete(s.sessions, sess)
		s.mu.Unlock()
		s.active.Add(-1)
	}()

	log.Printf("[pty] spawned pid=%d sessions=%d", p.Pid(), s.active.Load())
	_ = conn.json(map[string]any{
		"type":    "connected",
		"pid":     p.Pid(),
		"pty":     true,
		"backend": "conpty",
		"ui":      s.cfg.PublicUI(),
	})

	done := make(chan struct{})
	var once sync.Once
	closeDone := func() {
		once.Do(func() {
			close(done)
			_ = rawConn.Close()
		})
	}

	go func() {
		defer closeDone()
		buf := make([]byte, 32*1024)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				if werr := conn.json(map[string]any{"type": "output", "data": string(buf[:n])}); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		code, err := p.Wait()
		if err != nil {
			log.Printf("[pty] wait: %v", err)
		}
		_ = conn.json(map[string]any{"type": "exit", "code": code})
		closeDone()
	}()

	rawConn.SetReadLimit(8 << 20)
	for {
		select {
		case <-done:
			return
		default:
		}
		_ = rawConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		_, raw, err := rawConn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			Type string  `json:"type"`
			Data string  `json:"data"`
			Cols float64 `json:"cols"`
			Rows float64 `json:"rows"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			_, _ = p.Write(raw)
			continue
		}
		switch msg.Type {
		case "input":
			_, _ = p.Write([]byte(msg.Data))
		case "resize":
			cols := int(msg.Cols)
			rows := int(msg.Rows)
			if cols >= 20 && rows >= 5 && cols <= 400 && rows <= 150 {
				if err := p.Resize(cols, rows); err != nil {
					log.Printf("[pty] resize: %v", err)
				}
			}
		case "ping":
			_ = conn.json(map[string]any{"type": "pong"})
		}
	}
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		_ = exec.Command("xdg-open", url).Start()
	}
}
