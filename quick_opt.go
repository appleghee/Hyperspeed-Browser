package main

import (
	"net/http"
	"sync"
	"time"
)

// =============================================================================
// QuickOpt — 5 low-overhead optimisation engines combined
// =============================================================================

type QuickOptEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool

	mddpEnabled bool
	htpEnabled  bool
	psfqEnabled bool
	daeEnabled  bool
	fdtfEnabled bool

	mddp *MDDPState
	htp  *HTPState
	psfq *PSFQState
	dae  *DAEState
	fdtf *FDTFState
}

type MDDPState struct {
	DNSHits     int            `json:"dnsHits"`
	TCPHits     int            `json:"tcpHits"`
	CacheHits   int            `json:"cacheHits"`
	dnsCache    map[string]int64
}

type HTPState struct {
	Preconnects int `json:"preconnects"`
	Hovers      int `json:"hovers"`
	Cancels     int `json:"cancels"`
}

type PSFQState struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Low      int `json:"low"`
	Total    int `json:"total"`
}

type DAEState struct {
	Decoded   int `json:"decoded"`
	Evicted   int `json:"evicted"`
	Restored  int `json:"restored"`
	ActiveObs int `json:"activeObs"`
}

type FDTFState struct {
	FontsLoaded int `json:"fontsLoaded"`
	Fallbacks   int `json:"fallbacks"`
	Swapped     int `json:"swapped"`
	Locked      int `json:"locked"`
}

type QuickOptStats struct {
	MDDP *MDDPState `json:"mddp"`
	HTP  *HTPState  `json:"htp"`
	PSFQ *PSFQState `json:"psfq"`
	DAE  *DAEState  `json:"dae"`
	FDTF *FDTFState `json:"fdtf"`
}

func NewQuickOptEngine(b *browser) *QuickOptEngine {
	return &QuickOptEngine{
		b:       b,
		enabled: true,
		mddp:    &MDDPState{dnsCache: make(map[string]int64)},
		htp:     &HTPState{},
		psfq:    &PSFQState{},
		dae:     &DAEState{},
		fdtf:    &FDTFState{},
	}
}

// =============================================================================
// MDDP — Mouse-Down DNS Prefetch + Adaptive DNS/TCP
// =============================================================================

const mddpJS = `(function(){
if(window.__mbMDDP)return;
window.__mbMDDP={cache:{},ttl:300000};
document.addEventListener('mousedown',function(e){
var el=e.target.closest('a,area');
if(!el)return;
try{
var url=new URL(el.href);
var o=url.origin;
if(!o||o===location.origin)return;
var now=Date.now();
if(window.__mbMDDP.cache[o]&&(now-window.__mbMDDP.cache[o])<window.__mbMDDP.ttl)return;
window.__mbMDDP.cache[o]=now;
var rtt=navigator.connection?navigator.connection.rtt:-1;
var dns=document.createElement('link');
dns.rel='dns-prefetch';dns.href=o;
document.head.appendChild(dns);
if(rtt<0||rtt>150){
var pc=document.createElement('link');
pc.rel='preconnect';pc.href=o;pc.setAttribute('data-mddp','1');
document.head.appendChild(pc);
window.__mbMDDP._tcp=(window.__mbMDDP._tcp||0)+1;}
window.__mbMDDP._hits=(window.__mbMDDP._hits||0)+1;
}catch(ex){}},
true);})()`

// =============================================================================
// HTP — Hover-Triggered Preconnect (50ms intent)
// =============================================================================

const htpJS = `(function(){
if(window.__mbHTP)return;
window.__mbHTP={timers:{},hovers:0,cancels:0,connects:0};
document.addEventListener('mouseover',function(e){
var el=e.target.closest('a');
if(!el)return;
try{
var url=new URL(el.href);
var o=url.origin;
if(!o||o===location.origin)return;
if(window.__mbHTP.timers[o])return;
window.__mbHTP.hovers++;
window.__mbHTP.timers[o]=setTimeout(function(){
delete window.__mbHTP.timers[o];
var pc=document.createElement('link');
pc.rel='preconnect';pc.href=o;pc.setAttribute('data-htp','1');
document.head.appendChild(pc);
window.__mbHTP.connects++;
},50);
}catch(ex){}},
true);
document.addEventListener('mouseout',function(e){
var el=e.target.closest('a');
if(!el)return;
try{
var url=new URL(el.href);
var o=url.origin;
if(window.__mbHTP.timers[o]){clearTimeout(window.__mbHTP.timers[o]);delete window.__mbHTP.timers[o];window.__mbHTP.cancels++;}
}catch(ex){}},
true);})()`

// =============================================================================
// PSFQ — Priority-Static Fetch Queue (Critical > High > Low)
// =============================================================================

