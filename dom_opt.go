package main

import (
	"net/http"
	"sync"
	"time"
)

type RHDGCEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	stats   RHDGCStats
}

type RHDGCStats struct {
	Total     int     `json:"total"`
	Detached  int     `json:"detached"`
	Frozen    int     `json:"frozen"`
	Destroyed int     `json:"destroyed"`
	MemoryMB  float64 `json:"memoryMB"`
	Status    string  `json:"status"`
}

var rhdgcJS = `(function(){
if(window.__mbRHD)return;
var R=window.__mbRHD={
store:{},uid:0,total:0,detached:0,frozen:0,destroyed:0,
init:function(){
var T=this;
var obs=new MutationObserver(function(muts){
muts.forEach(function(m){
if(m.type==='childList'){
m.removedNodes.forEach(function(n){
if(n.nodeType===1)T.track(n);});}});});
obs.observe(document.documentElement,{childList:true,subtree:true});
T._obs=obs;
T._interval=setInterval(function(){T.scan();},5000);
},
track:function(n){
var id='r'+(++this.uid);
this.store[id]={
node:n,heat:50,last:Date.now(),
frozen:false,destroyed:false,tag:n.tagName||'',
depth:this._depth(n),size:(n.offsetHeight||0)*(n.offsetWidth||0),
children:n.querySelectorAll('*').length
};
n.__rhd=id;
},
_depth:function(n){var d=0;while(n){n=n.parentNode;d++;}return d;},
scan:function(){
var T=this,Ts=T.store;
var now=Date.now();
T.total=Object.keys(Ts).length;
T.detached=0;T.frozen=0;T.destroyed=0;
Object.keys(Ts).forEach(function(id){
var e=Ts[id];
if(e.destroyed){T.destroyed++;return;}
if(!e.node||!e.node.parentNode){T.detached++;}else return;
if(e.frozen){T.frozen++;}else{
var sec=(now-e.last)/1000;
e.heat=Math.max(0,e.heat-sec*5);
if(e.heat<=0.1){T._freeze(id);}}
var frozenSec=(now-e.last)/1000;
if(e.frozen&&frozenSec>30){T._destroy(id);}
});
T._cleanup();
},
_freeze:function(id){
var e=this.store[id];
if(!e||!e.node)return;
try{
var n=e.node;
var p=n.parentNode;
if(p&&p.nodeType===1){
var ph=document.createElement(n.tagName||'div');
ph.style.cssText='display:none;width:'+(n.offsetWidth||0)+'px;height:'+(n.offsetHeight||0)+'px';
ph.setAttribute('data-rhd-frozen','1');
p.replaceChild(ph,n);
e.node=ph;}
e.heat=0;e.frozen=true;
}catch(ex){}
},
_destroy:function(id){
var e=this.store[id];
if(!e)return;
e.destroyed=true;
if(e.node&&e.node.parentNode){
try{e.node.parentNode.removeChild(e.node);}catch(ex){}
e.node=null;
delete this.store[id];
}
},
_cleanup:function(){
var T=this;
Object.keys(T.store).forEach(function(id){
var e=T.store[id];
if(e.destroyed&&(!e.node))delete T.store[id];});
},
stats:function(){
var T=this;
var mem=performance.memory?Math.round(performance.memory.usedJSHeapSize/1048576*10)/10:0;
return{total:T.total,detached:T.detached,frozen:T.frozen,destroyed:T.destroyed,memoryMB:mem,status:T.store?'active':'inactive'};
}
};
R.init();
})()`

func NewRHDGCEngine(b *browser) *RHDGCEngine {
	return &RHDGCEngine{b: b, enabled: true}
}

func (r *RHDGCEngine) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.enabled {
		return
	}
	r.b.syncExec(rhdgcJS)
}

var rhdGatherJS = "(function(){var s=window.__mbRHD;if(!s)return{total:0,detached:0,frozen:0,destroyed:0,memoryMB:0,status:'n/a'};return s.stats();})()"

