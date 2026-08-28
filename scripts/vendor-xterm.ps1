# Download pinned xterm UMD + CSS into cmd/browser-console/web/vendor.
# No npm. Uses jsDelivr npm CDN at build time only.
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
if (-not $Root) { $Root = Get-Location }
$Dest = Join-Path $Root "cmd\browser-console\web\vendor"
New-Item -ItemType Directory -Force -Path $Dest | Out-Null

$files = @(
  @{ Url = "https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/css/xterm.css"; Out = "xterm.css" },
  @{ Url = "https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/lib/xterm.js"; Out = "xterm.js" },
  @{ Url = "https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.10.0/lib/addon-fit.js"; Out = "addon-fit.js" },
  @{ Url = "https://cdn.jsdelivr.net/npm/@xterm/addon-web-links@0.11.0/lib/addon-web-links.js"; Out = "addon-web-links.js" },
  @{ Url = "https://cdn.jsdelivr.net/npm/@xterm/addon-unicode11@0.8.0/lib/addon-unicode11.js"; Out = "addon-unicode11.js" }
)

foreach ($f in $files) {
  $out = Join-Path $Dest $f.Out
  Write-Host "[vendor] $($f.Out) <- $($f.Url)"
  Invoke-WebRequest -Uri $f.Url -OutFile $out -UseBasicParsing
  if ((Get-Item $out).Length -lt 100) { throw "$($f.Out) too small" }
}
Write-Host "[vendor] wrote $Dest"
Get-ChildItem $Dest | Format-Table Name, Length
