package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Optimizer - unified optimization engine
// =============================================================================

type Optimizer struct {
	b *browser

	mu      sync.RWMutex
	enabled bool
	profile OptimizerProfile

	metrics    *MetricsCollector
	csso       *CSSJSOptimizer
	media      *MediaOptimizer
	netq       *NetworkQueue
	cache      *SmartCache
	tuner      *AutoTuner
	vd         *ValueDensityEngine
	crg        *CRGEngine
	quick      *QuickOptEngine
	rhd        *RHDGCEngine
	pvc        *PVCEngine
	ehs        *EHSEngine
	rpc        *RPCEngine
	qse        *QSEEngine
	lod        *LODEngine
}

type OptimizerProfile struct {
	Name                 string `json:"name"`
	SnapshotDepth        int    `json:"snapshotDepth"`
	TurboBurstCount      int    `json:"turboBurstCount"`
	TurboSlowIntervalMs  int    `json:"turboSlowIntervalMs"`
	EvalTimeoutMs        int    `json:"evalTimeoutMs"`
	CacheMaxEntries      int    `json:"cacheMaxEntries"`
	CacheTTLMs           int    `json:"cacheTTLMs"`
	NetworkMaxConcurrent int    `json:"networkMaxConcurrent"`
	LazyLoadImages       bool   `json:"lazyLoadImages"`
	DeferNonCriticalCSS  bool   `json:"deferNonCriticalCSS"`
	DeferNonCriticalJS   bool   `json:"deferNonCriticalJS"`
	RemoveTracking       bool   `json:"removeTracking"`
	AutoTuneEnabled      bool   `json:"autoTuneEnabled"`
}

var defaultProfile = OptimizerProfile{
	Name:                 "balanced",
	SnapshotDepth:        12,
	TurboBurstCount:      8,
	TurboSlowIntervalMs:  4000,
	EvalTimeoutMs:        10000,
	CacheMaxEntries:      250,
	CacheTTLMs:           90000,
	NetworkMaxConcurrent: 6,
	LazyLoadImages:       true,
	DeferNonCriticalCSS:  true,
	DeferNonCriticalJS:   true,
	RemoveTracking:       true,
	AutoTuneEnabled:      true,
}

var speedProfile = OptimizerProfile{
	Name:                 "speed",
	SnapshotDepth:        6,
	TurboBurstCount:      4,
	TurboSlowIntervalMs:  6000,
	EvalTimeoutMs:        4000,
	CacheMaxEntries:      800,
	CacheTTLMs:           300000,
	NetworkMaxConcurrent: 3,
	LazyLoadImages:       true,
	DeferNonCriticalCSS:  true,
	DeferNonCriticalJS:   true,
	RemoveTracking:       true,
	AutoTuneEnabled:      true,
}

var compatProfile = OptimizerProfile{
	Name:                 "compat",
	SnapshotDepth:        15,
	TurboBurstCount:      15,
	TurboSlowIntervalMs:  2000,
	EvalTimeoutMs:        15000,
	CacheMaxEntries:      100,
	CacheTTLMs:           30000,
	NetworkMaxConcurrent: 8,
	LazyLoadImages:       false,
	DeferNonCriticalCSS:  false,
	DeferNonCriticalJS:   false,
	RemoveTracking:       false,
	AutoTuneEnabled:      false,
}

var ecoProfile = OptimizerProfile{
	Name:                 "eco",
	SnapshotDepth:        10,
	TurboBurstCount:      8,
	TurboSlowIntervalMs:  5000,
	EvalTimeoutMs:        15000,
	CacheMaxEntries:      300,
	CacheTTLMs:           120000,
	NetworkMaxConcurrent: 3,
	LazyLoadImages:       true,
	DeferNonCriticalCSS:  true,
	DeferNonCriticalJS:   true,
	RemoveTracking:       true,
	AutoTuneEnabled:      true,
}

var aggressiveProfile = OptimizerProfile{
	Name:                 "aggressive",
	SnapshotDepth:        5,
	TurboBurstCount:      3,
	TurboSlowIntervalMs:  8000,
	EvalTimeoutMs:        3000,
	CacheMaxEntries:      1000,
	CacheTTLMs:           600000,
	NetworkMaxConcurrent: 2,
	LazyLoadImages:       true,
	DeferNonCriticalCSS:  true,
	DeferNonCriticalJS:   true,
	RemoveTracking:       true,
	AutoTuneEnabled:      true,
}

