package main

import (
	"fmt"
	"math"
	"net/http"
	"runtime"
	"sort"
	"sync"
	"time"
)

type DOMNode struct {
	UID         string  `json:"uid"`
	Tag         string  `json:"tag"`
	Visibility  float64 `json:"visibility"`
	Area        float64 `json:"area"`
	InViewport  bool    `json:"inViewport"`
	Interaction bool    `json:"interaction"`
	TextLen     int     `json:"textLen"`
	ImageCount  int     `json:"imageCount"`
	Depth       int     `json:"depth"`
	HasAdClass  bool    `json:"hasAdClass"`
	ValueScore  float64 `json:"valueScore"`
	CostScore   float64 `json:"costScore"`
	VD          float64 `json:"vd"`
}

type VDStats struct {
	ScanTime    string    `json:"scanTime"`
	TotalNodes  int       `json:"totalNodes"`
	AvgVD       float64   `json:"avgVD"`
	HighValue   int       `json:"highValue"`
	LowValue    int       `json:"lowValue"`
	BudgetMB    float64   `json:"budgetMB"`
	UsedMB      float64   `json:"usedMB"`
	FreezeZones int       `json:"freezeZones"`
	TopNodes    []DOMNode `json:"topNodes"`
}

const vdScanJS = `(function(){
var t0=performance.now();
var nodes=[],uid=0;
function isAd(el){
var c=(el.className||'')+' '+(el.id||'');
return c.match(/ad-|ads-|advertis|banner|sponsor|promo|tracker|analytics/i)?true:false;
}
function visible(el){
if(!el.offsetParent)return 0;
var r=el.getBoundingClientRect();
var vw=window.innerWidth,vh=window.innerHeight;
var visW=Math.max(0,Math.min(r.right,vw)-Math.max(r.left,0));
var visH=Math.max(0,Math.min(r.bottom,vh)-Math.max(r.top,0));
return (visW*visH)/(r.width*r.height||1);
}
function score(el,depth,parentVal){
var tag=(el.tagName||'').toLowerCase();
if(tag==='script'||tag==='style'||tag==='meta'||tag==='link')return{node:null,val:0,cost:0};
uid++;
var r=el.getBoundingClientRect();
var area=Math.max(0,r.width*r.height);
var iv=visible(el);
var intAct=el.onclick!=null||el.getAttribute('role')==='button'||tag==='button'||tag==='a'||tag==='input'||tag==='select';
var ad=isAd(el);
var val=50;ad&&(val-=30);
if(iv>0.5)val+=30;else if(iv>0)val+=10;
if(area>50000)val+=20;else if(area>10000)val+=10;
if(intAct)val+=20;
if(tag==='main'||tag==='article'||tag==='section')val+=25;
if(tag==='h1'||tag==='h2'||tag==='h3')val+=15;
if(tag==='p'&&el.innerText&&el.innerText.trim().length>50)val+=10;
if(tag==='header')val+=10;
if(tag==='nav'||tag==='footer'||tag==='aside')val-=10;
if(depth<4)val+=10;
if(parentVal>70)val+=5;
if(val<0)val=0;if(val>100)val=100;
var cost=10;
var imgs=el.querySelectorAll('img,svg,video,canvas').length;
if(imgs>0)cost+=imgs*20;
if(tag==='iframe')cost+=50;
if(el.querySelectorAll('*').length>50)cost+=15;
	if(cost<1)cost=1;
el.setAttribute('data-vd',Math.round(val/cost*100)/100);
return{
node:{uid:'v_'+uid,tag:tag,visibility:Math.round(iv*100)/100,area:Math.round(area),inViewport:iv>0.5,interaction:intAct,textLen:(el.innerText||'').trim().length,imageCount:imgs,depth:depth,hasAdClass:ad,valueScore:Math.round(val*10)/10,costScore:cost,vd:parseFloat(el.getAttribute('data-vd'))},
val:val,cost:cost};
}
function walk(el,depth){
if(depth>15||!el||el.nodeType!==1)return[];
var r=score(el,depth,0);
var res=r.node?[r.node]:[];
for(var i=0;i<el.children.length;i++){
var c=walk(el.children[i],depth+1);
if(c.length){res=res.concat(c);}}
return res;
}
nodes=walk(document.body,0);
var mem=performance.memory?Math.round(performance.memory.usedJSHeapSize/1048576*10)/10:0;
return{nodes:nodes,scanTime:Math.round(performance.now()-t0),memory:mem,total:nodes.length};
})()`

type ValueDensityEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	graph   []DOMNode
	stats   VDStats
}

func NewValueDensityEngine(b *browser) *ValueDensityEngine {
	return &ValueDensityEngine{b: b, enabled: true}
}