func (r *RHDGCEngine) Gather() *RHDGCStats {
	var s RHDGCStats
	if err := r.b.syncUnwrapInto(rhdGatherJS, 5*time.Second, &s); err != nil {
		return &r.stats
	}
	r.mu.Lock()
	r.stats = s
	r.mu.Unlock()
	return &s
}

// =============================================================================
// PVC: Predictive Visibility Collapse
// =============================================================================

type PVCEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	stats   PVCStats
}

type PVCStats struct {
	Total     int     `json:"total"`
	Full      int     `json:"full"`
	Skeleton  int     `json:"skeleton"`
	Collapsed int     `json:"collapsed"`
	Expanded  int     `json:"expanded"`
	MemoryMB  float64 `json:"memoryMB"`
	Status    string  `json:"status"`
}

var pvcJS = `(function(){
if(window.__mbPVC)return;
var P=window.__mbPVC={
store:{},uid:0,total:0,full:0,skeleton:0,collapsed:0,expanded:0,
scrollY:0,scrollV:0,scrollDir:0,viewportH:0,
init:function(){
var T=this;
T.viewportH=window.innerHeight;
T.scrollY=window.scrollY;
document.addEventListener('scroll',function(){
var now=Date.now();
var sy=window.scrollY;
var dt=Math.max(16,now-(T._lastST||now));
T.scrollV=(Math.abs(sy-T.scrollY))/dt*1000;
T.scrollDir=sy>T.scrollY?1:-1;
T.scrollY=sy;
T._lastST=now;
T._predict();},{passive:true});
var obs=new IntersectionObserver(function(es){
es.forEach(function(e){
var el=e.target;
var vid=e.intersectionRatio;
var id=el.__pvc;
if(id&&P.store[id]){
var s=P.store[id];
s.vis=vid;
s.inVP=e.isIntersecting;
s.rect=el.getBoundingClientRect();
s.area=s.rect.width*s.rect.height||1;
T._score(id);}});
},{rootMargin:'200% 0px'});
P._obs=obs;
var all=document.querySelectorAll('section,article,main,aside,div[class],div[id]');
all.forEach(function(el){T._register(el);});
var mo=new MutationObserver(function(muts){
muts.forEach(function(m){
m.addedNodes.forEach(function(n){
if(n.nodeType===1&&!n.__pvc){
var tag=n.tagName.toLowerCase();
if(tag==='section'||tag==='article'||tag==='main'||tag==='aside'||tag==='div')T._register(n);
n.querySelectorAll('section,article,main,aside,div[class],div[id]').forEach(function(c){T._register(c);});}});});});
mo.observe(document.body,{childList:true,subtree:true});
P._mo=mo;
setInterval(function(){T._scoreAll();},5000);
},
_register:function(el){
if(el.__pvc)return;
var id='p'+(++this.uid);
el.__pvc=id;
this._obs.observe(el);
var rect=el.getBoundingClientRect();
this.store[id]={
el:el,tag:el.tagName||'div',
rect:rect,area:rect.width*rect.height||1,
vis:0,inVP:false,
score:0,state:'FULL',expands:0,
children:el.querySelectorAll('*').length,
height:rect.height||0
};
},
_score:function(id){
var s=this.store[id];
if(!s)return;
var cy=this.viewportH/2;
var ey=s.rect.top+s.rect.height/2;
var distVP=Math.abs(ey-cy)/Math.max(this.viewportH,1);
var centerScore=Math.max(0,1-distVP*1.5);
var scrollBoost=0;
if(this.scrollDir>0&&ey>cy)scrollBoost=0.1;
if(this.scrollDir<0&&ey<cy)scrollBoost=0.1;
var sizeScore=Math.min(1,s.area/(this.viewportH*400));
s.score=Math.min(1,centerScore+scrollBoost+sizeScore*0.3+s.vis*0.3);
if(!s.inVP)s.score*=0.3;
this._apply(id);
},
_apply:function(id){
var s=this.store[id];
if(!s||!s.el)return;
var old=s.state;
if(s.score>0.3)s.state='FULL';
else if(s.score>0.05)s.state='SKELETON';
else s.state='COLLAPSED';
if(old===s.state)return;
var el=s.el;
switch(s.state){
case'FULL':
el.style.visibility='';
el.style.display='';
el.removeAttribute('data-pvc-state');
if(old==='COLLAPSED'||old==='SKELETON'){s.expands++;P.expanded++;}
break;
case'SKELETON':
el.style.visibility='hidden';
el.style.display='block';
el.setAttribute('data-pvc-state','skeleton');
break;
case'COLLAPSED':
el.style.display='none';
el.setAttribute('data-pvc-state','collapsed');
break;}
},
_scoreAll:function(){
	var T=this;
	T.total=0;T.full=0;T.skeleton=0;T.collapsed=0;
	var orphans=[];
	Object.keys(T.store).forEach(function(id){
		var s=T.store[id];
		if(!s||!s.el||!s.el.parentNode){orphans.push(id);return;}
		T.total++;
		T._score(id);
		if(s.state==='FULL')T.full++;
		else if(s.state==='SKELETON')T.skeleton++;
		else T.collapsed++;
	});
	for(var i=0;i<orphans.length;i++)delete T.store[orphans[i]];
},
_predict:function(){
var T=this;
var vh=T.viewportH;
var scrollDir=T.scrollDir;
Object.keys(T.store).forEach(function(id){
var s=T.store[id];
if(!s||!s.el)return;
if(s.state!=='COLLAPSED'&&s.state!=='SKELETON')return;
var top=s.rect.top;
if(scrollDir>0){
if(top>0&&top<vh*1.5)T._expand(id);}
else{
if(top<vh&&top> -vh*0.5)T._expand(id);}
});
},
_expand:function(id){
var s=this.store[id];
if(!s||!s.el)return;
P.expanded++;
var el=s.el;
el.style.display='';
el.style.visibility='visible';
el.removeAttribute('data-pvc-state');
s.state='FULL';
this.full++;
},
stats:function(){
var mem=performance.memory?Math.round(performance.memory.usedJSHeapSize/1048576*10)/10:0;
return{total:P.total,full:P.full,skeleton:P.skeleton,collapsed:P.collapsed,expanded:P.expanded,memoryMB:mem,status:P.store?'active':'inactive'};
}
};
P.init();
})()`