var turboProfile = OptimizerProfile{
	Name:                 "turbo",
	SnapshotDepth:        4,
	TurboBurstCount:      2,
	TurboSlowIntervalMs:  10000,
	EvalTimeoutMs:        2000,
	CacheMaxEntries:      1500,
	CacheTTLMs:           900000,
	NetworkMaxConcurrent: 1,
	LazyLoadImages:       true,
	DeferNonCriticalCSS:  true,
	DeferNonCriticalJS:   true,
	RemoveTracking:       true,
	AutoTuneEnabled:      true,
}

var mobileProfile = OptimizerProfile{
	Name:                 "mobile",
	SnapshotDepth:        8,
	TurboBurstCount:      5,
	TurboSlowIntervalMs:  4000,
	EvalTimeoutMs:        8000,
	CacheMaxEntries:      400,
	CacheTTLMs:           180000,
	NetworkMaxConcurrent: 4,
	LazyLoadImages:       true,
	DeferNonCriticalCSS:  true,
	DeferNonCriticalJS:   true,
	RemoveTracking:       true,
	AutoTuneEnabled:      true,
}

func NewOptimizer(b *browser) *Optimizer {
	o := &Optimizer{
		b:          b,
		enabled:    true,
		profile:    defaultProfile,
		metrics:    NewMetricsCollector(b),
		csso:       NewCSSJSOptimizer(b),
		media:      NewMediaOptimizer(b),
		netq:       NewNetworkQueue(b),
		cache:      NewSmartCache(defaultProfile.CacheMaxEntries, defaultProfile.CacheTTLMs),
		tuner:      NewAutoTuner(b),
		vd:         NewValueDensityEngine(b),
		crg:        NewCRGEngine(b),
		quick:      NewQuickOptEngine(b),
		rhd:        NewRHDGCEngine(b),
		pvc:        NewPVCEngine(b),
		ehs:        NewEHSEngine(b),
		rpc:        NewRPCEngine(b),
		qse:        NewQSEEngine(b),
		lod:        NewLODEngine(b),
	}
	o.netq.maxConcurrent = defaultProfile.NetworkMaxConcurrent
	return o
}

func (o *Optimizer) ApplyProfile(name string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	var p OptimizerProfile
	switch name {
	case "balanced":
		p = defaultProfile
	case "speed":
		p = speedProfile
	case "compat":
		p = compatProfile
	case "eco":
		p = ecoProfile
	case "aggressive":
		p = aggressiveProfile
	case "turbo":
		p = turboProfile
	case "mobile":
		p = mobileProfile
	default:
		return false
	}
	o.profile = p
	o.netq.maxConcurrent = p.NetworkMaxConcurrent
	o.cache.Resize(p.CacheMaxEntries, p.CacheTTLMs)
	return true
}

func (o *Optimizer) IsEnabled() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.enabled
}

func (o *Optimizer) SetEnabled(v bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.enabled = v
}

// =============================================================================
// 1. MetricsCollector - thu thập chỉ số trang
// =============================================================================

type PageMetrics struct {
	URL           string  `json:"url"`
	LoadTimeMs    float64 `json:"loadTimeMs"`
	DOMReadyMs    float64 `json:"domReadyMs"`
	FirstPaintMs  float64 `json:"firstPaintMs"`
	ResourceCount int     `json:"resourceCount"`
	TotalSizeKB   float64 `json:"totalSizeKB"`
	ScriptCount   int     `json:"scriptCount"`
	StyleCount    int     `json:"styleCount"`
	ImageCount    int     `json:"imageCount"`
	RequestCount  int     `json:"requestCount"`
	ErrorCount    int     `json:"errorCount"`
	MemoryUsageMB float64 `json:"memoryUsageMB"`
	DOMNodeCount  int     `json:"domNodeCount"`
	Score         float64 `json:"score"`
	CollectedAt   string  `json:"collectedAt"`
}