const psfqJS = `(function(){
if(window.__mbPSFQ)return;
window.__mbPSFQ={critical:[],high:[],low:[],c:0,h:0,l:0,total:0};
var _fetch=window.fetch;
window.fetch=function(u,o){
var prio=(o&&o.priority)||'auto';
var url=typeof u==='string'?u:u.url||'';
var cls='low';
if(url.match(/\.(css|html?)$/i)||prio==='critical')cls='critical';
else if(!url.match(/\.(png|jpg|webp|gif|svg|avif|woff2?|mp4|fetch|api)/i)||prio==='high')cls='high';
window.__mbPSFQ.total++;
window.__mbPSFQ[cls]=(window.__mbPSFQ[cls]||0)+1;
if(cls==='low'){
return new Promise(function(r){
setTimeout(function(){r(_fetch(u,o));},10);});}
return _fetch(u,o);
};
var _XHR=XMLHttpRequest.prototype.open;
XMLHttpRequest.prototype.open=function(m,u,a){
this.__mbUrl=u;
return _XHR.apply(this,arguments);
};})()`

// =============================================================================
// DAE — Decode-on-Approach with Eviction (IntersectionObserver)
// =============================================================================

const daeJS = `(function(){
if(window.__mbDAE)return;
window.__mbDAE={decoded:0,evicted:0,restored:0,obs:null};
var isSup=window.IntersectionObserver?true:false;
if(!isSup)return;
var obs=new IntersectionObserver(function(es){
es.forEach(function(e){
var img=e.target;
if(e.isIntersecting){
var src=img.getAttribute('data-src');
if(src){img.src=src;img.removeAttribute('data-src');window.__mbDAE.restored++;}
if(img.tagName==='IMG'&&!img.complete){img.decode().then(function(){window.__mbDAE.decoded++;})["catch"](function(){});}
}else{
if(img.tagName!=='IMG')return;
var rect=img.getBoundingClientRect();
if(rect.bottom< -1000||rect.top>window.innerHeight+1000){
if(img.src&&!img.src.startsWith('data:')&&!img.complete){
var tid=setTimeout(function(){
if(!img.complete){
img.setAttribute('data-src',img.src);
img.removeAttribute('srcset');img.src='';
window.__mbDAE.evicted++;}
},1000);
img.__mbEvt=tid;
if(img.__mbEvtTO)clearTimeout(img.__mbEvtTO);
img.__mbEvtTO=tid;}}});
},{rootMargin:'500px'});
window.__mbDAE.obs=obs;
var imgs=document.querySelectorAll('img');
imgs.forEach(function(i){obs.observe(i);});
var mo=new MutationObserver(function(muts){
muts.forEach(function(m){
m.addedNodes.forEach(function(n){
if(n.nodeType===1&&n.tagName==='IMG')window.__mbDAE.obs.observe(n);
if(n.nodeType===1)n.querySelectorAll('img').forEach(function(i){window.__mbDAE.obs.observe(i);});});});});
mo.observe(document.body,{childList:true,subtree:true});
window.__mbDAE._mo=mo;
})()`

// =============================================================================
// FDTF — Font Display Timeout Fallback (soft: 1s fallback, 3s lock)
// =============================================================================

const fdtfJS = `(function(){
if(window.__mbFDTF)return;
window.__mbFDTF={loaded:0,fallbacks:0,swapped:0,locked:0,timers:{}};
document.fonts.ready.then(function(){
window.__mbFDTF.loaded=document.fonts.size;});
var els=document.createElement('style');
els.textContent='@font-face{font-family:"__mbFDTF_fb";src:local("Arial");size-adjust:100%}';
document.head.appendChild(els);
var obs=new MutationObserver(function(muts){
muts.forEach(function(m){
if(m.type==='childList'){m.addedNodes.forEach(function(n){
if(n.nodeType===1&&n.tagName==='LINK'&&n.rel==='stylesheet'&&n.href&&n.href.match(/font|googleapis|fontawesome|typekit/i)){
var t1=setTimeout(function(){n.disabled=true;
var st=document.createElement('style');
st.textContent='*{font-family:"__mbFDTF_fb",serif!important}';
st.id='__mbFDTF_fb';
document.head.appendChild(st);
window.__mbFDTF.fallbacks++;
var t3=setTimeout(function(){
var fb=document.getElementById('__mbFDTF_fb');
if(fb)fb.remove();
n.disabled=false;
window.__mbFDTF.locked++;
},3000);
window.__mbFDTF.timers[n.href]=t3;},1000);
window.__mbFDTF.timers[n.href]=t1;
n.addEventListener('load',function(){
if(window.__mbFDTF.timers[n.href]){clearTimeout(window.__mbFDTF.timers[n.href]);delete window.__mbFDTF.timers[n.href];}
window.__mbFDTF.loaded++;
var fb=document.getElementById('__mbFDTF_fb');
if(fb){setTimeout(function(){if(fb.parentNode)fb.parentNode.removeChild(fb);},200);
window.__mbFDTF.swapped++;}});}});}});});
obs.observe(document.head,{childList:true,subtree:true});
})()`

// =============================================================================
// Engine methods
// =============================================================================

