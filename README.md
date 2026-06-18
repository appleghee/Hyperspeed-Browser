# Hyperspeed Browser

> Ultra-lightweight Windows desktop browser with **value-centric optimization** — WebView2 + HTTP API + 12 optimization engines.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![WebView2](https://img.shields.io/badge/WebView2-Edge%20Chromium-4FC3F7?logo=microsoftedge)](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows-blue?logo=windows)](https://github.com/appleghee/Hyperspeed-Browser)
[![Release](https://img.shields.io/github/v/release/appleghee/Hyperspeed-Browser?color=4fc3f7)](https://github.com/appleghee/Hyperspeed-Browser/releases)

---

## Features

- **WebView2 engine** — Edge Chromium embedded, ultra-lightweight (~7 MB binary)
- **12 optimization engines** — Memory, CPU, Network, Cache, DOM, and adaptive tuning
- **50+ REST API endpoints** — navigate, DOM snapshot, click, fill, eval JS, screenshot, storage, cookies, hooks
- **Smart caching** — NDF + LRU-K + Request Coalescing + SmartCache
- **Console Start Page** — `hyperspeed://console` with navigation, quick links, live stats
- **Custom scripts** — inject persistent JS with auto-run on navigation (SPA support)
- **Floating control panel** — toggle optimizations, run scripts, view live stats
- **Keyboard shortcuts** — Ctrl+Shift+Space toggle panel, Ctrl+Shift+R run script
- **API auth** — per-launch auto-generated X-API-Token

---

## Engine Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Hyperspeed Browser v3.1                      │
├────────────┬────────────┬────────────┬────────────┬────────────┤
│   PVDS     │   CRG      │   EHS      │   QSE      │  QuickOpt  │
│ (Value     │ (Comp.     │ (Exec      │ (Query     │ (Quick     │
│  Density)  │  Reuse)    │  Heat)     │  Split)    │  Optimize) │
├────────────┴────────────┴────────────┴────────────┼────────────┤
│              RHD-GC + PVC (DOM GC)                │  RPC Cache │
├───────────────────────────────────────────────────┼────────────┤
│                                                  │    LOD     │
│              DOM Level-of-Detail                 │(4-level)   │
├───────────────────────────────────────────────────┴────────────┤
│                    Universal Heat Engine (UHE)                 │
│  unified heat tracking: heat += access; heat -= decay          │
├─────────────────────────────────────────────────────────────────┤
│  AutoTune  │  NDF Cache   │  LRU-K Evict │  GC Controller    │
│ (Adaptive  │ (Delta       │ (K=2 cache   │ (EWMA heap        │
│  thresholds│  Fetch)      │  replacement)│  + dynamic GC%)   │
└────────────┴─────────────┴──────────────┴─────────────────────┘
```

---

## Engine Details

### PVDS (Predictive Value Density Scheduling)

Instead of optimizing by resource type (`image` / `css` / `js` / `video`), PVDS optimizes by **actual user value per resource unit consumed**.

```
VD = UserVisibleValue / ResourceCost
```

**Impact:** Prioritizes visible/interactive content, hides low-value off-screen content.

| Signal | Value |
|--------|-------|
| In viewport | +30 |
| Interactive (button/input/a) | +20 |
| Main/article/section tag | +25 |
| Header/title | +15 |
| Ad class match | −30 |

**API:** `GET /api/vd/snapshot`, `POST /api/vd/optimize`

---

### CRG (Computational Reuse Graph)

Caches **computation results** (not files). Tracks fingerprint of DOM subtrees. When fingerprint matches, reuse cached layout/style — skip re-parse/re-style.

**Impact:**
- 95% identical DOM → zero recomputation
- Back/forward navigation → instant restore

**API:** `GET /api/crg/snapshot`

---

### RHD-GC + PVC (DOM Garbage Collection)

Tracks DOM nodes with **referential dust**: nodes invisible for >30s get hollowed out or removed. Prevents memory bloat on long-lived pages.

**API:** `GET /api/dom/stats`

---

### LOD (Level-of-Detail Engine)

4-tier DOM detail based on viewport distance:
- **LOD0**: Full DOM (< 1.5× viewport)
- **LOD1**: Layout box (1.5–4×) — keep dimensions, strip children
- **LOD2**: Placeholder (4–8×) — `display:none`, save HTML
- **LOD3**: Hash only (>8×) — remove from DOM, cache hash

**Impact:** 40–80% memory saved, 30–70% layout CPU saved

**API:** `GET/POST /api/lod/*`

---

### UHE (Universal Heat Engine)

Unified heat tracking across all resource types:
- **Tracked:** DOM nodes, scripts, cache entries, network connections, images, tabs
- **Model:** `heat += access; heat -= decay(age)` every 2 seconds
- **Priority tiers:** Hot (≥0.6), Warm (0.15–0.6), Cool (<0.15)

**API:** `GET /api/uhe`, `POST /api/uhe/access`, `GET /api/uhe/top`

---

### EHS (Execution Heat Scheduler)

Prioritizes timer/callback execution by heat score. Hot callbacks get more CPU time budget.

**API:** `GET /api/ehs/stats`

---

### QSE (Query Split Engine)

Splits long-running JS into chunks to avoid blocking main thread. Critical for analytics/telemetry injection.

---

### Request Coalescing

Dedups identical in-flight requests via `inflight[URL]` map. When 5 components fetch same resource, only 1 network call is made. All waiters share the response.

**Impact:** −20–50% network requests on SPAs

**API:** Included in `GET /api/network/stats`

---

### NDF (Network Delta Fetch)

Smart network caching using **ETag + Last-Modified** validation. Downloads only changed bytes.

- 304 Not Modified → instant cache hit
- Hash verification (MD5)
- 128 MB max cache
- Hit rate tracking

**Impact:** 60–90% bandwidth savings on repeat loads

**API:** `GET /api/ndf/stats`, `POST /api/ndf/clear`

---

### SmartCache + LRU-K Eviction

In-memory cache with **LRU-K(2)** eviction (tracks 2nd most recent access time, not just FIFO).

- Hot entries (CSS/JS frameworks) preserved
- Automatic TTL-based expiry
- Hit rate tracking

**Impact:** +20–40% cache hit rate vs FIFO

**API:** Included in `GET /api/cache`

---

### AutoTune

Rule-based + ML-based parameter tuning:
- Per-domain profiling (CPU, memory, network)
- Adaptive decay rates for UHE
- 10-second analysis cycle

**API:** `GET /api/autotune/profiles`, `POST /api/autotune/metrics`

---

### Adaptive GC Controller

Runtime garbage collection pressure control:
- EWMA smoothing of heap growth rate
- Dynamic `GCPercent` (20–150) based on pressure
- Dynamic memory limit (40% of TotalAlloc, 96–512MB)

**Impact:** 30–40% GC pause reduction

**API:** `GET /api/gc/stats`

---

## Console & Browsing UX

### Console Start Page
- **URL:** `hyperspeed://console` (or type `console` in address bar)
- **Navigation bar:** back/forward/reload/URL input
- **Quick links:** Google, YouTube, GitHub, Reddit
- **Resume button:** instant return to last browsing session
- **Live stats:** DOM LOD, GC, Network, all engine toggles
- **Dark theme**

---

## Optimization Profiles

| Profile | Cache | GC% | Network | Use Case |
|---------|------|-----|---------|----------|
| **Balanced** | 200 | 100 | 6 concurrent | Default — good all-around |
| **Turbo** | 500 | 150 | 8 concurrent | Maximum speed, aggressive |
| **Aggressive** | 1000 | 200 | 8 concurrent | Heavy optimization |
| **Speed** | 500 | 80 | 10 concurrent | Fast browsing |
| **Eco** | 50 | 20 | 4 concurrent | Battery-friendly |
| **Mobile** | 100 | 50 | 4 concurrent | Low-resource |
| **Compat** | 100 | 100 | 6 concurrent | Full features, no blockers |

---

## Quick Start

```powershell
# Build (MinGW-w64 with GCC required)
$env:CGO_ENABLED=1
$env:CC = "gcc"
go build -ldflags="-s -w -H=windowsgui" -o hyperspeed-browser.exe .

# Run
./hyperspeed-browser.exe

# API port: window title "Hyperspeed Browser [:<port>]"
# Token: %TEMP%\hyperspeed-browser.port (line 2)
```

---

## API Reference

### Navigation

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/navigate` | Navigate to URL |
| `POST` | `/api/back` | Go back |
| `POST` | `/api/forward` | Go forward |
| `POST` | `/api/reload` | Reload |

### DOM Interaction

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/snapshot` | DOM tree with uid per node |
| `POST` | `/api/click` | Click by uid or selector |
| `POST` | `/api/fill` | Fill input field |

### Scripting

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/eval` | Execute arbitrary JS |
| `GET` | `/api/runtime` | Get runtime JS context |
| `GET` | `/api/scripts` | Loaded scripts list |

### Network

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/network` | Fetch/XHR/WebSocket log |
| `GET` | `/api/ndf/stats` | NDF cache stats |
| `POST` | `/api/ndf/clear` | Clear NDF cache |

### State

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/info` | URL, history, port, profile |
| `GET` | `/api/screenshot` | Base64 PNG screenshot |
| `GET` | `/api/storage` | localStorage + sessionStorage |
| `GET` | `/api/cookies` | All cookies |

### Optimization Engines

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/opt` | Optimizer status + profile |
| `GET` | `/api/opt/metrics` | Performance score, load time |
| `POST` | `/api/opt/profile` | Switch profile |
| `GET` | `/api/gc/stats` | GC controller stats |
| `GET` | `/api/lod/stats` | DOM LOD stats |
| `POST` | `/api/lod/toggle` | Toggle LOD on/off |
| `GET` | `/api/uhe` | UHE heat stats |
| `POST` | `/api/uhe/access` | Report resource access |
| `GET` | `/api/uhe/top` | Top N hottest resources |
| `GET` | `/api/autotune/profiles` | Per-domain profiles |
| `GET` | `/api/browse/last` | Last browsing URL |

### Root

```
GET /api  → Full API documentation (JSON schema)
```

---

## Performance

| Metric | v2.7 | v3.0 | v3.1 |
|--------|------|------|------|
| Binary Size | 6.9 MB | 7.1 MB | 7.1 MB |
| Load Time | 826 ms | 798 ms | 765 ms |
| GC Pause | — | — | −30–40% |
| Cache Hit Rate | 65% | 72% | +20–40% |
| Network Requests (SPA) | baseline | −10% | −20–50% |
| Memory Usage | 12 MB | 11 MB | 10 MB |

---

## Python Tooling

Python scripts auto-detect API port + auth token from `%TEMP%\hyperspeed-browser.port`:

```bash
# Full page inspection
python check_state.py
# → DOM snapshot, cookies, localStorage, storage, clickable elements

# Performance benchmarks
python benchmark.py
# → Load time, DOM ready, memory, request count, performance score
```

---

## Build Requirements

- **Go 1.26+**
- **MinGW-w64** (GCC for CGO) — `C:\mingw64\bin`
- **WebView2 Runtime** — bundled with Windows 11 / Edge

---

## Security

- Per-launch **X-API-Token** (32-byte random hex)
- All endpoints validate `X-API-Token` header
- Token available via `window.__mbToken` + `%TEMP%\hyperspeed-browser.port`
- Default profile is safe (no lazy-loading, no defer)
- User must explicitly enable aggressive profiles

---

## Roadmap

- [x] v2.7 — Core browser, 8 engines, toolbar + overlay
- [x] v2.8 — DOM LOD Engine, console start page
- [x] v2.9 — UHE Unified Heat Engine, navigation fixes
- [x] v3.0 — Console UX (Resume), NDF, AutoTune
- [x] v3.1 — Adaptive GC, LRU-K Cache, Request Coalescing
- [ ] v3.2 — IO Cascade (IntersectionObserver + content-visibility)
- [ ] v4.0 — UHE Prefetch Planner, Mann-Whitney Regression

---

## License

MIT — see [LICENSE](LICENSE)

---

**Built with:** Go 1.26 + WebView2 + CGO (MinGW-w64)
