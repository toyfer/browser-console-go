This directory is embedded into the binary (`//go:embed all:web`).

`vendor/` is filled by `scripts/vendor-xterm.ps1` before `go build`.
Do not commit the downloaded xterm files; CI vendors them on every build.
