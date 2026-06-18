# Hyperspeed Browser Benchmark

> Performance data across versions v2.7 → v3.1. Measured with `benchmark.py` on Windows 11, 16 GB RAM, SSD, Balanced profile unless noted.

## Version Comparison

| Metric | v2.7 | v3.0 | v3.1 | Trend |
|--------|------|------|------|-------|
| Binary Size | 6.9 MB | 7.1 MB | 7.1 MB | Stable |
| Avg Load Time | 826 ms | 798 ms | 765 ms | −7.4% |
| Avg DOM Ready | 597 ms | 578 ms | 554 ms | −7.2% |
| Avg First Paint | 432 ms | 418 ms | 399 ms | −7.6% |
| Memory (idle) | 12.3 MB | 11.0 MB | 10.2 MB | −17% |
| GC Pause | — | — | −30–40% | Improved |
| Cache Hit Rate | 65% | 72% | +20–40% | Improved |
| Network Requests (SPA) | baseline | −10% | −20–50% | Improved |
| Performance Score | 95/100 | 96/100 | 97/100 | +2.1% |

## Real-World Site Results (v3.1)

```
  Site            Load(ms)    DOM(ms)     Paint(ms)   Reqs     Mem(MB)    Score
  -------------------------------------------------------------------------------
  Google          765         558         398         12       9.8        98
  GitHub          712         521         376         18       10.5       96
  Wikipedia       798         583         423         16       10.2       95
  -------------------------------------------------------------------------------
  Average         758         554         399         15       10.2       97
```

## Visual Charts

### Page Load Performance
<img src="https://quickchart.io/chart?bkg=rgb(22,22,40)&c={type:'bar',data:{labels:['Google','GitHub','Wikipedia'],datasets:[{label:'Load Time (ms)',data:[765,712,798],backgroundColor:'rgba(79,195,247,0.7)'},{label:'DOM Ready (ms)',data:[558,521,583],backgroundColor:'rgba(102,187,106,0.7)'}]},options:{title:{display:true,text:'Page Load Performance v3.1',color:'#e0e0e0',font:{size:16}},legend:{labels:{color:'#aaa'}},scales:{x:{ticks:{color:'#ccc'}},y:{ticks:{color:'#ccc'}}}}}" alt="Load Chart" width="500">

### Memory Stability
<img src="https://quickchart.io/chart?bkg=rgb(22,22,40)&c={type:'line',data:{labels:['0min','1min','5min','10min','30min'],datasets:[{label:'Memory (MB)',data:[8.2,9.5,10.0,10.2,10.2],borderColor:'rgba(79,195,247,1)',backgroundColor:'rgba(79,195,247,0.1)',fill:true}]},options:{title:{display:true,text:'Memory Stability v3.1',color:'#e0e0e0',font:{size:16}},legend:{labels:{color:'#aaa'}},scales:{x:{ticks:{color:'#ccc'}},y:{ticks:{color:'#ccc'}}}}}" alt="Memory Chart" width="500">

## Profile Comparison (v3.1)

| Profile | Load | DOM | Mem | Score |
|---------|------|-----|-----|-------|
| Turbo | 612 ms | 445 ms | 8.8 MB | 99 |
| Aggressive | 678 ms | 489 ms | 9.2 MB | 98 |
| Speed | 723 ms | 534 ms | 9.8 MB | 97 |
| **Balanced** | **765 ms** | **554 ms** | **10.2 MB** | **97** |
| Eco | 945 ms | 756 ms | 8.2 MB | 91 |
| Mobile | 878 ms | 689 ms | 8.7 MB | 93 |
| Compat | 1123 ms | 856 ms | 12.1 MB | 85 |

## Key Improvements by Version

- **v2.8:** DOM LOD → −40–80% memory on heavy sites
- **v3.0:** NDF → −60–90% bandwidth on repeat loads
- **v3.1:** Adaptive GC → −30–40% GC pause + LRU-K → +20–40% cache hit

## How to Run

```bash
# Start browser, then:
python benchmark.py

# Results appear in console with full breakdown
```

---

*Last updated: June 2026*
