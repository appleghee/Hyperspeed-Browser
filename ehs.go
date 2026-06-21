package main

import (
	"net/http"
	"sync"
	"time"
)

type EHSEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	stats   EHSStats
}

type EHSStats struct {
	Hot     int     `json:"hot"`
	Warm    int     `json:"warm"`
	Cold    int     `json:"cold"`
	Dormant int     `json:"dormant"`
	Splits  int     `json:"splits"`
	Total   int     `json:"total"`
	Status  string  `json:"status"`
}

const ehsJS = `(function(){
if(window.__mbEHS)return;
(function(){
var H=window.__mbEHS={
hot:[],warm:[],cold:[],dormant:[],
uid:0,hotN:0,warmN:0,coldN:0,dormantN:0,splitN:0,totalN:0,
running:false,interval:null,
_nextId:function(){return 'ehs_'+(++H.uid);},
_add:function(cb,ms,src,heat){
var task={
id:H._nextId(),cb:cb,ms:ms||0,src:src||'unknown',
heat:heat||50,started:false,lastActive:Date.now(),
sliced:false,sliceIdx:0,dormant:false
};
H.totalN++;
if(task.heat>100){H.hot.push(task);H.hotN++;}
else if(task.heat>30){H.warm.push(task);H.warmN++;}
else{H.cold.push(task);H.coldN++;}
H._schedule();
},
_schedule:function(){
if(H.running)return;
H.running=true;
H._tick();
},
_tick:function(){
if(!H.running)return;
var task=H._pick();
if(task){
if(task.heat<20&&(Date.now()-task.lastActive)>30000&&!task.dormant){
task.dormant=true;H.dormant.push(task);H.dormantN++;H._next();return;}
if(task.heat<30&&(task.ms||0)>20&&!task.sliced){
H._split(task);H._next();return;}
H._exec(task);
}else{H._next();}
},
_pick:function(){
if(H.hot.length)return H.hot.shift();
if(H.warm.length)return H.warm.shift();
if(H.cold.length)return H.cold.shift();
if(H.dormant.length&&!H.hot.length&&!H.warm.length&&!H.cold.length){
var now=Date.now();
for(var i=0;i<H.dormant.length;i++){
if((now-H.dormant[i].lastActive)>60000)continue;
var t=H.dormant.splice(i,1)[0];t.dormant=false;return t;}
}
return null;
},
_exec:function(task){
try{task.cb();}catch(ex){}
task.lastActive=Date.now();
H._next();
},
_split:function(task){
H.splitN++;
var totalMs=task.ms||50;
var chunk=5;
var parts=Math.max(2,Math.ceil(totalMs/chunk));
var orig=task.cb;
var idx=0;
task.sliced=true;
for(var i=0;i<parts;i++){
(function(i){
var delay=i*chunk;
var tid='__ehs_slice_'+task.id+'_'+i;
setTimeout(function(){
if(!task.dormant){try{orig();}catch(ex){}}
task.lastActive=Date.now();
},delay);
})();
}
},
_next:function(){
var T=this;
setTimeout(function(){
T.running=false;
if(T.hot.length||T.warm.length||T.cold.length)T._schedule();
},1);
},
_boost:function(heat){
return function(origFn){
return function(){
H._add(function(){return origFn.apply(this,arguments);},0,'user',heat);
};
};
}
};
var _st=setTimeout;
 setTimeout=function(cb,ms){
 if(typeof cb==='function'){
 var heat=50;
 if(!ms||ms<=16)heat=100;
 else if(ms>1000)heat=5;
 else heat=Math.max(5,Math.min(200,200-ms*0.15));
 var wrapper=function(){try{cb();}catch(ex){}};
 H._add(wrapper,ms||0,'timeout',heat);
 return _st(wrapper,ms);}
 return _st(cb,ms);
 };
var _si=setInterval;
setInterval=function(cb,ms){
if(typeof cb==='function'){
var wrapper=function(){try{cb();}catch(ex){}};
H._add(wrapper,ms||1000,'interval',30);
return _si(wrapper,ms);}
return _si(cb,ms);
};
var _raf=requestAnimationFrame;
requestAnimationFrame=function(cb){
var wrapper=function(t){try{cb(t);}catch(ex){}};
H._add(wrapper,16,'raf',100);
return _raf(wrapper);
};
var _ric=requestIdleCallback;
requestIdleCallback=function(cb,opts){
if(H.cold.length||H.dormant.length){
var t=H.cold.shift()||H.dormant.shift();
if(t){try{t.cb();}catch(ex){}}}
return _ric(cb,opts);
};
var _addEL=EventTarget.prototype.addEventListener;
EventTarget.prototype.addEventListener=function(ev,fn,opts){
var self=this;
if(typeof fn==='function'&&(ev==='click'||ev==='keydown'||ev==='keyup'||ev==='mousedown'||ev==='touchstart')){
var wrapped=function(e){
H._add(function(){return fn.call(self,e);},0,'event_'+ev,200);
};
return _addEL.call(this,ev,wrapped,opts);}
return _addEL.call(this,ev,fn,opts);
};
H.interval=setInterval(function(){
var now=Date.now();
[H.hot,H.warm,H.cold].forEach(function(q){
q.forEach(function(t){
var sec=(now-t.lastActive)/1000;
t.heat=Math.max(0,t.heat-sec*2);
});
});
},5000);
})();
})()`

