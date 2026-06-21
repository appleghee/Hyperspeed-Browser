# AGENTS.md — Hyperspeed Browser v3.2.0 Genesis

## Build & Run

```powershell
# Release build (recommended)
.\build.ps1

# Development build (console window, no stripping)
.\build.ps1 -Mode dev
```

Windows-only. Requires **MinGW-w64** (`C:\mingw64\bin\gcc`) + **WebView2 Runtime** (Windows 11 includes it).

No tests, no linter configured in repo. `go vet` and `go build` are the verification steps.

## Architecture

Single package `main`, single binary. `main.go` + engine files (`*.go`, same package).

Data flow: `main()` → `browser{WebView}` → bind Go↔JS (`w.Bind`) → start HTTP API goroutine → `w.Init(...)` (bootstrap JS) → engine starts → turbo post-load injection → `w.Run()`.

## JS Injection Conventions

Bootstrap JS merged into one `w.Init(...)` call at `main.go:218`. Inline `const` strings are the convention.

Post-load JS goes through `b.w.Dispatch(func() { b.w.Eval(...) })` (see `injectTurboLoop` at `main.go:341`). Guard with `window.__mbXxx` flags.

## Sync Eval

- `syncEval(js, timeout)` → `__evalCb(id, JSON.stringify(result))` → callback. Only safe way.
- `syncExec(js)` = fire-and-forget.
- `syncUnwrap(js, timeout)` = eval + JSON-unmarshal.
- `syncUnwrapInto(js, timeout, &target)` = eval + unmarshal directly into target struct.

## Runtime Server

- `127.0.0.1:0` random port. CORS `*`.
- Auth: `X-API-Token` header (32-byte hex).
- Port file: `%TEMP%\hyperspeed-browser.port` (line 1 = port, line 2 = token).

## Engine Architecture (v3.2 Genesis)

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Hyperspeed Browser v3.2 Genesis                 │
├────────────────┬───────────┬──────────┬──────────┬─────────────────┤
│ 19 Core        │ 13 Genesis│ IO       │ Runtime  │ Infrastructure  │
│ Engines        │ Engines   │ Cascade  │ Core     │                 │
├────────────────┼───────────┼──────────┼──────────┼─────────────────┤
│ PVDS, CRG,     │ DNA, HBM, │ LOD1-2   │ UHE,     │ AutoTune,       │
│ EHS, QSE,      │ AVP, NCG, │ content- │ HLRC,    │ AdaptiveGC,     │
│ 5×QuickOpt,    │ DOM Comp, │ visibi-  │ NDF,     │ SmartCache,     │
│ RHD-GC, PVC,   │ PCE, UPM, │ lity:auto│ RPC,     │ NetworkQueue    │
│ RPC, LOD       │ DRA, MCS, │          │ PVDS     │                 │
│                 │ CBL, UEE, │          │          │                 │
│                 │ HFS, RCM  │          │          │                 │
└────────────────┴───────────┴──────────┴──────────┴─────────────────┘
```

## Key Structural Gotchas

- `Optimizer` struct owns 32 engine sub-objects. Adding a new engine requires wiring into `NewOptimizer` + registering API routes in `startAPI()` + adding Start() calls in `main()`.
- `b.opt` is nil-guarded in every handler.
- History and browse history protected by `b.mu`; eval dispatch runs on WebView thread, must not block.
- All 13 Genesis engines use `//go:embed` for their respective `.js` files. Create the JS file and wire into the Go engine struct, then register routes.

## Genesis Engines

| Engine | File | API | Description |
|--------|------|-----|-------------|
| DNA | dna.go | `/api/dna/*` | Page DNA — per-site behavioral fingerprint |
| HBM | hbm.go | `/api/hbm/stats` | Heat-Based Memory allocator |
| AVP | avp.go | `/api/avp/stats` | Adaptive Viewport Predictor |
| DOM Compress | dom_compress.go | `/api/domcompress/stats` | Binary DOM transport |
| NCG | ncg.go | `/api/ncg/stats` | Network Cost Graph |
| PCE | pce.go | `/api/pce/stats` | Page Change Engine |
| UPM | upm.go | `/api/upm/stats` | User Presence Model |
| DRA | extra_engines.go | `/api/dra/stats` | Dynamic Resource Adjustment |
| MCS | extra_engines.go | `/api/mcs/stats` | Micro-Controller Scheduler |
| CBL | extra_engines.go | `/api/cbl/stats` | Content-Based Loading |
| UEE | extra_engines.go | `/api/uee/stats` | Unified Event Engine |
| HFS | extra_engines.go | `/api/hfs/stats` | Heat-File System |
| RCM | extra_engines.go | `/api/rcm/stats` | Resource Cost Model |