const metricsCollectJS = `(function(){
var p=performance||{};
var t=p.timing||{};
var n=p.getEntriesByType?p.getEntriesByType('resource')||[]:[];
var imgs=document.images.length;
var scripts=document.scripts.length;
var styles=document.styleSheets.length;
var totalSize=n.reduce(function(s,e){return s+(e.transferSize||0)},0);
var imgsCount=document.querySelectorAll('img,video,source').length;
var errs=window.__networkLog?window.__networkLog.filter(function(r){return r.status>=400}).length:0;
var nodes=document.querySelectorAll('*').length;
var mem=performance.memory?performance.memory.usedJSHeapSize/1048576:0;
var score=100;
if(totalSize>2097152)score-=15;
if(n.length>80)score-=10;
if(imgsCount>40)score-=10;
if(scripts>25)score-=5;
if(nodes>2000)score-=10;
if(errs>0)score-=errs*3;
if(score<10)score=10;
return({
loadTimeMs:(t.loadEventEnd-t.navigationStart)||0,
domReadyMs:(t.domContentLoadedEventEnd-t.navigationStart)||0,
firstPaintMs:p.getEntriesByType('paint').length?p.getEntriesByType('paint')[0].startTime:0,
resourceCount:n.length,
totalSizeKB:Math.round(totalSize/1024),
scriptCount:scripts,
styleCount:styles.length,
imageCount:imgsCount,
requestCount:n.length,
errorCount:errs,
memoryUsageMB:Math.round(mem*10)/10,
domNodeCount:nodes,
score:Math.max(10,Math.min(100,score))
})})(),window.__lastMetrics=this`

func NewMetricsCollector(b *browser) *MetricsCollector {
	return &MetricsCollector{b: b, history: make([]PageMetrics, 0, 50)}
}

type MetricsCollector struct {
	b       *browser
	mu      sync.Mutex
	history []PageMetrics
}

func (mc *MetricsCollector) Collect() (*PageMetrics, error) {
	val, err := mc.b.syncUnwrap(metricsCollectJS, 10*time.Second)
	if err != nil {
		return nil, err
	}
	url := ""
	if u, err2 := mc.b.syncUnwrap("location.href", 5*time.Second); err2 == nil {
		url = fmt.Sprint(u)
	}
	var pm PageMetrics
	switch v := val.(type) {
	case map[string]interface{}:
		b, _ := json.Marshal(v)
		json.Unmarshal(b, &pm)
	case string:
		json.Unmarshal([]byte(v), &pm)
	}
	pm.URL = url
	pm.CollectedAt = time.Now().UTC().Format(time.RFC3339)

	mc.mu.Lock()
	mc.history = append(mc.history, pm)
	if len(mc.history) > 50 {
		mc.history = mc.history[len(mc.history)-50:]
	}
	mc.mu.Unlock()
	return &pm, nil
}

func (mc *MetricsCollector) History() []PageMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	out := make([]PageMetrics, len(mc.history))
	copy(out, mc.history)
	return out
}

func (mc *MetricsCollector) Average() PageMetrics {
	mc.mu.Lock()
	n := len(mc.history)
	if n == 0 {
		mc.mu.Unlock()
		return PageMetrics{Score: 100}
	}
	var lt, dr, tk, sc float64
	var rc, scnt, ic, rqc, ec, dnc int
	for _, m := range mc.history {
		lt += m.LoadTimeMs
		dr += m.DOMReadyMs
		rc += m.ResourceCount
		tk += m.TotalSizeKB
		scnt += m.ScriptCount
		ic += m.ImageCount
		rqc += m.RequestCount
		ec += m.ErrorCount
		sc += m.Score
		dnc += m.DOMNodeCount
	}
	mc.mu.Unlock()
	nf := float64(n)
	return PageMetrics{
		LoadTimeMs:    lt / nf,
		DOMReadyMs:    dr / nf,
		ResourceCount: int(float64(rc) / nf),
		TotalSizeKB:   tk / nf,
		ScriptCount:   int(float64(scnt) / nf),
		ImageCount:    int(float64(ic) / nf),
		RequestCount:  int(float64(rqc) / nf),
		ErrorCount:    int(float64(ec) / nf),
		Score:         sc / nf,
		DOMNodeCount:  int(float64(dnc) / nf),
	}
}

const priorityInjectJS = `(function(){
if(window.__mbPrioritySet)return;
window.__mbPrioritySet=true;
var B=['google-analytics.com','googletagmanager.com','doubleclick.net','facebook.net','adsystem','adservice'];
function isB(u){return u?B.some(function(b){return u.indexOf(b)>=0}):false}
var o=document.createElement('link');o.rel='preconnect';o.href='https://fonts.googleapis.com';document.head.appendChild(o);
var links=document.querySelectorAll('link[rel=stylesheet]');for(var i=0;i<links.length;i++){
if(i<2)links[i].media='all';else links[i].media='print';links[i].onload=function(){this.media='all'}}
var imgs=document.querySelectorAll('img[loading=lazy]');for(var i=0;i<imgs.length;i++){
var r=imgs[i].getBoundingClientRect();if(r.top<window.innerHeight+200)imgs[i].loading='eager'}
})()`

// =============================================================================
// 3. CompilerTuner
// =============================================================================

