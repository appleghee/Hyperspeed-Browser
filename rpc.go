package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type RPCEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	stats   RPCStats
}

type RPCStats struct {
	Entries    int            `json:"entries"`
	Hits       int            `json:"hits"`
	Misses     int            `json:"misses"`
	Evicted    int            `json:"evicted"`
	HitRate    string         `json:"hitRate"`
	MemorySaved string        `json:"memorySaved"`
	Genomes    int            `json:"genomes"`
	TopDomain  string         `json:"topDomain"`
	Status     string         `json:"status"`
}

const rpcJS = `(function(){
if(window.__mbRPC)return;
var RPC=window.__mbRPC={
store:{},genomes:{},hits:0,misses:0,evicted:0,uid:0,
init:function(){
var T=this;
try{var s=localStorage.getItem('__rpc_genomes');if(s)T.genomes=JSON.parse(s);}catch(e){}
try{var c=localStorage.getItem('__rpc_store');if(c)T.store=JSON.parse(c);}catch(e){}
T._save();
setInterval(function(){T._decay();},60000);
},
_save:function(){
try{localStorage.setItem('__rpc_genomes',JSON.stringify(RPC.genomes));}catch(e){}
try{
var s={};
Object.keys(RPC.store).forEach(function(k){
if(RPC.store[k].score>1)s[k]=RPC.store[k];});
localStorage.setItem('__rpc_store',JSON.stringify(s));
}catch(e){}
},
_track:function(url,type,size){
if(!url)return;
var key=url.split('?')[0].split('#')[0];
var domain=(key.split('/')[2]||'unknown').replace('www.','');
var ext=key.split('.').pop().toLowerCase();
var now=Date.now();
if(!RPC.store[key]){
RPC.store[key]={
url:key,domain:domain,type:type||'other',ext:ext,
size:size||0,hits:0,misses:0,score:10,firstSeen:now,lastSeen:now
};
}
var e=RPC.store[key];
e.lastSeen=now;
e.hits++;
e.score=Math.min(200,e.score+5);
RPC.hits++;
RPC._updateGenome(domain,ext,type);
},
_miss:function(url){
var key=url.split('?')[0].split('#')[0];
if(RPC.store[key]){RPC.store[key].misses++;RPC.store[key].score=Math.max(0,RPC.store[key].score-3);}
RPC.misses++;
},
_updateGenome:function(domain,ext,type){
if(!RPC.genomes[domain]){
RPC.genomes[domain]={hits:0,types:{},exts:{},navs:0,lastVisit:Date.now()};}
var g=RPC.genomes[domain];
g.hits++;
g.lastVisit=Date.now();
g.types[type]=(g.types[type]||0)+1;
g.exts[ext]=(g.exts[ext]||0)+1;
g.navs+=g.hits>1?2:1;
},
_decay:function(){
var now=Date.now();
Object.keys(RPC.store).forEach(function(k){
var e=RPC.store[k];
var inactive=(now-e.lastSeen)/1000;
e.score=Math.max(1,e.score-inactive*0.5);
var memCost=e.size||1;
var evictScore=e.score/memCost;
if(evictScore<0.001&&inactive>120){
delete RPC.store[k];RPC.evicted++;return;}
if(inactive>3600){delete RPC.store[k];RPC.evicted++;}
});
RPC._save();
},
_genomeScore:function(domain,type){
var g=RPC.genomes[domain];
if(!g)return 50;
return((g.types[type]||0)*10)+(g.hits||0);
},
_predict:function(url){
var domain=(url.split('/')[2]||'').replace('www.','');
var ext=url.split('.').pop().toLowerCase();
var g=RPC.genomes[domain];
if(!g)return{score:10,reason:'new_domain'};
var typeScore=0;
Object.keys(g.types).forEach(function(t){typeScore+=g.types[t];});
var extScore=(g.exts[ext]||0)*20;
var navScore=Math.min(g.navs||0,200);
var total=Math.min(200,typeScore+extScore+navScore);
return{score:Math.max(5,total),reason:total>100?'high_confidence':'medium'};
},
stats:function(){
var T=RPC;
var total=Object.keys(T.store).length;
var tot=T.hits+T.misses;
var hitRate=tot>0?Math.round(T.hits/tot*100)+'%':'0%';
var mem=T.evicted*50+'KB';
var domains=Object.keys(T.genomes);
var top='';
if(domains.length){top=domains.sort(function(a,b){return T.genomes[b].hits-T.genomes[a].hits;})[0];}
return{entries:total,hits:T.hits,misses:T.misses,evicted:T.evicted,hitRate:hitRate,memorySaved:mem,genomes:domains.length,topDomain:top,status:'ok'};
}
};
RPC.init();
// Hook fetch
var _fetch=window.fetch;
window.fetch=function(u,o){
var url=typeof u==='string'?u:u&&u.url?u.url:'';
var start=Date.now();
RPC._track(url,'fetch',0);
return _fetch.call(this,u,o).then(function(r){
var cl=r.headers.get('content-length');
var size=cl?parseInt(cl):0;
if(r.ok&&size>0)RPC.store[url.split('?')[0]]&&(RPC.store[url.split('?')[0]].size=size);
return r;})["catch"](function(err){RPC._miss(url);throw err;});
};
// Hook Image - save original descriptor first
var _imgDesc=Object.getOwnPropertyDescriptor(Image.prototype,'src');
var _imgSet=_imgDesc&&_imgDesc.set;
Object.defineProperty(Image.prototype,'src',{
get:function(){return this._rpc_src||'';},
set:function(v){
this._rpc_src=v;
var url=v||'';
RPC._track(url,'image',0);
if(_imgSet)_imgSet.call(this,v);
},
configurable:true
});
// Hook XHR
var _XHRO=XMLHttpRequest.prototype.open;
XMLHttpRequest.prototype.open=function(m,u,a){
this._rpc_url=u;
RPC._track(u,'xhr',0);
return _XHRO.apply(this,arguments);
};
// Hook Link prefetch/preload
document.addEventListener('DOMContentLoaded',function(){
document.querySelectorAll('link[rel="prefetch"],link[rel="preload"],link[rel="preconnect"]').forEach(function(ln){
var href=ln.getAttribute('href');
if(href)RPC._track(href,ln.getAttribute('as')||'link',0);
});
});
// Hook navigation for genome
var _ps=history.pushState;
history.pushState=function(){
var domain=location.hostname.replace('www.','');
if(RPC.genomes[domain])RPC.genomes[domain].navs=(RPC.genomes[domain].navs||0)+1;
return _ps.apply(this,arguments);
};
window.addEventListener('popstate',function(){
var domain=location.hostname.replace('www.','');
if(RPC.genomes[domain])RPC.genomes[domain].navs=(RPC.genomes[domain].navs||0)+1;
});
console.log('%c[RPC] Revisit Probability Cache active','color:#66bb6a;font-size:11px');
console.log('[RPC] '+Object.keys(RPC.genomes).length+' genomes loaded');
})()`