func NewPVCEngine(b *browser) *PVCEngine {
	return &PVCEngine{b: b, enabled: true}
}

func (p *PVCEngine) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.enabled {
		return
	}
	p.b.syncExec(pvcJS)
}

var pvcGatherJS = "(function(){var s=window.__mbPVC;if(!s)return{total:0,full:0,skeleton:0,collapsed:0,expanded:0,memoryMB:0,status:'n/a'};return s.stats();})()"

func (p *PVCEngine) Gather() *PVCStats {
	var s PVCStats
	if err := p.b.syncUnwrapInto(pvcGatherJS, 5*time.Second, &s); err != nil {
		return &p.stats
	}
	p.mu.Lock()
	p.stats = s
	p.mu.Unlock()
	return &s
}

// =============================================================================
// API handlers
// =============================================================================

func (b *browser) handleRHDGCStart(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.rhd == nil {
		writeError(w, 503, "RHD not init")
		return
	}
	b.opt.rhd.Start()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "RHD-GC started"})
}

func (b *browser) handleRHDGCStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.rhd == nil {
		writeError(w, 503, "RHD not init")
		return
	}
	s := b.opt.rhd.Gather()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}

func (b *browser) handlePVCStart(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.pvc == nil {
		writeError(w, 503, "PVC not init")
		return
	}
	b.opt.pvc.Start()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "PVC started"})
}

func (b *browser) handlePVCStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.pvc == nil {
		writeError(w, 503, "PVC not init")
		return
	}
	s := b.opt.pvc.Gather()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}
