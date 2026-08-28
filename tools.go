// Package keepdeps exists so `go mod tidy` on Linux still pins
// the Windows-only ConPTY module. It is not imported by the binary.
package keepdeps

import _ "github.com/UserExistsError/conpty"
