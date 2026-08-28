package pty

import "io"

// Session is a running ConPTY (or a stub on non-Windows).
type Session interface {
	io.ReadWriteCloser
	Resize(cols, rows int) error
	Pid() int
	Wait() (uint32, error)
}
