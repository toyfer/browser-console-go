package ui

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/toyfer/browser-console-go/internal/config"
)

func IndexHTML(cfg *config.Config) string {
	bg, fg, cursor, sel := "#0c0c0c", "#cccccc", "#cccccc", "#264f78"
	if cfg.UI.Theme == "light" {
		bg, fg, cursor, sel = "#ffffff", "#1e1e1e", "#1e1e1e", "#add6ff"
	}
	fontFamily, _ := json.Marshal(cfg.UI.FontFamily)
	fontWeight, _ := json.Marshal(cfg.UI.FontWeight)
	bgJ, _ := json.Marshal(bg)
	fgJ, _ := json.Marshal(fg)
	cursorJ, _ := json.Marshal(cursor)
	selJ, _ := json.Marshal(sel)

	return fmt.Sprintf(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>Console</title>
<link rel="stylesheet" href="/vendor/xterm.css" />
<style>
  html, body {
    margin: 0; padding: 0; height: 100%%; width: 100%%;
    overflow: hidden; background: %s;
  }
  #terminal {
    position: absolute; inset: 0;
    padding: 0 0 0 2px;
    box-sizing: border-box;
  }
  .xterm, .xterm-viewport, .xterm-screen { height: 100%%; }
  .xterm-viewport { overflow-y: auto !important; }
  .xterm {
    font-variant-ligatures: none;
    font-feature-settings: "liga" 0, "calt" 0;
    text-rendering: geometricPrecision;
  }
  .console-hidden .hidden-overlay {
    display: flex; align-items: center; justify-content: center;
    height: 100%%; box-sizing: border-box; padding: 24px; text-align: center;
    color: #888; font: 14px/1.6 monospace;
  }
  .debug-banner {
    position: fixed; top: 0; left: 0; right: 0; z-index: 10;
    background: #3b78ff; color: #fff; padding: 4px 8px;
    font: 12px/1.5 monospace;
  }
</style>
</head>
<body>
<div id="terminal"></div>
<script src="/vendor/xterm.js"></script>
<script src="/vendor/addon-fit.js"></script>
<script src="/vendor/addon-web-links.js"></script>
<script src="/vendor/addon-unicode11.js"></script>
<script>
(function () {
function ctor(ns, name) {
  if (typeof ns === 'function') return ns;
  if (ns && typeof ns[name] === 'function') return ns[name];
  return null;
}
const Terminal = ctor(window.Terminal, 'Terminal');
const FitAddon = ctor(window.FitAddon, 'FitAddon');
const WebLinksAddon = ctor(window.WebLinksAddon, 'WebLinksAddon');
const Unicode11Addon = ctor(window.Unicode11Addon, 'Unicode11Addon');

if (!Terminal || !FitAddon || !WebLinksAddon || !Unicode11Addon) {
  document.body.innerHTML =
    '<pre style="color:#e74856;padding:16px;font:14px/1.4 monospace">' +
    '[error] local xterm assets missing. Rebuild with scripts/vendor-xterm.ps1 so they are embedded.' +
    '</pre>';
  return;
}

const BOOT = {
  fontFamily: %s,
  fontSize: %s,
  fontWeight: %s,
  lineHeight: %s,
  consoleShow: %s,
  consoleDebug: %s,
  windowsPty: { backend: 'conpty', buildNumber: 22621 },
  theme: {
    background: %s,
    foreground: %s,
    cursor: %s,
    cursorAccent: %s,
    selectionBackground: %s,
    black: '#0c0c0c', red: '#c50f1f', green: '#13a10e', yellow: '#c19c00',
    blue: '#0037da', magenta: '#881798', cyan: '#3a96dd', white: '#cccccc',
    brightBlack: '#767676', brightRed: '#e74856', brightGreen: '#16c60c',
    brightYellow: '#f9f1a5', brightBlue: '#3b78ff', brightMagenta: '#b4009e',
    brightCyan: '#61d6d6', brightWhite: '#f2f2f2',
  }
};

function start() {
  const host = document.getElementById('terminal');
  const hidden = BOOT.consoleShow === false;
  const debug = BOOT.consoleDebug === true;

  if (hidden && !debug) {
    // Headless: spawn the shell but show no terminal, and never auto-reconnect
    // so the process is killed as soon as this tab is closed.
    host.classList.add('console-hidden');
    host.innerHTML = '<div class="hidden-overlay">Console is hidden (console.show=false). Set console.debug=true to view it.</div>';
    const socket = new WebSocket((location.protocol === 'https:' ? 'wss:' : 'ws:') + '//' + location.host + '/ws');
    socket.onmessage = function () {};
    socket.onerror = function () {};
    socket.onclose = function () { socket = null; };
    return;
  }

  if (debug) {
    document.body.classList.add('console-debug');
    const banner = document.createElement('div');
    banner.className = 'debug-banner';
    banner.textContent = 'debug: console shown (console.show=false is overridden by console.debug=true)';
    document.body.prepend(banner);
  }

  const termOptions = {
    fontFamily: BOOT.fontFamily,
    fontSize: BOOT.fontSize,
    fontWeight: BOOT.fontWeight,
    fontWeightBold: 'bold',
    lineHeight: BOOT.lineHeight || 1.0,
    letterSpacing: 0,
    cursorBlink: true,
    cursorStyle: 'block',
    cursorWidth: 1,
    theme: BOOT.theme,
    allowTransparency: false,
    convertEol: false,
    scrollback: 10000,
    macOptionIsMeta: true,
    rightClickSelectsWord: true,
    drawBoldTextInBrightColors: true,
    rescaleOverlappingGlyphs: true,
    allowProposedApi: true,
    smoothScrollDuration: 0,
    ignoreBracketedPasteMode: false,
    windowsPty: BOOT.windowsPty,
  };

  const term = new Terminal(termOptions);
  const fitAddon = new FitAddon();
  const unicode11 = new Unicode11Addon();
  term.loadAddon(fitAddon);
  term.loadAddon(new WebLinksAddon());
  term.loadAddon(unicode11);
  try { term.unicode.activeVersion = '11'; } catch (e) { console.warn('unicode11', e); }

  term.open(host);

  let ws = null;
  let reconnectTimer = null;
  let lastSent = { cols: 0, rows: 0 };
  let resizeTimer = null;
  let intentionalClose = false;

  function propose() {
    let dims = null;
    try { dims = fitAddon.proposeDimensions(); } catch {}
    try { fitAddon.fit(); } catch {}
    const cols = Math.max(2, (dims && dims.cols) || term.cols || 0);
    const rows = Math.max(1, (dims && dims.rows) || term.rows || 0);
    return { cols: cols | 0, rows: rows | 0 };
  }

  function sendResize(force) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    const size = propose();
    if (size.cols < 20 || size.rows < 5) return;
    if (size.cols > 500 || size.rows > 200) return;
    if (!force && size.cols === lastSent.cols && size.rows === lastSent.rows) return;
    lastSent = size;
    try {
      ws.send(JSON.stringify({ type: 'resize', cols: size.cols, rows: size.rows }));
    } catch {}
  }

  function scheduleResize(force) {
    clearTimeout(resizeTimer);
    requestAnimationFrame(function () {
      requestAnimationFrame(function () {
        resizeTimer = setTimeout(function () { sendResize(!!force); }, 80);
      });
    });
  }

  function connect() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const socket = new WebSocket(proto + '//' + location.host + '/ws');
    socket.binaryType = 'arraybuffer';
    ws = socket;

    socket.onopen = function () {
      lastSent = { cols: 0, rows: 0 };
      scheduleResize(true);
      setTimeout(function () { scheduleResize(true); }, 150);
      setTimeout(function () { scheduleResize(true); }, 500);
    };

    socket.onmessage = function (ev) {
      let msg;
      try {
        const text = typeof ev.data === 'string' ? ev.data : new TextDecoder('utf-8').decode(ev.data);
        msg = JSON.parse(text);
      } catch {
        const t = typeof ev.data === 'string' ? ev.data : new TextDecoder('utf-8').decode(ev.data);
        term.write(t);
        return;
      }
      if (msg.type === 'output') {
        term.write(msg.data);
      } else if (msg.type === 'connected') {
        scheduleResize(true);
      } else if (msg.type === 'exit') {
        intentionalClose = true;
        term.writeln('\r\n\x1b[31m[exit ' + msg.code + ']\x1b[0m');
      }
    };

    socket.onclose = function () {
      if (ws === socket) ws = null;
      if (intentionalClose) return;
      clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(connect, 1200);
    };

    socket.onerror = function () {};
  }

  term.onData(function (data) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data: data }));
    }
  });

  term.onBinary(function (data) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data: data }));
    }
  });

  term.attachCustomKeyEventHandler(function (e) {
    if ((e.ctrlKey || e.metaKey) && !e.altKey && (e.key === 'c' || e.key === 'C')) {
      if (e.shiftKey || (e.metaKey && !e.ctrlKey)) {
        if (term.hasSelection()) {
          const selText = term.getSelection();
          if (selText) navigator.clipboard.writeText(selText);
          return false;
        }
      }
    }
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === 'v' || e.key === 'V')) {
      navigator.clipboard.readText().then(function (t) {
        if (t && ws && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'input', data: t }));
        }
      });
      return false;
    }
    return true;
  });

  if (typeof ResizeObserver !== 'undefined') {
    const ro = new ResizeObserver(function () { scheduleResize(false); });
    ro.observe(host);
  }
  window.addEventListener('resize', function () { scheduleResize(false); });
  document.addEventListener('visibilitychange', function () {
    if (!document.hidden) scheduleResize(true);
  });

  propose();
  term.focus();
  connect();
  document.addEventListener('mousedown', function () { term.focus(); });
}

