# Hyperspeed Browser

> Ultra-lightweight Windows desktop browser with **value-centric optimization** — WebView2 + HTTP API + PVDS engine.

[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go)](https://go.dev)
[![WebView2](https://img.shields.io/badge/WebView2-Edge%20Chromium-4FC3F7?logo=microsoftedge)](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows-blue?logo=windows)](https://github.com/appleghee/Hyperspeed-Browser)
[![Release](https://img.shields.io/github/v/release/appleghee/Hyperspeed-Browser?color=4fc3f7)](https://github.com/appleghee/Hyperspeed-Browser/releases)

---

## Features

- **WebView2 engine** — Edge Chromium embedded, lightweight (~6.9 MB binary)
- **PVDS engine** — Predictive Value Density Scheduling: value-centric optimization (not resource-type-based)
- **22+ REST API endpoints** — navigate, click, fill, eval JS, screenshot, snapshot DOM, storage, cookies, hook
- **Smart optimization** — 8 subsystems, 7 profiles, real-time auto-tuning
- **Custom scripts** — inject persistent JS with auto-run on navigation (SPA support)
- **Floating control panel** — toggle optimizations, run scripts, view live stats & PVDS
- **Keyboard shortcuts** — Ctrl+Shift+Space toggle panel, Ctrl+Shift+R run script
- **API auth** — per-launch auto-generated X-API-Token
- **Python tooling** — inspector + benchmark suite

## PVDS: Value-Centric Optimization

Instead of optimizing by resource type (`image` / `css` / `js` / `video`), PVDS optimizes by **actual user value per resource unit consumed**.

### Formula
```
Value Density (VD) = UserVisibleValue / ResourceCost
```

### Pipeline

| Step | Description |
|------|-------------|
| **DOM Graph** | Walks `document.body` (depth ≤ 15), tags every node with `data-vd` attribute |
| **Value Scoring** | Viewport (+50%), interactive (+20), main/article (+25), nav/footer (-10), ad class (-30) |
| **Cost Estimation** | Base 10 + images×20 + iframes×50 + deep subtrees×15 |
| **VD Scheduler** | CPU/memory/network allocated by VD descending (not load order, not viewport-first) |
| **Adaptive Memory** | 200–1200 MB budget (proportional to system RAM); auto-evicts VD < 0.2 when over 80% budget |
| **Dynamic Freeze** | Offscreen iframes blanked, low-VD images hidden, low-VD zones set `pointer-events: none` |
| **CPU Quantum** | High-VD JS tasks get priority execution |

### Score Heuristics

| Signal | Value delta |
|--------|-------------|
| In viewport (>50%) | +30 |
| Viewport partially visible | +10 |
| Area > 50,000 px² | +20 |
| Interactive (button/a/input/select/onclick) | +20 |
| Tag is `main` / `article` / `section` | +25 |
| Tag is `h1` / `h2` / `h3` | +15 |
| Text length > 50 chars | +10 |
| Header | +10 |
| Depth < 4 (close to root) | +10 |
| Ad class/id match | −30 |
| Nav / footer / aside | −10 |

### API

| Endpoint | Description |
|----------|-------------|
| `GET /api/vd/snapshot` | Live VD report: Avg VD, high/low count, memory budget, frozen zones, top 10 nodes |
| `POST /api/vd/optimize` | Run VD scan + eviction + scheduling in one call |

---

## Performance

| Metric | Value |
|--------|-------|
| Binary size | ~6.9 MB |
| Load time (avg) | ~800 ms |
| DOM ready | ~600 ms |
| Memory usage | ~12 MB idle |
| Performance score | 95/100 |
| API response | <1 ms (localhost) |

<img src="https://quickchart.io/chart?c={type:'bar',data:{labels:['Google','GitHub','Wikipedia'],datasets:[{label:'Load time (ms)',data:[823,756,899],backgroundColor:'rgba(79,195,247,0.6)'}]},options:{plugins:{legend:{display:false}}}}" alt="benchmark-chart" width="400">

---

## Quick Start

```powershell
# Build (MinGW-w64 with GCC required)
$env:CGO_ENABLED=1
go build -ldflags="-s -w -H windowsgui" -o hyperspeed-browser.exe .

# Run
./hyperspeed-browser.exe
```

The API port is shown in the window title (`Hyperspeed Browser [:<port>]`) and written to `%TEMP%\hyperspeed-browser.port`.

---

## API Reference

All endpoints require `X-API-Token` header (auto-generated per launch, available at `%TEMP%\hyperspeed-browser.port` line 2).

### Navigation

| Method | Endpoint | Body | Description |
|--------|----------|------|-------------|
| `POST` | `/api/navigate` | `{"url": "https://..."}` | Navigate to URL |
| `POST` | `/api/back` | — | Go back in history |
| `POST` | `/api/forward` | — | Go forward in history |
| `POST` | `/api/reload` | — | Reload current page |

### DOM Interaction

| Method | Endpoint | Body | Description |
|--------|----------|------|-------------|
| `GET` | `/api/snapshot` | — | DOM tree with `uid` per node |
| `POST` | `/api/click` | `{"uid": "..."}` or `{"selector": "..."}` | Click element |
| `POST` | `/api/fill` | `{"uid": "...", "value": "..."}` | Fill input field |

### Scripting

| Method | Endpoint | Body | Description |
|--------|----------|------|-------------|
| `POST` | `/api/eval` | `{"code": "..."}` | Execute arbitrary JavaScript, returns result |
| `GET` | `/api/runtime` | — | Get runtime JS context info |
| `GET` | `/api/scripts` | — | Get loaded scripts list |

### Network

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/network` | Capture fetch / XHR / WebSocket log |
| `GET` | `/api/ws` | WebSocket messages |
| `GET` | `/api/sse` | Server-Sent Events stream |

### State

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/info` | URL, history length, port, profile, connected |
| `GET` | `/api/source` | Current page HTML source |
| `GET` | `/api/screenshot` | Base64 PNG screenshot (window-cropped) |
| `GET` | `/api/storage` | localStorage + sessionStorage keys |
| `GET` | `/api/cookies` | All cookies (name, value, domain, path, expiry) |

### Hooks & Instrumentation

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/hook` | Install fetch/XHR/WebSocket interceptors |
| `POST` | `/api/unhook` | Restore original fetch/XHR/WebSocket |

### Optimization

| Method | Endpoint | Body | Description |
|--------|----------|------|-------------|
| `GET` | `/api/opt` | — | Optimizer status + active profile |
| `GET` | `/api/opt/metrics` | — | Performance score, load time, request count |
| `POST` | `/api/opt/profile` | `{"profile": "turbo"}` | Switch profile (7 available) |
| `POST` | `/api/opt/run` | — | Run full optimization pipeline |
| `POST` | `/api/opt/tune` | — | Auto-tune parameters |

### PVDS (v2.0+)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/vd/snapshot` | Live VD report |
| `POST` | `/api/vd/optimize` | Run VD scan + eviction + freeze |

### Root

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api` | Full API documentation (returned as JSON schema) |

---

## Optimization Profiles

| Profile | LazyImages | DeferCSS | DeferJS | BlockTrackers | Cache | AutoTune | Use Case |
|---------|:----------:|:--------:|:-------:|:-------------:|:-----:|:--------:|----------|
| **Balanced** | ✓ | ✓ | ✓ | ✓ | 200 entries | ✓ | Default — good all-around |
| **Turbo** | ✓ | ✓ | ✓ | ✓ | 500 entries | ✓ | Maximum speed, aggressive caching |
| **Aggressive** | ✓ | ✓ | ✓ | ✓ | 1000 entries | ✓ | Heavy optimization, low memory budget |
| **Speed** | ✓ | ✓ | ✓ | ✓ | 500 entries | ✓ | Fast browsing, moderate resources |
| **Eco** | ✓ | ✓ | ✓ | ✓ | 50 entries | — | Battery-friendly, low CPU/memory |
| **Mobile** | ✓ | ✓ | ✓ | ✓ | 100 entries | — | Resource-constrained environments |
| **Compat** | — | — | — | — | 100 entries | — | Full features, no blockers |

---

## Optimizer Engine: 8 Subsystems

```
┌─────────────────────────────────────────────────────┐
│                   Optimizer                          │
├─────────┬─────────┬──────────┬──────────┬───────────┤
│ Metrics │ Resource│ CSS/JS   │  Media   │  Network  │
│Collector│Classifier│Optimizer│Optimizer │   Queue   │
├─────────┴─────────┴──────────┴──────────┼───────────┤
│           Smart Cache                    │ AutoTuner│
├─────────────────────────────────────────┴───────────┤
│              Value Density Engine (PVDS)             │
└─────────────────────────────────────────────────────┘
```

### Subsystems
- **MetricsCollector** — load time, DOM ready, resource count, memory, performance score
- **ResourceClassifier** — classifies scripts/styles/media by criticality
- **CSSJSOptimizer** — defers non-critical CSS/JS, inlines critical path
- **MediaOptimizer** — lazy loading, image dimension hints, responsive attributes
- **NetworkQueue** — concurrent request throttling, domain prioritization
- **SmartCache** — TTL-based cache with LRU eviction, preload hints
- **AutoTuner** — rule-based parameter adjustment from observed page behavior
- **ValueDensityEngine (PVDS)** — DOM graph analysis, VD scoring, adaptive eviction

---

## Custom Scripts

The floating control panel includes a **Custom Script** section:

| Feature | Description |
|---------|-------------|
| **Textarea** | Paste any JavaScript |
| **Save** | Persists to `localStorage.__mb_customScript` (survives page loads) |
| **Auto toggle** | Injects script on every page load (including SPA pushState/popstate) |
| **Ctrl+Shift+R** | Run script immediately |
| **Panel toggle** | Ctrl+Shift+Space |

Auto-inject works via:
- `DOMContentLoaded` listener
- `history.pushState` / `history.replaceState` monkey-patch
- `popstate` event listener (500 ms delay for SPA render)

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+Shift+Space` | Toggle optimizer panel |
| `Ctrl+Shift+R` | Run custom script |

---

## Project Structure

```
├── main.go                 # Core browser, HTTP API, auth, JS injection constants
├── optimizer.go            # Optimization engine (8 subsystems, 7 profiles)
├── value_density.go        # PVDS engine (DOM graph, VD scoring, eviction, freeze)
├── optimizer-gui.js        # Floating control panel (injected at runtime)
├── popup-blocker.js        # Popup blocker script (injected at runtime)
├── check_state.py          # Python inspector — snapshot, cookies, storage, clickable elements
├── benchmark.py            # Performance benchmark suite (real-world sites)
├── hyperspeed-browser.exe  # Built binary
├── icon.ico / icon.rc / icon.syso  # App icon (16–256px, embedded via windres)
├── README.md
└── LICENSE                 # MIT
```

---

## Python Tooling

Python scripts auto-detect the API port + auth token from `%TEMP%\hyperspeed-browser.port`:

```bash
# Full page inspection
python check_state.py
# → DOM snapshot, cookies, localStorage, sessionStorage, clickable elements

# Performance benchmarks
python benchmark.py
# → Load time, DOM ready, memory, request count, performance score
```

---

## Security

- **Per-launch API token**: 32-byte random hex string generated on every launch
- All API endpoints validate `X-API-Token` header
- Token passed to JS via `window.__mbToken` and written to port file
- Default profile is **safe** (no lazy-loading, no deferring, no tracking removal, no auto-tune)
- User must explicitly switch to turbo/aggressive/speed/eco/mobile for enhanced features

---

## Building from Source

### Requirements
- [Go 1.21+](https://go.dev/dl/)
- [MinGW-w64](https://www.mingw-w64.org/) (with GCC for CGO)
- [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (included with Windows 11 / Edge)

### Build
```powershell
$env:CGO_ENABLED=1
$env:Path = "C:\mingw64\bin;$env:Path"
go build -ldflags="-s -w -H windowsgui" -o hyperspeed-browser.exe .
```

---

## License

MIT
