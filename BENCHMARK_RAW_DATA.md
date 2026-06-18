# Raw Benchmark Data

## Binary Sizes

| File | Size (bytes) | Size (KB) | Notes |
|---|---|---|---|
| `mini-browser.exe` | 986,112 | 963 | UPX `--best`, release build |
| `mini-browser-debug.exe` | 2,585,600 | 2,525 | No UPX, debug build |
| `mini-browser-bench.exe` | 2,711,040 | 2,648 | No UPX, bench-instrumented |
| `mini-browser-bench-upx.exe` | 1,033,728 | 1,010 | UPX `--best` of bench binary |

## Cold Start Timings (from instrumented binary)

### Run 1 (first ever — WebView2 truly cold)
```
Process start → webview.New():       2190 ms
Before Run():                         2273 ms
First toolbar (startToWindow):        4334 ms  ← inflated by about:blank issue
```

### Run 2 (WebView2 partially cached)
```
Process start → webview.New():        919 ms
Before Run():                          935 ms
First toolbar (startToWindow):        1098 ms
  → toolbarInjectJS:                    8.7 ms
  → getNavStateJS:                     54.4 ms
  → Memory (WS/Priv):              19920/15748 KB
```

### Run 3 (after killing all WebView2 processes)
```
Process start → webview.New():       1024 ms
Before Run():                         1032 ms
First toolbar (startToWindow):        1214 ms
  → toolbarInjectJS:                    7.1 ms
  → getNavStateJS:                     65.6 ms
  → Memory (WS/Priv):              19440/15808 KB
```

## Memory Samples (from PowerShell Get-Process)

### Cold run (PID 7248, start 11:58:34)
| Label | Time from start | WorkingSet (KB) | Private (KB) |
|---|---|---|---|
| startup(3s) | ~3s | 19,964 | 15,772 |
| example.com+toolbar(6s) | ~6s | 20,024 | 15,800 |
| google.com(10s) | ~10s | 20,208 | 15,956 |
| github.com(15s) | ~15s | 20,208 | 15,956 |
| example.com(20s) | ~20s | 20,208 | 15,956 |

### Cold run (PID 6520, start 11:59:47)
| Label | Time from start | WorkingSet (KB) | Private (KB) |
|---|---|---|---|
| startup(3s) | ~3s | 19,440 | 15,808 |
| example.com+toolbar(6s) | ~6s | 19,704 | 16,052 |
| google.com(10s) | ~10s | 19,728 | 16,072 |
| github.com(15s) | ~15s | — | — |
| example.com(20s) | ~20s | — | — |

## Toolbar Injection Overhead (JS `performance.now()`)

### Per page load
| Page | Inject (ms) | getNavState (ms) | Notes |
|---|---|---|---|
| example.com (first) | 8.7 | 54.4 | Cold start |
| google.com | 194.2 | 32.4 | Heavy page |
| github.com | 1052.4 | 51.8 | Heavy page |
| example.com (again) | 4.3 | 59.2 | Simple page, cached |
| example.com | 7.1 | 65.6 | Cold start v3 |
| google.com | 303.0 | 41.2 | Heavy page |
| example.com | 5.8 | 59.2 | Simple page |

## Bound Function Call Latency (getNavState)

| Run # | Page | Latency (ms) |
|---|---|---|
| 1 | example.com | 54.4 |
| 1 | google.com | 32.4 |
| 1 | github.com | 51.8 |
| 1 | example.com (2nd) | 59.2 |
| 2 | example.com | 65.6 |
| 2 | google.com | 41.2 |
| 2 | example.com (2nd) | 59.2 |
| **Min** | | **32.4** |
| **Max** | | **65.6** |
| **Average** | | **52.0** |

## Navigation Latency (Go Dispatch → Navigate)

| Run | Latency (ms) |
|---|---|
| Navigate to google.com | 0.000 |
| Navigate to github.com | 0.000 |
| Navigate to example.com | 0.504 |
| Various dispatches | 0.000–3.192 |
| **Typical** | **~0.5** |
