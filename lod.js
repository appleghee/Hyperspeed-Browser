(function(){
if(window.__mbLOD)return;
var L=window.__mbLOD={
enabled:true,uid:0,levels:[],rects:{},saved:{},inflates:0,
INTERACTIVE:'a,button,input,select,textarea,audio,video,canvas,[tabindex],[contenteditable],[role=button],[role=link],[role=tab],[onclick],[onmousedown]',
init:function(){
var T=this;
var all=document.querySelectorAll('body *');
for(var i=0;i<all.length;i++)this._add(all[i]);
this.io=new IntersectionObserver(function(es){es.forEach(function(e){T._see(e)})},{rootMargin:'3000px'});
this.mo=new MutationObserver(function(muts){
for(var m=0;m<muts.length;m++){
var mu=muts[m];
if(mu.type==='childList'){
for(var n=0;n<mu.addedNodes.length;n++){
var nd=mu.addedNodes[n];
if(nd.nodeType===1){T._add(nd);T._addTree(nd)}
}
for(var n=0;n<mu.removedNodes.length;n++){
var nd=mu.removedNodes[n];
if(nd.nodeType===1)T._drop(nd)}
}}});
this.mo.observe(document.documentElement,{childList:true,subtree:true});
this._loop();
},
_loop:function(){
if(!this.enabled)return;
var T=this;
T._classify();
requestAnimationFrame(function(){T._loop()});
},
_add:function(n){
if(n.__lod||n.id==='__mb_bar'||n.id==='__mb_sbar')return;
var id='dl'+(++this.uid);
n.__lod=id;
this.levels[id]=0;
this.rects[id]={w:0,h:0,t:0,hi:false,len:0};
this.saved[id]=null;
if(this.io)try{this.io.observe(n)}catch(e){}
},
_addTree:function(n){
var kids=n.querySelectorAll('*');
for(var i=0;i<kids.length;i++)this._add(kids[i]);
},
_drop:function(n){
var id=n.__lod;
if(!id)return;
delete this.levels[id];
delete this.rects[id];
delete this.saved[id];
var kids=n.querySelectorAll('*');
for(var i=0;i<kids.length;i++){var k=kids[i];if(k.__lod){delete this.levels[k.__lod];delete this.rects[k.__lod];delete this.saved[k.__lod]}}
},
_see:function(e){
var n=e.target,id=n.__lod;
if(!id||!this.rects[id])return;
var r=e.boundingClientRect;
this.rects[id].w=r.width||1;this.rects[id].h=r.height||1;this.rects[id].t=r.top;
},
_classify:function(){
var vh=window.innerHeight,scrollY=window.scrollY||window.pageYOffset;
var T=this;
for(var id in T.levels){
var r=T.rects[id];if(!r)continue;
if(!r.t&&r.hi)continue;
var dist=Math.abs(r.t+r.h/2-vh/2)/Math.max(vh,1);
var nl=0;
if(dist<1.5)nl=0;
else if(dist<4)nl=1;
else if(dist<8)nl=2;
else nl=3;
var ol=T.levels[id];
if(nl===ol)continue;
T._apply(id,nl);
}
},
_apply:function(id,lv){
var T=this;
var el=document.querySelector('[__lod="'+id+'"]');
if(!el)return;
var ol=T.levels[id];
T.levels[id]=lv;
if(lv>ol){
this._compress(el,id,lv,ol);
}else{
this._inflate(el,id,lv,ol);
}
},
_compress:function(el,id,lv,ol){
var T=this,r=T.rects[id];
if(lv>=1&&ol<1){
if(el.querySelector(this.INTERACTIVE)){T.levels[id]=0;return}
var h=el.innerHTML||'';
T.saved[id]={html:h,len:h.length,w:r.w||el.offsetWidth||100,hg:r.h||el.offsetHeight||20,tag:el.tagName,cls:el.className};
r.hi=false;
}
if(lv===1){
var sd=T.saved[id];
el.innerHTML='';
el.style.minWidth=sd.w+'px';el.style.minHeight=sd.hg+'px';
el.style.overflow='hidden';
}else if(lv===2){
el.innerHTML='';
el.style.display='none';
}else if(lv===3){
var p=el.parentNode;
if(p){
var ph=document.createElement('div');
ph.style.display='none';
ph.setAttribute('data-lod',id);
p.replaceChild(ph,el);
}
}
},
_inflate:function(el,id,lv,ol){
var T=this,sd=T.saved[id];
if(lv===0){
if(sd){
if(ol===3&&!el.parentNode){
var ph=document.querySelector('[data-lod="'+id+'"]');
if(ph&&ph.parentNode){
ph.parentNode.replaceChild(el,ph);
}
}
el.innerHTML=sd.html;
el.style.minWidth='';el.style.minHeight='';el.style.overflow='';
el.style.display='';
T.inflates++;
}
}
}
};
if(document.body){L.init()}else{document.addEventListener('DOMContentLoaded',function(){L.init()})}
})();
