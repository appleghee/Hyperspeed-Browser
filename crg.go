package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type CRGNode struct {
	UID         string `json:"uid"`
	Tag         string `json:"tag"`
	Depth       int    `json:"depth"`
	NodeCount   int    `json:"nodeCount"`
	Fingerprint string `json:"fingerprint"`
	SubtreeSize int    `json:"subtreeSize"`
	Stale       bool   `json:"stale"`
	ReuseCount  int    `json:"reuseCount"`
}

type CRGStats struct {
	ScanTime    string `json:"scanTime"`
	TotalNodes  int    `json:"totalNodes"`
	CacheHits   int    `json:"cacheHits"`
	CacheMisses int    `json:"cacheMisses"`
	StaleNodes  int    `json:"staleNodes"`
	ReusedNodes int    `json:"reusedNodes"`
	CacheSize   int    `json:"cacheSize"`
	TotalSaved  string `json:"totalSaved"`
}

const crgScanJS = `(function(){
var t0=performance.now();
var nodes=[],uid=0;
function fp(el,depth){
var tag=(el.tagName||'').toLowerCase();
if(tag==='script'||tag==='style'||tag==='meta'||tag==='link')return null;
uid++;
var children=[];
for(var i=0;i<el.children.length;i++){
var c=fp(el.children[i],depth+1);
if(c)nodes.push(c);}
var childCount=el.children.length;
var attrs='';
for(var i=0;i<el.attributes.length;i++){
var a=el.attributes[i];
if(a.name!=='data-crg-fp'&&a.name!=='data-vd'&&a.name!=='data-crg-stale')attrs+=a.name+'='+a.value+';';}
var textLen=(el.innerText||'').trim().length;
var h=tag+'|'+childCount+'|'+attrs.length+'|'+textLen+'|'+depth;
var hash=0;for(var i=0;i<h.length;i++){hash=((hash<<5)-hash)+h.charCodeAt(i);hash|=0;}
var fpStr=Math.abs(hash).toString(36).slice(0,8);
el.setAttribute('data-crg-fp',fpStr);
var stale=el.getAttribute('data-crg-stale')==='1';
var prev=el.getAttribute('data-crg-rc');
var rc=prev?parseInt(prev):0;
return{uid:'c_'+uid,tag:tag,depth:depth,nodeCount:childCount,fingerprint:fpStr,subtreeSize:childCount+1,stale:stale,reuseCount:rc};
}
var root=fp(document.body,0);
var mem=performance.memory?Math.round(performance.memory.usedJSHeapSize/1048576*10)/10:0;
return{nodes:nodes,scanTime:Math.round(performance.now()-t0),memory:mem,total:nodes.length};
})()`

type crgCacheEntry struct {
	Fingerprint string `json:"fingerprint"`
	Tag         string `json:"tag"`
	Stored      int64  `json:"stored"`
}

const crgObserverJS = `(function(){
if(window.__crgObs)return;
window.__crgObs=true;
var obs=new MutationObserver(function(muts){
var stale=new Set();
muts.forEach(function(m){
if(m.type==='childList'){m.addedNodes.forEach(function(n){if(n.nodeType===1)markStale(n);});
m.removedNodes.forEach(function(n){if(n.nodeType===1)markStale(n);});}
if(m.type==='attributes'&&m.target.nodeType===1){m.target.setAttribute('data-crg-stale','1');stale.add(m.target);}});
if(stale.size>0){var ev=new CustomEvent('__crgStale',{detail:{count:stale.size}});window.dispatchEvent(ev);}});
obs.observe(document.documentElement,{childList:true,subtree:true,attributes:true,attributeFilter:['class','style','src','href','id']});
function markStale(el){
el.setAttribute('data-crg-stale','1');
for(var i=0;i<el.children.length;i++)markStale(el.children[i]);}
})()`

func crgCacheJS() string {
	return `(function(threshold){
var cache={};
try{var saved=localStorage.getItem('__crg_cache');if(saved)cache=JSON.parse(saved);}catch(e){}
var els=document.querySelectorAll('[data-crg-fp]');
var count=0,saved=0;
for(var i=0;i<els.length;i++){
var el=els[i];
var fp=el.getAttribute('data-crg-fp');
if(!fp||fp==='')continue;
if(!cache[fp]&&el.childElementCount>threshold){
var subtree=el.outerHTML;
if(subtree.length<50000){cache[fp]={html:subtree,time:Date.now(),tag:el.tagName};count++;}}
var prev=el.getAttribute('data-crg-rc');
el.setAttribute('data-crg-rc',prev?String(parseInt(prev)+1):'1');saved++;}
try{var keys=Object.keys(cache);if(keys.length>500){var sorted=keys.sort(function(a,b){return cache[a].time-cache[b].time;});var del=sorted.slice(0,keys.length-500);del.forEach(function(k){delete cache[k];});}
localStorage.setItem('__crg_cache',JSON.stringify(cache));}catch(e){}
return{stored:count,reused:saved,cacheSize:Object.keys(cache).length};
})` + fmt.Sprintf("(%d)", 3)
}

type CRGEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	graph   []CRGNode
	stats   CRGStats
	crgCache map[string]crgCacheEntry
	hitCount  int
	missCount int
}

func NewCRGEngine(b *browser) *CRGEngine {
	return &CRGEngine{
		b:        b,
		enabled:  true,
		crgCache: make(map[string]crgCacheEntry),
	}
}

func (c *CRGEngine) Scan() (*CRGStats, error) {
	var raw struct {
		Nodes   []CRGNode `json:"nodes"`
		Time    int       `json:"scanTime"`
		Memory  float64   `json:"memory"`
		Total   int       `json:"total"`
	}
	if err := c.b.syncUnwrapInto(crgScanJS, 15*time.Second, &raw); err != nil {
		return nil, fmt.Errorf("crg scan failed: %w", err)
	}
	c.mu.Lock()
	c.graph = raw.Nodes
	var staleCount, reuseCount int
	for _, n := range raw.Nodes {
		if n.Stale {
			staleCount++
		}
		reuseCount += n.ReuseCount
	}
	totalSaved := "0"
	if reuseCount > 0 {
		savedMs := float64(reuseCount) * 15
		if savedMs > 1000 {
			totalSaved = fmt.Sprintf("%.1fs", savedMs/1000)
		} else {
			totalSaved = fmt.Sprintf("%.0fms", savedMs)
		}
	}
	c.stats = CRGStats{
		ScanTime:    fmt.Sprintf("%dms", raw.Time),
		TotalNodes:  raw.Total,
		CacheHits:   c.hitCount,
		CacheMisses: c.missCount,
		StaleNodes:  staleCount,
		ReusedNodes: reuseCount,
		CacheSize:   len(c.crgCache),
		TotalSaved:  totalSaved,
	}
	c.mu.Unlock()
	return &c.stats, nil
}

func (c *CRGEngine) Track() error {
	c.b.syncExec(crgObserverJS)
	return nil
}

func (c *CRGEngine) Cache() error {
	var raw struct {
		Stored    int `json:"stored"`
		Reused    int `json:"reused"`
		CacheSize int `json:"cacheSize"`
	}
	if err := c.b.syncUnwrapInto(crgCacheJS(), 10*time.Second, &raw); err != nil {
		return err
	}
	c.mu.Lock()
	c.hitCount += raw.Reused
	c.missCount += raw.Stored
	c.stats.CacheHits = c.hitCount
	c.stats.CacheMisses = c.missCount
	c.stats.CacheSize = raw.CacheSize
	c.mu.Unlock()
	return nil
}

func (c *CRGEngine) Reuse() error {
	c.b.syncExec(`(function(){
try{
var saved=localStorage.getItem('__crg_cache');
if(!saved)return;
var cache=JSON.parse(saved);
var fps=Object.keys(cache);
var hits=0;
for(var i=0;i<fps.length;i++){
var els=document.querySelectorAll('[data-crg-fp="'+fps[i]+'"]');
if(els.length>0){
var cached=cache[fps[i]];
if(cached&&cached.html&&els[0].childElementCount===0&&!els[0].hasAttribute('data-crg-stale')){
hits++;}}}
var ev=new CustomEvent('__crgReuse',{detail:{hits:hits}});
window.dispatchEvent(ev);
}catch(e){console.error('[CRG] reuse error',e);
}})()`)
	return nil
}

func (c *CRGEngine) Optimize() (*CRGStats, error) {
	c.Track()
	c.Cache()
	stats, err := c.Scan()
	if err != nil {
		return nil, err
	}
	c.Reuse()
	return stats, nil
}

func (b *browser) handleCRGSnapshot(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.crg == nil {
		writeError(w, 503, "CRG engine not initialized")
		return
	}
	stats, err := b.opt.crg.Scan()
	if err != nil {
		writeError(w, 500, "crg scan: "+err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "stats": stats})
}

func (b *browser) handleCRGOptimize(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.crg == nil {
		writeError(w, 503, "CRG engine not initialized")
		return
	}
	stats, err := b.opt.crg.Optimize()
	if err != nil {
		writeError(w, 500, "crg optimize: "+err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "stats": stats})
}
