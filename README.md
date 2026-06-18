# Hyperspeed Browser

> Ultra-lightweight Windows desktop browser powered by WebView2 with a local HTTP API for automation and headless-like control.

[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go)](https://go.dev)
[![WebView2](https://img.shields.io/badge/WebView2-Edge%20Chromium-4FC3F7?logo=microsoftedge)](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows-blue?logo=windows)](https://github.com/appleghee/Hyperspeed-Browser)
[![Performance](https://img.shields.io/badge/performance-A%2B-66bb6a)](BENCHMARK.md)
[![Size](https://img.shields.io/badge/size-6.9%20MB-4fc3f7)](https://github.com/appleghee/Hyperspeed-Browser/releases)

---

## Features

- **WebView2 engine** — Edge Chromium embedded browser window
- **22 REST API endpoints** — navigate, click, fill, eval JS, screenshot, snapshot DOM
- **Smart optimization engine** — 8 subsystems with 7 performance profiles
- **Real-time metrics** — load time, memory, resource count, performance score
- **Auto-tuning** — adapts to page behavior with rule-based optimization
- **Floating control panel** — toggle lazy images, defer JS, block trackers, caching, custom scripts

## Performance

> Run `python benchmark.py` with the browser open to measure your own results.

| Metric | Value |
|--------|-------|
| Binary size | ~6.9 MB |
| Load time (avg) | ~800 ms |
| DOM ready | ~600 ms |
| Memory usage | ~12 MB |
| Performance score | 95/100 |
| API response | &lt;1 ms |

<img src="https://quickchart.io/chart?c={type:'bar',data:{labels:['Google','GitHub','Wikipedia'],datasets:[{label:'Load time (ms)',data:[823,756,899],backgroundColor:'rgba(79,195,247,0.6)'}]},options:{plugins:{legend:{display:false}}}}" alt="benchmark-chart" width="400">

## Quick Start

```bash
# Build (requires MinGW-w64 with GCC in PATH)
$env:CGO_ENABLED=1
go build -o hyperspeed-browser.exe .

# Run
./hyperspeed-browser.exe
```

The API port is printed in the window title (`Hyperspeed Browser [:<port>]`) and written to `%TEMP%\hyperspeed-browser.port`.

## API Overview

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/snapshot` | DOM tree with unique IDs per node |
| `POST` | `/api/click` | Click element by `uid` or CSS `selector` |
| `POST` | `/api/fill` | Fill input field |
| `POST` | `/api/eval` | Execute arbitrary JavaScript |
| `POST` | `/api/navigate` | Navigate to URL |
| `GET` | `/api/screenshot` | Base64 PNG (window-cropped) |
| `GET` | `/api/info` | Browser state (URL, history, port) |
| `GET` | `/api/network` | Captured fetch/XHR log |
| `GET` | `/api/opt/metrics` | Page performance metrics |
| `POST` | `/api/opt/profile` | Switch optimization profile |
| `POST` | `/api/opt/run` | Run optimization pipeline |

Full API docs at `GET /api` when the browser is running.

## Optimization Profiles

| Profile | Use Case |
|---------|----------|
| **Balanced** | Default — good all-around performance |
| **Turbo** | Maximum speed, minimal snapshot depth |
| **Aggressive** | Heavy caching, low memory budget |
| **Speed** | Fast browsing, moderate caching |
| **Eco** | Low CPU/memory, battery-friendly |
| **Mobile** | Resource-constrained environments |
| **Compat** | Full features, no blocker/defer |

## Project Structure

```
├── main.go              # Core browser + HTTP API + JS constants
├── optimizer.go         # Performance engine (8 subsystems)
├── optimizer-gui.js     # Floating control panel (injected at runtime)
├── popup-blocker.js     # Popup blocker script
├── check_state.py       # Python inspector (auto-detect port)
├── benchmark.py         # Performance benchmark suite
├── BENCHMARK.md         # Benchmark results
```

## Python Scripts

Python tools auto-detect the API port from `%TEMP%\hyperspeed-browser.port`:

```bash
python check_state.py    # Snapshot + clickable elements + storage + cookies
python benchmark.py      # Run real-world performance benchmarks
```

## License

MIT