type CompilerTuner struct {
	GoVersion   string `json:"goVersion"`
	GCPercent   int    `json:"gcPercent"`
	NumCPU      int    `json:"numCPU"`
	GOMAXPROCS  int    `json:"gomaxprocs"`
	BuildMode   string `json:"buildMode"`
	MemoryLimit int64  `json:"memoryLimit"`
}

func GetCompilerTuner() CompilerTuner {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return CompilerTuner{
		GoVersion:   runtime.Version(),
		GCPercent:   int(debug.SetGCPercent(-1)),
		NumCPU:      runtime.NumCPU(),
		GOMAXPROCS:  runtime.GOMAXPROCS(0),
		BuildMode:   "default",
		MemoryLimit: int64(debug.SetMemoryLimit(-1)),
	}
}

func TuneCompiler(profile string) {
	switch profile {
	case "speed":
		debug.SetGCPercent(50)
		debug.SetMemoryLimit(256 * 1024 * 1024)
		runtime.GOMAXPROCS(runtime.NumCPU())
	case "balanced":
		debug.SetGCPercent(100)
		debug.SetMemoryLimit(384 * 1024 * 1024)
		runtime.GOMAXPROCS(runtime.NumCPU())
	case "eco":
		debug.SetGCPercent(150)
		debug.SetMemoryLimit(256 * 1024 * 1024)
		runtime.GOMAXPROCS(2)
	case "compat":
		debug.SetGCPercent(200)
		debug.SetMemoryLimit(512 * 1024 * 1024)
		runtime.GOMAXPROCS(1)
	case "aggressive":
		debug.SetGCPercent(30)
		debug.SetMemoryLimit(192 * 1024 * 1024)
		runtime.GOMAXPROCS(runtime.NumCPU())
	case "turbo":
		debug.SetGCPercent(15)
		debug.SetMemoryLimit(128 * 1024 * 1024)
		runtime.GOMAXPROCS(runtime.NumCPU())
	case "mobile":
		debug.SetGCPercent(80)
		debug.SetMemoryLimit(256 * 1024 * 1024)
		runtime.GOMAXPROCS(2)
	}
}

// =============================================================================
// 4. CSSJSOptimizer
// =============================================================================

type CSSJSOptimizer struct {
	b *browser
}

func NewCSSJSOptimizer(b *browser) *CSSJSOptimizer {
	return &CSSJSOptimizer{b: b}
}

const cssInject = `(function(){
if(window.__mbCSMin)return;
window.__mbCSMin=true;
var s=document.createElement('style');s.id='__mb_css_overrides';
s.textContent='img[loading=lazy]{content-visibility:auto}script[src][async],script[src][defer]{font-display:swap}';
document.head.appendChild(s);
var sheets=document.styleSheets;var used=new Set();
try{for(var i=0;i<sheets.length;i++){var rules=sheets[i].cssRules;
if(!rules)continue;for(var j=0;j<rules.length;j++){
var sel=rules[j].selectorText;if(!sel)continue;
try{if(document.querySelector(sel))used.add(i)}catch(e){}}}}
catch(e){}
var toRemove=[];for(var i=0;i<sheets.length;i++){
if(sheets[i].href&&!used.has(i)&&sheets[i].cssRules&&sheets[i].cssRules.length>3){
var link=document.querySelector('link[href="'+sheets[i].href+'"]');
if(link&&link.media!='print')toRemove.push(sheets[i].href)}}
window.__mbRemovedSheets=toRemove;
if(toRemove.length>3){
var removed=0;
for(var i=0;i<toRemove.length;i++){
var l=document.querySelector('link[href="'+toRemove[i]+'"]');
if(l&&l.media!='print'){l.media='print';removed++}}
window.__mbDeferredSheets=removed}
})(),window.__mbCSSDone=1`

const jsDeferInject = `(function(){
if(window.__mbJSDefer)return;
window.__mbJSDefer=true;
var scripts=document.querySelectorAll('script[src]:not([async]):not([defer])');
var count=0;
for(var i=0;i<scripts.length;i++){
var s=scripts[i];
if(s.src&&!s.src.includes('polyfill')&&!s.src.includes('modernizr')){
var cl=s.cloneNode(true);cl.defer=true;cl.async=false;
s.parentNode.replaceChild(cl,s);count++}}
window.__mbDeferredJSCount=count;
})(),window.__mbJSDone=1`

// =============================================================================
// 5. MediaOptimizer - xử lý ảnh và đa phương tiện
// =============================================================================

type MediaOptimizer struct {
	b *browser
}

func NewMediaOptimizer(b *browser) *MediaOptimizer {
	return &MediaOptimizer{b: b}
}