function boot() {
  fetch('/api/ui', { cache: 'no-store' }).then(function (r) {
    return r.ok ? r.json() : null;
  }).then(function (j) {
    if (j) {
      if (j.fontFamily) BOOT.fontFamily = j.fontFamily;
      if (typeof j.fontSize === 'number' && j.fontSize > 0) BOOT.fontSize = j.fontSize;
      if (j.fontWeight) BOOT.fontWeight = j.fontWeight;
      if (typeof j.lineHeight === 'number' && j.lineHeight > 0) BOOT.lineHeight = j.lineHeight;
      if (typeof j.consoleShow === 'boolean') BOOT.consoleShow = j.consoleShow;
      if (typeof j.consoleDebug === 'boolean') BOOT.consoleDebug = j.consoleDebug;
    }
  }).catch(function () {}).then(function () {
    if (!document.fonts) {
      start();
      return;
    }
    const names = BOOT.fontFamily.split(',').map(function (s) {
      return s.trim().replace(/^["']|["']$/g, '');
    }).filter(Boolean).slice(0, 5);
    Promise.all(names.map(function (n) {
      return document.fonts.load(BOOT.fontSize + 'px "' + n + '"').catch(function () { return null; });
    })).then(function () {
      return document.fonts.ready;
    }).catch(function () {}).then(start);
  });
}

boot();
})();
</script>
</body>
</html>`, bg, fontFamily, strconv.FormatFloat(cfg.UI.FontSize, 'f', -1, 64), fontWeight, strconv.FormatFloat(cfg.UI.LineHeight, 'f', -1, 64), strconv.FormatBool(cfg.Console.Show), strconv.FormatBool(cfg.Console.Debug), bgJ, fgJ, cursorJ, bgJ, selJ)
}
