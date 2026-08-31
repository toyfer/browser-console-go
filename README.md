# browser-console-go

Windows 専用のブラウザ PTY コンソールを **1 バイナリ** にしたもの。
[toyfer/browser-console](https://github.com/toyfer/browser-console) と同じ UX（xterm.js + `shell.json`）で、Node / node-pty は使わない。

- OS: **Windows 10 1809+ / Windows 11**（ConPTY）
- 配布: `browser-console.exe` + `shell.json`
- 実行時ネット不要（xterm は embed）
- ビルド: GitHub Actions

## 使い方

1. [Releases](https://github.com/toyfer/browser-console-go/releases) から `browser-console-windows-x64.zip` を落とす
2. 展開し、`shell.json` を `browser-console.exe` と同じディレクトリに置く
3. `browser-console.exe` を起動
4. ブラウザで `http://127.0.0.1:8080`

閉域へ持ち込むときは zip をネットありで落としてからフォルダごと。

`shell.json` が見つからない場合、起動時に既定設定で **`shell.json` を自動生成**します（書き込みできない場合はメモリ上の既定値でそのまま動作）。

## shell.json

```json
{
  "shell": "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
  "shellArgs": ["-NoLogo", "-NoExit"],
  "cwd": null,
  "env": { "TERM": "xterm-256color", "COLORTERM": "truecolor" },
  "server": { "host": "127.0.0.1", "port": 8080, "openBrowser": true },
  "pty": { "cols": 120, "rows": 30 },
  "console": { "show": true, "debug": false },
  "ui": {
    "fontFamily": "Consolas, Cascadia Mono, MS Gothic, BIZ UDGothic, monospace",
    "fontSize": 15,
    "fontWeight": "normal",
    "lineHeight": 1.0,
    "theme": "dark"
  }
}
```

- `shell` は絶対パス
- 探索順: exe と同じディレクトリ → cwd
- ファイル名: `shell.json` / `shell.config.json` / `config.json`

### console オプション

| オプション | 既定 | 説明 |
|---|---|---|
| `show` | `true` | ブラウザ上でコンソールパネルを表示するか。`false` で非表示（ヘッドレス実行） |
| `debug` | `false` | `true` にすると `show=false` でもパネルを強制表示し、デバッグ用バナーを追加 |

- `show=false` かつ `debug=false` のとき、シェルプロセスは起動しますがパネルは表示されず、**タブを閉じるとプロセスが終了**します（自動再接続しないため）。
- デバッグしたいときは `debug=true` にするとパネルが表示され、バナーが付きます。

## 操作

| キー | 動作 |
|---|---|
| Tab | 補完 |
| Ctrl+C | 中断 |
| Ctrl+Shift+C | コピー（選択時） |
| Ctrl+Shift+V | ペースト |

## 開発（Windows）

```powershell
go mod tidy
.\scripts\vendor-xterm.ps1
go test ./...
go build -o browser-console.exe .\cmd\browser-console
.\browser-console.exe
```

`vendor-xterm.ps1` はビルド時だけ jsDelivr に接続する。生成した exe は外部通信しない。

## Release

```bash
git tag v0.1.0
git push origin v0.1.0
```

または Actions → Release → Run workflow。

## GitHub MCP から CI ログを読む

Actions のログ API は MCP から取れないので、毎ランの要約を **`ci-status` ブランチ** に書く。

```
ci-status
└─ runs/<run_id>/
    ├─ summary.md      # ref / sha / 成否 / Actions URL
    ├─ test.log        # linux go test
    ├─ build-test.log  # windows go test
    └─ SHA256.txt
```

MCP での例:

1. `list_commits` `owner=toyfer` `repo=browser-console-go` `sha=ci-status`
2. `get_file_contents` `path=runs/<run_id>/summary.md` `ref=ci-status`
3. 同じディレクトリの `*.log`

Actions の UI: https://github.com/toyfer/browser-console-go/actions

## 技術

- HTTP: `net/http`
- WebSocket: `gorilla/websocket`（Origin チェック、接続上限 4）
- PTY: `UserExistsError/conpty` → kernel32 `CreatePseudoConsole`
- UI: `@xterm/xterm@5.5.0` + Fit / WebLinks / Unicode11（embed）
- シャットダウン: Ctrl+C で全 ConPTY を `Close()`

## セキュリティ

- 既定 `127.0.0.1`
- `/health` にシェルパスは出さない
- 公開するなら認証を自分で

## License

MIT