var rpcGatherJS = "(function(){var s=window.__mbRPC;if(!s)return{entries:0,hits:0,misses:0,evicted:0,hitRate:'0%',memorySaved:'0',genomes:0,topDomain:'',status:'n/a'};return s.stats();})()"

var rpcGenomeJS = "(function(){var R=window.__mbRPC;if(!R)return;var g=R.genomes[(location.hostname||'').replace('www.','')];if(!g)return;console.log('[RPC] genome loaded:',Object.keys(g.types).length,'types,',g.hits,'hits');})()"

func NewRPCEngine(b *browser) *RPCEngine {
	return &RPCEngine{b: b, enabled: true}
}

func (r *RPCEngine) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.enabled {
		return
	}
	r.b.syncExec(rpcJS)
}

func (r *RPCEngine) Gather() *RPCStats {
	val, err := r.b.syncUnwrap(rpcGatherJS, 5*time.Second)
	if err != nil {
		return &r.stats
	}
	b, _ := json.Marshal(val)
	var s RPCStats
	json.Unmarshal(b, &s)
	r.mu.Lock()
	r.stats = s
	r.mu.Unlock()
	return &s
}

func (b *browser) handleRPCStart(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.rpc == nil {
		writeError(w, 503, "RPC not init")
		return
	}
	b.opt.rpc.Start()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "RPC cache engine started"})
}

func (b *browser) handleRPCStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.rpc == nil {
		writeError(w, 503, "RPC not init")
		return
	}
	s := b.opt.rpc.Gather()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}