const mediaInject = `(function(){
if(window.__mbMediaOpt)return;
window.__mbMediaOpt=true;
var imgs=document.querySelectorAll('img:not([loading]),img[loading=eager]');
for(var i=0;i<imgs.length;i++){
var r=imgs[i].getBoundingClientRect();
if(r.top>window.innerHeight+500)imgs[i].loading='lazy';
if(imgs[i].width>800&&!imgs[i].srcSet){
var s=imgs[i].src;if(s&&!s.includes('svg')&&!s.includes('data:')){
var w=Math.round(Math.min(imgs[i].width,1200));
imgs[i].sizes=w+'px';imgs[i].srcSet=s+'?w='+w+' '+w+'w'}}}
var videos=document.querySelectorAll('video:not([preload])');
for(var i=0;i<videos.length;i++)videos[i].preload='metadata';
window.__mbMediaOptimized=imgs.length;
})(),window.__mbMediaDone=1`

// =============================================================================
// 6. NetworkQueue
// =============================================================================

type Priority int

type RequestItem struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Method    string    `json:"method"`
	Priority  Priority  `json:"priority"`
	Size      int       `json:"size"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	index     int
}

type PriorityQueue []*RequestItem

func (pq PriorityQueue) Len() int { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority < pq[j].Priority
	}
	return pq[i].CreatedAt.Before(pq[j].CreatedAt)
}
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*RequestItem)
	item.index = n
	*pq = append(*pq, item)
}
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}

type NetworkQueue struct {
	b              *browser
	mu             sync.Mutex
	queue          PriorityQueue
	active         int
	maxConcurrent  int
	totalQueued    int
	totalDropped   int
	blockedDomains []string
}

func NewNetworkQueue(b *browser) *NetworkQueue {
	return &NetworkQueue{
		b:             b,
		queue:         make(PriorityQueue, 0),
		maxConcurrent: 6,
		blockedDomains: []string{},
	}
}

func (nq *NetworkQueue) Enqueue(url, method string, priority Priority) {
	nq.mu.Lock()
	defer nq.mu.Unlock()

	for _, d := range nq.blockedDomains {
		if strings.Contains(url, d) {
			nq.totalDropped++
			return
		}
	}

	item := &RequestItem{
		ID:        fmt.Sprintf("rq_%d", nq.totalQueued),
		URL:       url,
		Method:    method,
		Priority:  priority,
		Status:    "queued",
		CreatedAt: time.Now(),
	}
	nq.totalQueued++
	heap.Push(&nq.queue, item)
}

func (nq *NetworkQueue) Stats() map[string]interface{} {
	nq.mu.Lock()
	defer nq.mu.Unlock()
	return map[string]interface{}{
		"queued":        nq.queue.Len(),
		"active":        nq.active,
		"maxConcurrent": nq.maxConcurrent,
		"totalQueued":   nq.totalQueued,
		"totalDropped":  nq.totalDropped,
	}
}

const networkThrottleJS = `(function(){
if(window.__mbNetThrottle)return;
window.__mbNetThrottle=true;
var maxC=6,active=0,pending=[];
var of=window.__origFetch||window.fetch;
function process(){while(active<maxC&&pending.length){
var p=pending.shift();active++;
p._fetch().then(function(){active--;process()})['catch'](function(){active--;process()})}}
window.fetch=function(u,o){
var r={url:typeof u=='string'?u:(u&&u.url)||'',method:(o&&o.method)||'GET'};
var p={_fetch:function(){return of.call(this,u,o)},url:r.url,method:r.method};
var B=['google-analytics.com','googletagmanager.com','doubleclick.net','facebook.net'];
for(var i=0;i<B.length;i++){if(r.url.indexOf(B[i])>=0)return Promise.resolve(new Response('',{status:204}))}
pending.push(p);process();
return new Promise(function(res){p._resolve=res})};
window.__mbNetThrottleActive=function(){return{active:active,pending:pending.length,max:maxC}};
})(),window.__mbNetThrottleDone=1`

// =============================================================================
// 7. SmartCache
// =============================================================================

type cacheEntry struct {
	data      interface{}
	size      int
	createdAt time.Time
	ttl       time.Duration
	hitCount  int
}

type SmartCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	maxSize int
	ttl     time.Duration
	hits    int64
	misses  int64
	evicted int64
}

func NewSmartCache(maxEntries int, ttlMs int) *SmartCache {
	return &SmartCache{
		entries: make(map[string]*cacheEntry),
		maxSize: maxEntries,
		ttl:     time.Duration(ttlMs) * time.Millisecond,
	}
}

func (sc *SmartCache) Get(key string) (interface{}, bool) {
	sc.mu.RLock()
	entry, ok := sc.entries[key]
	if !ok {
		sc.mu.RUnlock()
		sc.mu.Lock()
		sc.misses++
		sc.mu.Unlock()
		return nil, false
	}
	if time.Since(entry.createdAt) > entry.ttl {
		sc.mu.RUnlock()
		sc.mu.Lock()
		delete(sc.entries, key)
		sc.evicted++
		sc.mu.Unlock()
		return nil, false
	}
	entry.hitCount++
	sc.mu.RUnlock()
	sc.mu.Lock()
	sc.hits++
	sc.mu.Unlock()
	return entry.data, true
}

func (sc *SmartCache) Set(key string, data interface{}, size int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if len(sc.entries) >= sc.maxSize {
		sc.evictOldest()
	}

	sc.entries[key] = &cacheEntry{
		data:      data,
		size:      size,
		createdAt: time.Now(),
		ttl:       sc.ttl,
	}
}

func (sc *SmartCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true
	for k, v := range sc.entries {
		if first || v.createdAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.createdAt
			first = false
		}
	}
	if oldestKey != "" {
		delete(sc.entries, oldestKey)
		sc.evicted++
		// Batch evict expired entries too during scan
		now := time.Now()
		for k, v := range sc.entries {
			if now.Sub(v.createdAt) > v.ttl {
				delete(sc.entries, k)
				sc.evicted++
			}
		}
	}
}

func (sc *SmartCache) Resize(maxEntries int, ttlMs int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.maxSize = maxEntries
	sc.ttl = time.Duration(ttlMs) * time.Millisecond
	if len(sc.entries) > maxEntries {
		count := len(sc.entries) - maxEntries
		for i := 0; i < count; i++ {
			sc.evictOldest()
		}
	}
}

func (sc *SmartCache) Stats() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return map[string]interface{}{
		"size":    len(sc.entries),
		"maxSize": sc.maxSize,
		"hits":    sc.hits,
		"misses":  sc.misses,
		"evicted": sc.evicted,
		"ttlMs":   sc.ttl.Milliseconds(),
	}
}

func (sc *SmartCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.entries = make(map[string]*cacheEntry)
}

const cacheInjectJS = `(function(){
if(window.__mbCacheInject)return;
window.__mbCacheInject=true;
var c={};
try{c=JSON.parse(localStorage.getItem('__mb_cache')||'{}')}catch(e){}
var pending={};
var of=window.__origFetch||window.fetch;
window.fetch=function(u,o){
var key=typeof u=='string'?u:(u&&u.url)||'';
if(!key||key.indexOf('http')!==0)return of.call(this,u,o);
if(c[key]&&Date.now()-c[key].t<60000){return Promise.resolve(new Response(c[key].d,{status:200,headers:{'content-type':c[key].ct}}))}
var p=of.call(this,u,o).then(function(r){
var ct=r.headers.get('content-type')||'';
if(r.ok&&ct&&(ct.includes('json')||ct.includes('text')||ct.includes('javascript'))){
var cl=r.clone();cl.text().then(function(t){
if(t.length<51200){c[key]={d:t,ct:ct,t:Date.now()}
try{localStorage.setItem('__mb_cache',JSON.stringify(c))}catch(e){}}})['catch'](function(){})}
return r});
return p};
})(),window.__mbCacheDone=1`

// =============================================================================
// 8. AutoTuner - kiểm tra hồi quy và tự động điều chỉnh
// =============================================================================

type TuningRule struct {
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
	Action    string  `json:"action"`
	Enabled   bool    `json:"enabled"`
}

type AutoTuner struct {
	b         *browser
	mu        sync.Mutex
	rules     []TuningRule
	results   []TuneResult
	snapshots []PageMetrics
	enabled   bool
}

type TuneResult struct {
	Rule        string      `json:"rule"`
	Action      string      `json:"action"`
	Before      PageMetrics `json:"before"`
	After       PageMetrics `json:"after"`
	Improvement float64     `json:"improvement"`
	AppliedAt   string      `json:"appliedAt"`
}

func NewAutoTuner(b *browser) *AutoTuner {
	return &AutoTuner{
		b:       b,
		enabled: true,
		rules: []TuningRule{
			{Name: "high-load-time", Metric: "loadTimeMs", Operator: ">", Threshold: 5000, Action: "enable-lazy-load", Enabled: true},
			{Name: "too-many-requests", Metric: "requestCount", Operator: ">", Threshold: 60, Action: "enable-request-queue", Enabled: true},
			{Name: "too-many-images", Metric: "imageCount", Operator: ">", Threshold: 30, Action: "enable-lazy-images", Enabled: true},
			{Name: "large-dom", Metric: "domNodeCount", Operator: ">", Threshold: 2000, Action: "reduce-snapshot-depth", Enabled: true},
			{Name: "memory-high", Metric: "memoryUsageMB", Operator: ">", Threshold: 200, Action: "force-gc", Enabled: true},
			{Name: "too-many-errors", Metric: "errorCount", Operator: ">", Threshold: 5, Action: "block-trackers", Enabled: true},
			{Name: "score-low", Metric: "score", Operator: "<", Threshold: 50, Action: "aggressive-optimize", Enabled: true},
		},
	}
}

func (at *AutoTuner) Evaluate(metrics *PageMetrics) []TuneResult {
	if !at.enabled {
		return nil
	}
	at.mu.Lock()
	defer at.mu.Unlock()

	at.snapshots = append(at.snapshots, *metrics)
	if len(at.snapshots) > 20 {
		at.snapshots = at.snapshots[len(at.snapshots)-20:]
	}

	var actions []TuneResult
	for _, rule := range at.rules {
		if !rule.Enabled {
			continue
		}
		var val float64
		switch rule.Metric {
		case "loadTimeMs":
			val = metrics.LoadTimeMs
		case "requestCount":
			val = float64(metrics.RequestCount)
		case "imageCount":
			val = float64(metrics.ImageCount)
		case "domNodeCount":
			val = float64(metrics.DOMNodeCount)
		case "memoryUsageMB":
			val = metrics.MemoryUsageMB
		case "errorCount":
			val = float64(metrics.ErrorCount)
		case "score":
			val = metrics.Score
		}

		triggered := false
		switch rule.Operator {
		case ">":
			triggered = val > rule.Threshold
		case "<":
			triggered = val < rule.Threshold
		case ">=":
			triggered = val >= rule.Threshold
		case "<=":
			triggered = val <= rule.Threshold
		}

		if triggered {
			result := TuneResult{
				Rule:      rule.Name,
				Action:    rule.Action,
				Before:    *metrics,
				AppliedAt: time.Now().UTC().Format(time.RFC3339),
			}
			at.applyAction(rule.Action)
			actions = append(actions, result)
		}
	}

	at.results = append(at.results, actions...)
	if len(at.results) > 50 {
		at.results = at.results[len(at.results)-50:]
	}
	return actions
}

func (at *AutoTuner) applyAction(action string) {
	js := ""
	switch action {
	case "enable-lazy-load":
		js = mediaInject
	case "enable-lazy-images":
		js = `document.querySelectorAll('img:not([loading])').forEach(function(i){i.loading='lazy'})`
	case "enable-request-queue":
		js = networkThrottleJS
	case "reduce-snapshot-depth":
	case "force-gc":
		js = `if(window.gc)gc()`
	case "block-trackers":
		js = ``
	case "aggressive-optimize":
		js = cssInject + jsDeferInject + mediaInject
	}
	if js != "" {
		at.b.syncExec(js)
	}
}

func (at *AutoTuner) Results() []TuneResult {
	at.mu.Lock()
	defer at.mu.Unlock()
	out := make([]TuneResult, len(at.results))
	copy(out, at.results)
	return out
}

// =============================================================================
// HTTP API handlers
// =============================================================================

func (b *browser) handleOptimizerInfo(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil {
		writeError(w, 503, "optimizer not initialized")
		return
	}
	o := b.opt

	avg := o.metrics.Average()
	prof := o.profile
	compileTune := GetCompilerTuner()

	writeJSON(w, map[string]interface{}{
		"ok":           true,
		"enabled":      o.IsEnabled(),
		"profile":      prof,
		"compiler":     compileTune,
		"avgMetrics":   avg,
		"networkQueue": o.netq.Stats(),
		"smartCache":   o.cache.Stats(),
	})
}

func (b *browser) handleOptimizerMetrics(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil {
		writeError(w, 503, "optimizer not initialized")
		return
	}
	pm, err := b.opt.metrics.Collect()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	actions := b.opt.tuner.Evaluate(pm)

	response := map[string]interface{}{
		"ok":      true,
		"metrics": pm,
	}
	if len(actions) > 0 {
		response["autoTuneActions"] = actions
	}

	writeJSON(w, response)
}

func (b *browser) handleOptimizerProfile(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil {
		writeError(w, 503, "optimizer not initialized")
		return
	}
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}

	var body struct {
		Profile string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return
	}

	if body.Profile == "" {
		writeError(w, 400, "profile required (balanced/speed/compat)")
		return
	}

	if !b.opt.ApplyProfile(body.Profile) {
		writeError(w, 400, "unknown profile: "+body.Profile)
		return
	}
	TuneCompiler(body.Profile)

	writeJSON(w, map[string]interface{}{
		"ok":      true,
		"profile": b.opt.profile,
	})
}

func (b *browser) handleOptimizerTune(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil {
		writeError(w, 503, "optimizer not initialized")
		return
	}

	results := b.opt.tuner.Results()

	allMetrics := b.opt.metrics.History()

	sort.Slice(allMetrics, func(i, j int) bool {
		return allMetrics[i].CollectedAt < allMetrics[j].CollectedAt
	})

	var trend string
	if len(allMetrics) >= 2 {
		first := allMetrics[0]
		last := allMetrics[len(allMetrics)-1]
		if last.Score > first.Score {
			trend = "improving"
		} else if last.Score < first.Score {
			trend = "degrading"
		} else {
			trend = "stable"
		}
	}

	writeJSON(w, map[string]interface{}{
		"ok":              true,
		"trend":           trend,
		"dataPoints":      len(allMetrics),
		"autoTuneResults": results,
	})
}

func (b *browser) handleOptimizerRunAll(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil {
		writeError(w, 503, "optimizer not initialized")
		return
	}
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}

	o := b.opt
	r.Body.Close()
	if !o.IsEnabled() {
		writeError(w, 400, "optimizer disabled")
		return
	}

	results := map[string]interface{}{}
	errors := []string{}

	if pm, err := o.metrics.Collect(); err == nil {
		results["metrics"] = pm
		tuneActions := o.tuner.Evaluate(pm)
		if len(tuneActions) > 0 {
			results["autoTune"] = tuneActions
		}
	} else {
		errors = append(errors, "metrics: "+err.Error())
	}

	o.b.syncExec(priorityInjectJS)
	results["priority"] = "injected"

	o.b.syncExec(cssInject)
	results["cssOptimize"] = "injected"

	if o.profile.DeferNonCriticalJS {
		o.b.syncExec(jsDeferInject)
		results["jsDefer"] = "injected"
	}

	if o.profile.LazyLoadImages {
		o.b.syncExec(mediaInject)
		results["media"] = "injected"
	}

	if o.profile.RemoveTracking || o.profile.AutoTuneEnabled {
		o.b.syncExec(networkThrottleJS)
		results["networkThrottle"] = "injected"
		o.b.syncExec(cacheInjectJS)
		results["smartCache"] = "injected"
	}

	response := map[string]interface{}{
		"ok":      true,
		"results": results,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}
	writeJSON(w, response)
}

// =============================================================================
// JS injection all-in-one
// =============================================================================

const optimizerInitJS = `(function(){
if(window.__mbOptInit)return;
window.__mbOptInit=true;
var t0=performance.now();
try{
(function(){
var B=['google-analytics.com','googletagmanager.com','doubleclick.net','facebook.net','adsystem','adservice','scorecardresearch','hotjar','mouseflow'];
function m(u){return u?B.some(function(b){return u.indexOf(b)>=0}):false}
var of=window.__origFetch||window.fetch;
window.fetch=function(i,o){var u=typeof i=='string'?i:(i&&i.url)||'';return m(u)?Promise.resolve(new Response('',{status:204})):of.call(this,i,o)};
var Ox=XMLHttpRequest;if(!window.__mbXHRBlocked){window.__mbXHRBlocked=true;
var X=Ox;XMLHttpRequest=function(){var x=new X(),bl=false;
var op=x.open.bind(x);x.open=function(mtd,url){bl=m(url);if(!bl)op(mtd,url)};
var sd=x.send.bind(x);x.send=function(b){if(!bl)sd(b)};return x}}
})();
(function(){
var os=Storage.prototype.setItem;
Storage.prototype.setItem=function(k,v){if(k[0]=='_')return;return os.call(this,k,v)};
var css=document.createElement('style');
css.textContent='img[loading=lazy]{content-visibility:auto}';
document.head.appendChild(css);
var imgs=document.querySelectorAll('img:not([loading])');
for(var i=0;i<imgs.length;i++){
var r=imgs[i].getBoundingClientRect();
if(r.top>window.innerHeight+300)imgs[i].loading='lazy'}
})();
}catch(e){window.__mbOptErr=String(e)}
window.__mbOptLoadTime=performance.now()-t0;
})()`
