//go:build tools

package tools

// Keep the Windows-only ConPTY module in go.mod when tidying on Linux.
import _ "github.com/UserExistsError/conpty"
