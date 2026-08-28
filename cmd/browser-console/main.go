package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/toyfer/browser-console-go/internal/config"
	"github.com/toyfer/browser-console-go/internal/server"
)

//go:embed all:web
var webRoot embed.FS

var (
	version = "dev"
	commit  = "none"
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.Printf("browser-console-go %s (%s)", version, commit)

	cfg, err := config.Load()
	if err != nil {
		log.Printf("[config] %v", err)
		log.Printf("place shell.json next to browser-console.exe")
		os.Exit(1)
	}
	log.Printf("[config] loaded: %s", cfg.Path)
	log.Printf("[config] shell: %s", cfg.Shell)
	log.Printf("[config] server: %s", cfg.URL())

	web, err := fs.Sub(webRoot, "web")
	if err != nil {
		log.Fatalf("embed web: %v", err)
	}

	srv := server.New(cfg, web)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("[server] %v", err)
		}
	case s := <-sig:
		log.Printf("[%s] shutting down...", s)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[server] shutdown: %v", err)
		}
	}
}