func (q *QuickOptEngine) InjectAll() {
	if !q.enabled {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	q.b.syncExec(mddpJS)
	q.b.syncExec(htpJS)
	q.b.syncExec(psfqJS)
	q.b.syncExec(daeJS)
	q.b.syncExec(fdtfJS)

	q.mddpEnabled = true
	q.htpEnabled = true
	q.psfqEnabled = true
	q.daeEnabled = true
	q.fdtfEnabled = true
}

func (q *QuickOptEngine) GatherStats() *QuickOptStats {
	q.mu.Lock()
	defer q.mu.Unlock()

	statsMDDP := q.gatherMDDP()
	statsHTP := q.gatherHTP()
	statsPSFQ := q.gatherPSFQ()
	statsDAE := q.gatherDAE()
	statsFDTF := q.gatherFDTF()

	return &QuickOptStats{
		MDDP: statsMDDP,
		HTP:  statsHTP,
		PSFQ: statsPSFQ,
		DAE:  statsDAE,
		FDTF: statsFDTF,
	}
}

func (q *QuickOptEngine) gatherMDDP() *MDDPState {
	var s struct {
		Hits  int `json:"hits"`
		TCP   int `json:"tcp"`
		Cache int `json:"cache"`
	}
	if err := q.b.syncUnwrapInto(`(function(){
var s=window.__mbMDDP||{};
return{hits:s._hits||0,tcp:s._tcp||0,cache:Object.keys(s.cache||{}).length};
})()`, 5*time.Second, &s); err != nil {
		return q.mddp
	}
	q.mddp.DNSHits = s.Hits
	q.mddp.TCPHits = s.TCP
	q.mddp.CacheHits = s.Cache
	return q.mddp
}

func (q *QuickOptEngine) gatherHTP() *HTPState {
	var s struct {
		Hovers    int `json:"hovers"`
		Cancels   int `json:"cancels"`
		Connects  int `json:"connects"`
	}
	if err := q.b.syncUnwrapInto(`(function(){
var s=window.__mbHTP||{};
return{hovers:s.hovers||0,cancels:s.cancels||0,connects:s.connects||0};
})()`, 5*time.Second, &s); err != nil {
		return q.htp
	}
	q.htp.Hovers = s.Hovers
	q.htp.Cancels = s.Cancels
	q.htp.Preconnects = s.Connects
	return q.htp
}

func (q *QuickOptEngine) gatherPSFQ() *PSFQState {
	var s struct {
		Critical int `json:"critical"`
		High     int `json:"high"`
		Low      int `json:"low"`
		Total    int `json:"total"`
	}
	if err := q.b.syncUnwrapInto(`(function(){
var s=window.__mbPSFQ||{};
return{critical:s.critical||0,high:s.high||0,low:s.low||0,total:s.total||0};
})()`, 5*time.Second, &s); err != nil {
		return q.psfq
	}
	q.psfq.Critical = s.Critical
	q.psfq.High = s.High
	q.psfq.Low = s.Low
	q.psfq.Total = s.Total
	return q.psfq
}

func (q *QuickOptEngine) gatherDAE() *DAEState {
	var s struct {
		Decoded  int `json:"decoded"`
		Evicted  int `json:"evicted"`
		Restored int `json:"restored"`
		Obs      int `json:"obs"`
	}
	if err := q.b.syncUnwrapInto(`(function(){
var s=window.__mbDAE||{};
return{decoded:s.decoded||0,evicted:s.evicted||0,restored:s.restored||0,obs:s.obs?1:0};
})()`, 5*time.Second, &s); err != nil {
		return q.dae
	}
	q.dae.Decoded = s.Decoded
	q.dae.Evicted = s.Evicted
	q.dae.Restored = s.Restored
	q.dae.ActiveObs = s.Obs
	return q.dae
}

func (q *QuickOptEngine) gatherFDTF() *FDTFState {
	var s struct {
		Loaded    int `json:"loaded"`
		Fallbacks int `json:"fallbacks"`
		Swapped   int `json:"swapped"`
		Locked    int `json:"locked"`
	}
	if err := q.b.syncUnwrapInto(`(function(){
var s=window.__mbFDTF||{};
return{loaded:s.loaded||0,fallbacks:s.fallbacks||0,swapped:s.swapped||0,locked:s.locked||0};
})()`, 5*time.Second, &s); err != nil {
		return q.fdtf
	}
	q.fdtf.FontsLoaded = s.Loaded
	q.fdtf.Fallbacks = s.Fallbacks
	q.fdtf.Swapped = s.Swapped
	q.fdtf.Locked = s.Locked
	return q.fdtf
}

// =============================================================================
// API handlers
// =============================================================================

func (b *browser) handleQuickOptInject(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.quick == nil {
		writeError(w, 503, "QuickOpt not initialized")
		return
	}
	b.opt.quick.InjectAll()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "All 5 quick-opt engines injected"})
}

func (b *browser) handleQuickOptStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.quick == nil {
		writeError(w, 503, "QuickOpt not initialized")
		return
	}
	stats := b.opt.quick.GatherStats()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": stats})
}