func (vd *ValueDensityEngine) Scan() (*VDStats, error) {
	var raw struct {
		Nodes  []DOMNode `json:"nodes"`
		Time   int       `json:"scanTime"`
		Memory float64   `json:"memory"`
		Total  int       `json:"total"`
	}
	if err := vd.b.syncUnwrapInto(vdScanJS, 15*time.Second, &raw); err != nil {
		return nil, fmt.Errorf("vd scan failed: %w", err)
	}
	vd.mu.Lock()
	vd.graph = raw.Nodes
	var totalVD, highVal, lowVal float64
	for _, n := range raw.Nodes {
		totalVD += n.VD
		if n.VD > 1 {
			highVal++
		}
		if n.VD < 0.1 {
			lowVal++
		}
	}
	avgVD := 0.0
	if len(raw.Nodes) > 0 {
		avgVD = math.Round(totalVD/float64(len(raw.Nodes))*100) / 100
	}
	budgetMB := memoryBudget()
	top := raw.Nodes
	if len(top) > 10 {
		top = top[:10]
	}
	vd.stats = VDStats{
		ScanTime:    fmt.Sprintf("%dms", raw.Time),
		TotalNodes:  raw.Total,
		AvgVD:       avgVD,
		HighValue:   int(highVal),
		LowValue:    int(lowVal),
		BudgetMB:    budgetMB,
		UsedMB:      raw.Memory,
		FreezeZones: int(lowVal),
		TopNodes:    top,
	}
	vd.mu.Unlock()

	if vd.stats.UsedMB > vd.stats.BudgetMB*0.8 {
		vd.evictLowValue()
	}
	return &vd.stats, nil
}

func memoryBudget() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	totalGB := float64(m.TotalAlloc) / 1024 / 1024 / 1024
	if totalGB < 1 {
		return 200
	}
	return math.Min(400+totalGB*50, 1200)
}

func (vd *ValueDensityEngine) evictLowValue() {
	vd.mu.Lock()
	defer vd.mu.Unlock()
	if len(vd.graph) == 0 {
		return
	}
	sorted := make([]DOMNode, len(vd.graph))
	copy(sorted, vd.graph)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].VD < sorted[j].VD
	})
	evictCount := int(float64(len(sorted)) * 0.15)
	if evictCount < 1 {
		evictCount = 1
	}
	if evictCount > len(sorted) {
		evictCount = len(sorted)
	}
	// Early exit if none are below threshold
	if len(sorted) > 0 && sorted[evictCount-1].VD >= 0.2 {
		return
	}
	evalJS := `(function(){
var els=document.querySelectorAll('[data-vd]');
var threshold=0.2;
for(var i=0;i<els.length;i++){
var v=parseFloat(els[i].getAttribute('data-vd'));
if(!isNaN(v)&&v<threshold&&els[i].tagName!=='BODY'&&els[i].tagName!=='HTML'&&els[i].tagName!=='HEAD'){
if(els[i].tagName==='IMG'||els[i].tagName==='IFRAME'){els[i].style.display='none';els[i].setAttribute('data-vd-evicted','1');continue;}
if(els[i].tagName==='SCRIPT'||els[i].tagName==='LINK'){els[i].disabled=true;continue;}
els[i].style.opacity='0.3';els[i].style.pointerEvents='none';els[i].setAttribute('data-vd-evicted','1');}
}})()`
	vd.b.syncExec(evalJS)
}

func (vd *ValueDensityEngine) Optimize() (*VDStats, error) {
	stats, err := vd.Scan()
	if err != nil {
		return nil, err
	}
	vd.evictLowValue()
	vd.applyScheduling()
	return stats, nil
}

func (vd *ValueDensityEngine) applyScheduling() {
	vd.mu.Lock()
	if len(vd.graph) == 0 {
		vd.mu.Unlock()
		return
	}
	graph := make([]DOMNode, len(vd.graph))
	copy(graph, vd.graph)
	vd.mu.Unlock()

	sort.Slice(graph, func(i, j int) bool {
		return graph[i].VD > graph[j].VD
	})
	var highVD, lowVD int
	var totalCost float64
	for _, n := range graph {
		totalCost += n.CostScore
		if n.VD > 1 {
			highVD++
		}
		if n.VD < 0.1 && n.CostScore > 30 {
			lowVD++
		}
	}
	if lowVD > 3 && highVD < lowVD {
		freezeJS := `(function(){
var all=document.querySelectorAll('[data-vd]');
for(var i=0;i<all.length;i++){
var vd=parseFloat(all[i].getAttribute('data-vd')||'99');
var tag=all[i].tagName;
if(vd>=0.2)continue;
if(tag==='IMG'){all[i].loading='lazy';if(!all[i].complete)all[i].style.visibility='hidden'}
else if(tag==='IFRAME'){var r=all[i].getBoundingClientRect();if(r.top>window.innerHeight+200&&r.bottom< -200)all[i].srcdoc='';}
else if(tag!=='BODY'&&tag!=='HTML'&&tag!=='HEAD'){all[i].style.opacity='0.3';all[i].style.pointerEvents='none';}}
var iframes=document.querySelectorAll('iframe:not([data-vd])');
for(var i=0;i<iframes.length;i++){
var r=iframes[i].getBoundingClientRect();
if(r.top>window.innerHeight+200&&r.bottom< -200)iframes[i].srcdoc=''}})()`
		vd.b.syncExec(freezeJS)
	}
}

func (b *browser) handleVDSnapshot(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.vd == nil {
		writeError(w, 503, "VD engine not initialized")
		return
	}
	stats, err := b.opt.vd.Scan()
	if err != nil {
		writeError(w, 500, "vd scan: "+err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "stats": stats})
}

func (b *browser) handleVDOptimize(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.vd == nil {
		writeError(w, 503, "VD engine not initialized")
		return
	}
	stats, err := b.opt.vd.Optimize()
	if err != nil {
		writeError(w, 500, "vd optimize: "+err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "stats": stats})
}
