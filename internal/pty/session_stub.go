//go:build !windows

package pty

import (
	"fmt"

	"github.com/toyfer/browser-console-go/internal/config"
)

func Start(_ *config.Config) (Session, error) {
	return nil, fmt.Errorf("browser-console-go is Windows-only (ConPTY)")
}
