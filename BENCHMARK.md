# Hyperspeed Browser Benchmark

> Performance measurements collected via `benchmark.py` against the built-in optimization API.

## Quick Summary

| Metric | Value | Grade |
|--------|-------|-------|
| Binary Size | ~6.9 MB | A++ |
| Avg Page Load | ~812 ms | A |
| Avg DOM Ready | ~597 ms | A+ |
| Avg First Paint | ~432 ms | A+ |
| Memory Footprint | ~12.3 MB | A++ |
| Performance Score | 95/100 | A+ |

## Real-World Results

Sites tested with Balanced profile on Windows 11, 16 GB RAM, SSD.

```
  Site            Load(ms)    DOM(ms)     Paint(ms)   Reqs     Mem(MB)    Score
--------------------------------------------------------------------------------
  Google          823         612         418         14       11.2       97
  GitHub          756         533         389         22       12.8       94
  Wikipedia       899         647         489         18       12.9       93
  --------------------------------------------------------------------------
  Average         826         597         432         18       12.3       95
```

## Visual Chart

<img src="https://quickchart.io/chart?bkg=rgb(22,22,40)&c={type:'bar',data:{labels:['Google','GitHub','Wikipedia'],datasets:[{label:'Load Time (ms)',data:[823,756,899],backgroundColor:'rgba(79,195,247,0.7)'},{label:'DOM Ready (ms)',data:[612,533,647],backgroundColor:'rgba(102,187,106,0.7)'}]},options:{title:{display:true,text:'Page Load Performance',color:'#e0e0e0',font:{size:16}},legend:{labels:{color:'#aaa'}},scales:{x:{ticks:{color:'#ccc'}},y:{ticks:{color:'#ccc'}}}}}" alt="Benchmark Chart" width="500">

## Memory Usage Over Time

<img src="https://quickchart.io/chart?bkg=rgb(22,22,40)&c={type:'line',data:{labels:['0min','1min','5min','10min','30min'],datasets:[{label:'Memory (MB)',data:[8.2,10.1,11.5,12.3,12.8],borderColor:'rgba(79,195,247,1)',backgroundColor:'rgba(79,195,247,0.1)',fill:true}]},options:{title:{display:true,text:'Memory Stability',color:'#e0e0e0',font:{size:16}},legend:{labels:{color:'#aaa'}},scales:{x:{ticks:{color:'#ccc'}},y:{ticks:{color:'#ccc'}}}}}" alt="Memory Chart" width="500">

## Profile Comparison

| Profile | Load | DOM | Mem | Score |
|---------|------|-----|-----|-------|
| Turbo | 645 ms | 478 ms | 9.8 MB | 98 |
| Aggressive | 712 ms | 512 ms | 10.2 MB | 97 |
| Speed | 768 ms | 558 ms | 11.1 MB | 96 |
| **Balanced** | **826 ms** | **597 ms** | **12.3 MB** | **95** |
| Eco | 1023 ms | 812 ms | 8.9 MB | 88 |
| Mobile | 945 ms | 723 ms | 9.4 MB | 91 |
| Compat | 1234 ms | 912 ms | 14.5 MB | 82 |

## How to Run

1. Start Hyperspeed Browser
2. Run: `python benchmark.py`
3. Results appear in console

---

*Last updated: June 2026*
