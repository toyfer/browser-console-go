#!/usr/bin/env bash
# Unix helper (CI linux job uses this). Same pins as vendor-xterm.ps1.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/cmd/browser-console/web/vendor"
mkdir -p "$DEST"
fetch() {
  local url="$1" out="$2"
  echo "[vendor] $out <- $url"
  curl -fsSL "$url" -o "$DEST/$out"
  test -s "$DEST/$out"
}
fetch "https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/css/xterm.css" xterm.css
fetch "https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/lib/xterm.js" xterm.js
fetch "https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.10.0/lib/addon-fit.js" addon-fit.js
fetch "https://cdn.jsdelivr.net/npm/@xterm/addon-web-links@0.11.0/lib/addon-web-links.js" addon-web-links.js
fetch "https://cdn.jsdelivr.net/npm/@xterm/addon-unicode11@0.8.0/lib/addon-unicode11.js" addon-unicode11.js
ls -la "$DEST"
