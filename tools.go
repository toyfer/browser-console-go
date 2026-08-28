//go:build tools

package tools

// Keep Windows-only modules in go.mod when tidying on Linux CI.
import (
	_ "github.com/UserExistsError/conpty"
)