var ehsGatherJS = "(function(){var s=window.__mbEHS;if(!s)return{hot:0,warm:0,cold:0,dormant:0,splits:0,total:0,status:'n/a'};return{hot:s.hot.length,warm:s.warm.length,cold:s.cold.length,dormant:s.dormant.length,splits:s.splitN,total:s.totalN,status:'ok'};})()"

const ehsEasterJS = `(function(){
if(window.__mbEggs)return;
window.__mbEggs={konami:[],egg:false};
// Konami code
var seq=[38,38,40,40,37,39,37,39,66,65];
var idx=0;
document.addEventListener('keydown',function(e){
if(e.keyCode===seq[idx]){idx++;if(idx===seq.length){
idx=0;window.__mbEggs.egg=true;
var p=document.getElementById('__mb_opt_panel');
if(p){p.style.background='linear-gradient(135deg,#ff6b6b,#ffd93d,#6bcb77,#4d96ff,#9b59b6)';p.style.backgroundSize='400%400%';p.style.animation='__mbKonami 3s ease infinite';}
var s=document.createElement('style');
s.textContent='@keyframes __mbKonami{0%{background-position:0%50%}50%{background-position:100%50%}100%{background-position:0%50%}}';
document.head.appendChild(s);
console.log('%c\u2605 EASTER EGG UNLOCKED! \u2605','font-size:20px;background:linear-gradient(90deg,#ff6b6b,#ffd93d,#6bcb77);padding:8px 16px;border-radius:8px;color:#000;font-weight:bold');
console.log('%cKonami code activated \u2014 Hyperspeed rainbow mode!','color:#9b59b6;font-style:italic');}
}else{idx=0;}
});
// OpenCode whisper
document.addEventListener('keydown',function(e){
if(window.__mbEggs._oc)return;
window.__mbEggs._oc=(window.__mbEggs._oc||'')+e.key;
if(window.__mbEggs._oc.length>8)window.__mbEggs._oc=window.__mbEggs._oc.slice(-8);
if(window.__mbEggs._oc==='opencode'){
console.log('%c[opencode] I see you reading the source. Nice.','color:#4fc3f7;font-size:14px;font-style:italic');
window.__mbEggs._oc='done';}
});
// Console greeting
console.log('%cHyperspeed Browser v3.2 Genesis','font-size:18px;font-weight:bold;color:#4fc3f7');
console.log('%c19 optimization engines active — IO Cascade + HLRC + DNA + HBM + AVP + more','color:#66bb6a;font-size:11px');
console.log('%c\u2728 Type the Konami code for a surprise...','color:#ffa726;font-size:11px');
console.log('%c\u2728 Or try typing "opencode"...','color:#ab47bc;font-size:11px');
})()`

func NewEHSEngine(b *browser) *EHSEngine {
	return &EHSEngine{b: b, enabled: true}
}

func (e *EHSEngine) Start() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.enabled {
		return
	}
	e.b.syncExec(ehsJS)
	e.b.syncExec(ehsEasterJS)
}

func (e *EHSEngine) Gather() *EHSStats {
	var s EHSStats
	if err := e.b.syncUnwrapInto(ehsGatherJS, 5*time.Second, &s); err != nil {
		return &e.stats
	}
	e.mu.Lock()
	e.stats = s
	e.mu.Unlock()
	return &s
}

func (b *browser) handleEHSStart(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.ehs == nil {
		writeError(w, 503, "EHS not init")
		return
	}
	b.opt.ehs.Start()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "EHS + easter eggs injected"})
}

func (b *browser) handleEHSStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.ehs == nil {
		writeError(w, 503, "EHS not init")
		return
	}
	s := b.opt.ehs.Gather()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}
