//go:build windows

package pty

import (
	"context"
	"fmt"

	"github.com/UserExistsError/conpty"

	"github.com/toyfer/browser-console-go/internal/config"
	"github.com/toyfer/browser-console-go/internal/winquote"
)

type conptySession struct {
	cpty *conpty.ConPty
}

func Start(cfg *config.Config) (Session, error) {
	if !conpty.IsConPtyAvailable() {
		return nil, fmt.Errorf("ConPTY is not available on this Windows version (need Windows 10 1809+)")
	}
	argv := append([]string{cfg.Shell}, cfg.ShellArgs...)
	cmdline := winquote.CommandLine(argv)

	opts := []conpty.ConPtyOption{
		conpty.ConPtyDimensions(cfg.Pty.Cols, cfg.Pty.Rows),
		conpty.ConPtyEnv(cfg.BuildEnv()),
	}
	if cfg.Cwd != "" {
		opts = append(opts, conpty.ConPtyWorkDir(cfg.Cwd))
	}

	cpty, err := conpty.Start(cmdline, opts...)
	if err != nil {
		return nil, fmt.Errorf("conpty start: %w", err)
	}
	return &conptySession{cpty: cpty}, nil
}

func (s *conptySession) Read(p []byte) (int, error)  { return s.cpty.Read(p) }
func (s *conptySession) Write(p []byte) (int, error) { return s.cpty.Write(p) }
func (s *conptySession) Close() error                { return s.cpty.Close() }
func (s *conptySession) Resize(cols, rows int) error { return s.cpty.Resize(cols, rows) }
func (s *conptySession) Pid() int                    { return s.cpty.Pid() }
func (s *conptySession) Wait() (uint32, error) {
	return s.cpty.Wait(context.Background())
}
