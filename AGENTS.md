# Mini-Browser — Agent Guide

**Windows-only** Go + WebView2 desktop browser with a local HTTP API for headless-like control.

## Build & Run

```bash
go build -o mini-browser.exe
./mini-browser.exe           # starts, shows window, writes port to %TEMP%\mini-browser.port
```

Release build uses `-ldflags="-s -w -H windowsgui"` + UPX `--best`.

## Architecture

- **Single file**: Everything is in `main.go` (~870 lines). No packages, no tests, no CI.
- **WebView2** via `github.com/webview/webview_go` — Edge Chromium embedded window.
- **HTTP API** on `127.0.0.1:<random-port>` (22 endpoints, documented at `GET /api`).
- **JS hooks** (runtime, toolbar, popup blocker) embedded via `//go:embed` and string constants.

## API (most-used endpoints)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/snapshot` | DOM snapshot tree with `uid` per node (keyed by `data-si` attribute) |
| `POST` | `/api/click` | Click element by `{"uid":"s_5"}` or `{"selector":"..."}` |
| `POST` | `/api/fill` | Fill input by `{"uid":"s_5","value":"..."}` or `{"selector":"..."}` |
| `POST` | `/api/eval` | Run arbitrary JS: `{"js":"..."}` |
| `POST` | `/api/navigate` | Navigate: `{"url":"..."}` |
| `GET` | `/api/screenshot` | Base64 PNG of **browser window** (uses PowerShell + Win32 API) |
| `GET` | `/api/network` | Captured fetch/XHR log |
| `GET` | `/api/info` | Browser state (URL, history, port) |

Click/fill accept either `selector` (CSS) or `uid` (data-si). UIDs are per-snapshot.

## Port discovery

The API port is printed to the window title (`Mini Browser [:<port>]`) and written to `$env:TEMP\mini-browser.port`.

## Python inspection scripts

| Script | Purpose |
|--------|---------|
| `check_state.py` | Snapshot + clickable elements + storage + cookies |
| `deepseek_snap.py` | Login state inspection (forms, inputs, buttons, messages) |
| `demo.py` | Prints structured DOM nodes from JSON stdin |

All auto-detect port from `%TEMP%\mini-browser.port`.

## Key constraints

- **Windows only** — uses MinGW-w64 for build, PowerShell + Win32 API for screenshot.
- **Single window** — no headless mode, the window must be visible for WebView2 to render.
- **Turbo loop** adaptive: burst 200ms×10 → 3s steady. Ad cleanup via MutationObserver, not polling.
- **`syncEval` timeout**: 10s default.

## Performance Optimizations (v2.2)

| Optimization | Before | After |
|-------------|--------|-------|
| Turbo loop | `setInterval(ka, 500)` + poll 500ms forever | `MutationObserver` + burst 200ms×10 → 3s |
| Screenshot | Full screen (PowerShell) | Browser window only (GetWindowRect) |
| Snapshot depth | 20 levels | 12 (skip empty node leafs) |
| syncEval timeout | 30s | 10s |
| runtimeJS | ~3KB verbose | ~1.8KB minified |
| Unused code | `getWindowRect`, `findWindow`, `unsafe` imports | Removed |

---

# DeepSeek Browser

A dedicated variant in `deepseek-browser/` that only loads **deepseek.com** domains.

## Build & Run

```bash
cd deepseek-browser
$env:CGO_ENABLED=1; go build -o ../deepseek-browser.exe .
./deepseek-browser.exe       # shows window, port in %TEMP%\deepseek-browser.port
```

Release build: `$env:CGO_ENABLED=1; go build -ldflags="-s -w -H windowsgui" -o ../deepseek-browser.exe .`

## Differences from Mini-Browser

| Aspect | Mini-Browser | DeepSeek Browser |
|--------|-------------|-----------------|
| Start URL | google.com | chat.deepseek.com |
| Navigation | any URL | deepseek.com only (403 otherwise) |
| Toolbar | back/forward/reload + URL bar | back/forward/reload + home button, **no URL bar** |
| Home button | none | navigates to chat.deepseek.com |
| CSP meta | broad | scoped to *.deepseek.com |
| Window title | "Mini Browser [:<port>]" | "DeepSeek Browser [:<port>]" |
| Port file | %TEMP%\mini-browser.port | %TEMP%\deepseek-browser.port |

## API

Same 22 endpoints as mini-browser — `POST /api/navigate` rejects non-deepseek.com URLs with 403.

## DeepSeek Browser Optimizations

Same performance improvements as mini-browser v2.2:
- Adaptive turbo loop (burst → 3s)
- Window-cropped screenshot
- Snapshot depth 20→12
- syncEval timeout 30s→10s
- Removed unused syscall/unsafe code
