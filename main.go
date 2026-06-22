package main

import (
	_ "embed"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	webview "github.com/webview/webview_go"
)

//go:embed popup-blocker.js
var popupBlockerJS string

//go:embed optimizer-gui.js
var optimizerGUIJS string

//go:embed startpage.html
var startPageHTML string

// Big overlay for non-tech users (Ctrl+Shift+L)
const overlayJS = `(function(){
if(window.__mbSearch)return;window.__mbSearch=true;
var s=document.createElement('style');
s.textContent='#__mb_sbar{position:fixed;top:0;left:0;width:100%;height:100%;z-index:2147483647;background:rgba(0,0,0,0.6);display:flex;align-items:flex-start;justify-content:center;padding-top:80px;font-family:Segoe UI,sans-serif;opacity:0;pointer-events:none;transition:opacity 0.15s}'+
'#__mb_sbar.show{opacity:1;pointer-events:auto}'+
'#__mb_sbar_inner{background:#1e1e1e;border:1px solid #444;border-radius:12px;padding:16px;width:500px;max-width:90vw;box-shadow:0 8px 40px rgba(0,0,0,0.5)}'+
'#__mb_sbar_inner input{width:100%;padding:12px;background:#2d2d2d;border:1px solid #555;border-radius:8px;color:#fff;font-size:16px;outline:none;box-sizing:border-box}'+
'#__mb_sbar_inner input:focus{border-color:#4fc3f7}'+
'#__mb_sbar_inner input::placeholder{color:#777}'+
'#__mb_sbar_inner .hint{color:#888;font-size:11px;margin-top:8px;text-align:center}';
(document.head||document.documentElement).appendChild(s);
var d=document.createElement('div');d.id='__mb_sbar';
d.innerHTML='<div id="__mb_sbar_inner"><input id="__mb_sbar_inp" placeholder="Search Google..."><div class="hint">Ctrl+Shift+L</div></div>';
function addSearchBar(){if(document.body){document.body.appendChild(d);setupSearchBar()}else{requestAnimationFrame(addSearchBar)}}
function setupSearchBar(){
var inp=document.getElementById('__mb_sbar_inp');
if(!inp){setTimeout(setupSearchBar,50);return}
inp.onkeydown=function(e){if(e.key=='Enter'){var v=this.value.trim();if(v){window.goNavigate('https://www.google.com/search?q='+encodeURIComponent(v));d.classList.remove('show')}}};
d.onclick=function(e){if(e.target===d)d.classList.remove('show')};
window.__mbSearchReady=true;
}
addSearchBar();
document.addEventListener("keydown",function(e){if(e.ctrlKey&&e.shiftKey&&e.code==="KeyL"){e.preventDefault();d.classList.toggle('show');if(d.classList.contains('show')){var i=document.getElementById('__mb_sbar_inp');if(i){i.focus();i.select()}}}},true);
document.addEventListener("keydown",function(e){if(e.ctrlKey&&e.shiftKey&&e.code==="KeyH"){e.preventDefault();if(typeof window.goNavigate==='function'){window.goNavigate('hyperspeed://console')}else{var b=window.__mbBase||'http://127.0.0.1:'+(window.__mbPort||6969);fetch(b+'/api/navigate',{method:'POST',headers:{'Content-Type':'application/json','X-API-Token':window.__mbToken||''},body:'{"url":"hyperspeed://console"}'})['catch'](function(){})}}},true);
})();`

// Navigation bar — always-visible, real navigation (Back, Forward, Reload, URL input)
// Navigation bar — always-visible, real navigation (Back, Forward, Reload, URL input)
const navBarJS = `(function(){
if(document.getElementById('__mb_nav'))return;
var s=document.createElement('style');
s.textContent='#__mb_nav{position:fixed;top:0;left:0;right:0;height:36px;z-index:2147483647;background:#1a1a2e;display:flex;align-items:center;padding:2px 4px;font-family:"Segoe UI",sans-serif;box-shadow:0 1px 6px rgba(0,0,0,.7);box-sizing:border-box;gap:2px;border-bottom:1px solid #333}'+
'#__mb_nav button{background:transparent;border:none;color:#ccc;font-size:15px;cursor:pointer;width:30px;height:28px;border-radius:4px;display:inline-flex;align-items:center;justify-content:center}'+
'#__mb_nav button:hover{background:rgba(255,255,255,0.1);color:#fff}'+
'#__mb_nav button:disabled{opacity:0.25;cursor:default}'+
'#__mb_nav input{flex:1;height:26px;padding:0 10px;background:#16213e;border:1px solid #0f3460;border-radius:6px;color:#e0e0e0;font-size:13px;outline:none;min-width:0;margin:0 2px}'+
'#__mb_nav input:focus{border-color:#4fc3f7}'+
'#__mb_nav input::placeholder{color:#555}'+
'body{padding-top:36px!important}';
(document.head||document.documentElement).appendChild(s);
var d=document.createElement('div');d.id='__mb_nav';
d.innerHTML='<button id="__n_b" title="Back (Alt+Left)">\u25C0</button><button id="__n_f" title="Forward (Alt+Right)">\u25B6</button><button id="__n_r" title="Reload (Ctrl+R)">\u21BB</button><input id="__n_u" placeholder="Search or enter URL..."><button id="__n_h" style="font-size:14px;color:#4fc3f7">\u2302</button>';
function appendBar(){var b=document.body||document.documentElement;if(b){b.insertBefore(d,b.firstChild);wireBar()}else{requestAnimationFrame(appendBar)}}
appendBar();
var u,b,f,r,h;function wireBar(){
u=document.getElementById('__n_u');b=document.getElementById('__n_b');f=document.getElementById('__n_f');r=document.getElementById('__n_r');h=document.getElementById('__n_h');
if(!u){setTimeout(wireBar,30);return}
u.addEventListener('keydown',function(e){if(e.key=='Enter'){var v=this.value.trim();if(!v)return;if(v.indexOf('://')>0){window.goNavigate(v)}else{window.goNavigate('https://www.google.com/search?q='+encodeURIComponent(v))}this.blur()}});
b.addEventListener('click',function(){if(!b.disabled)window.goBack()});
f.addEventListener('click',function(){if(!f.disabled)window.goForward()});
r.addEventListener('click',function(){window.goReload()});
h.addEventListener('click',function(){window.goNavigate('hyperspeed://console')});
var lu='';upd();setInterval(upd,300);
function upd(){var c=location.href;if(c!='about:blank'&&c!==lu){lu=c;u.value=c}try{var ns=JSON.parse(window.getNavState()||'{}');b.disabled=!ns.b;f.disabled=!ns.f}catch(e){}}
}
document.addEventListener('keydown',function(e){
if(e.ctrlKey&&e.code=='KeyL'){e.preventDefault();var i=document.getElementById('__n_u');if(i){i.focus();i.select()}}
if(e.altKey&&e.code=='ArrowLeft'&&!e.ctrlKey){e.preventDefault();var x=document.getElementById('__n_b');if(x&&!x.disabled)window.goBack()}
if(e.altKey&&e.code=='ArrowRight'&&!e.ctrlKey){e.preventDefault();var x=document.getElementById('__n_f');if(x&&!x.disabled)window.goForward()}
if(e.ctrlKey&&e.code=='KeyR'&&!e.altKey&&!e.shiftKey&&!e.metaKey){e.preventDefault();window.goReload()}
});
})();`

// runtime intercept JS - hooks fetch, XHR, WebSocket, EventSource for deep capture
const runtimeJS = `(function(){
if(window.__mbHooks)return;
window.__mbHooks=true;
var L=[],WL=[],SL=[];
 window.__networkLog=L;window.__wsLog=WL;window.__sseLog=SL;
 window.__mbNetworkMax=500;
window.__origFetch=window.fetch;window.__origXHR=XMLHttpRequest;window.__origWS=WebSocket;window.__origES=EventSource;
var _f=window.fetch,_X=XMLHttpRequest,_W=WebSocket,_E=EventSource;
function tr(s,m){return s&&typeof s=='string'?s.length<=m?s:s.slice(0,m)+' [truncated]':s}
function cap(a,n){while(a.length>n)a.shift()}
function rl(r,p){r.status=p.status;r.statusText=p.statusText;r.endTime=Date.now();r.contentType=p.headers.get('content-type')||'';
r.responseHeaders={};p.headers.forEach(function(v,k){r.responseHeaders[k]=v});
var ct=r.contentType;if(ct&&ct.match(/json|text|html|xml|javascript/)){
var c=p.clone();c.text().then(function(t){r.responseBody=tr(t,10240);r.bodyLength=t.length})['catch'](function(){r.responseBody='[body read failed]'})}}
window.fetch=function(u,o){var r={url:(typeof u=='string'?u:(u&&u.url)||''),method:(o&&o.method)||'GET',requestBody:(o&&o.body)?String(o.body):null,type:'fetch',startTime:Date.now()};L.length<500&&L.push(r);
return _f.call(this,u,o).then(function(p){rl(r,p);return p})['catch'](function(e){r.error=e.message;r.endTime=Date.now();throw e})};
window.XMLHttpRequest=function(){var x=new _X(),r={type:'xhr',startTime:Date.now()};L.length<500&&L.push(r);
var o=x.open.bind(x);x.open=function(){r.method=arguments[0];r.url=arguments[1];return o.apply(x,arguments)};
var s=x.send.bind(x);x.send=function(b){r.requestBody=b?String(b):null;r.startTime=Date.now();
x.addEventListener('readystatechange',function(){if(x.readyState==4){
r.status=x.status;r.statusText=x.statusText;r.endTime=Date.now();r.contentType=x.getResponseHeader('content-type')||'';
try{var t=x.responseText;if(t){r.responseBody=tr(t,10240);r.bodyLength=t.length}}catch(e){}}});
return s(b)};return x};
function mcap(m,lim){if(m.length>lim)m.splice(0,m.length-lim)}
window.WebSocket=function(url,p){var ws=new _W(url,p),en={url:url,type:'websocket',messages:[],readyState:ws.readyState};WL.length<50&&WL.push(en);
var s=ws.send.bind(ws);ws.send=function(d){mcap(en.messages,100);en.messages.push({direction:'outgoing',payload:String(d),time:Date.now()});return s(d)};
ws.addEventListener('open',function(){en.readyState=ws.readyState});
ws.addEventListener('message',function(e){mcap(en.messages,100);en.messages.push({direction:'incoming',payload:String(e.data),time:Date.now()})});
ws.addEventListener('close',function(){en.readyState=ws.readyState});return ws};
window.EventSource=function(url,c){var es=new _E(url,c),en={url:url,type:'eventsource',messages:[],readyState:es.readyState};SL.length<50&&SL.push(en);
es.addEventListener('open',function(){en.readyState=es.readyState});
es.addEventListener('message',function(e){mcap(en.messages,100);en.messages.push({event:e.type,data:String(e.data),time:Date.now()})});
es.addEventListener('error',function(){en.readyState=es.readyState});return es};
setInterval(function(){cap(L,500);cap(WL,50);cap(SL,50);for(var i=0;i<WL.length;i++)cap(WL[i].messages,100);for(var i=0;i<SL.length;i++)cap(SL[i].messages,100)},60000);
try{!function(){
window.__turboState='started';
var B=['google-analytics.com','googletagmanager.com','googleadservices.com','pagead2.googlesyndication.com','doubleclick.net','adservice.google.com'];
function m(u){return u?B.some(function(b){return u.indexOf(b)>=0}):false}
var of=window.fetch;window.fetch=function(i,o){var u=typeof i=='string'?i:(i&&i.url)||'';return m(u)?Promise.resolve(new Response('',{status:204})):of.call(this,i,o)};
window.__turboState='fetch ok';
var Ox=XMLHttpRequest;XMLHttpRequest=function(){var x=new Ox(),bl=false;var op=x.open.bind(x);x.open=function(mtd,url){bl=m(url);if(!bl)op(mtd,url)};var sd=x.send.bind(x);x.send=function(b){if(!bl)sd(b)};return x};
var os=Storage.prototype.setItem;Storage.prototype.setItem=function(k,v){if(k[0]=='_')return;return os.call(this,k,v)};
window.__turboState='hooks ok'}();
}catch(e){window.__turboErr=String(e)}
})()`

type benchPoint struct {
	label string
	t     time.Time
}

type browser struct {
	w       webview.WebView
	history []string
	idx     int
	curr    string

	apiPort  int
	apiToken string

	mu              sync.Mutex
	lastBrowseURL   string
	browseHistory   []string
	evalID   int
	evalReqs map[int]chan string
	evalPool sync.Pool
	srv      *http.Server
	portFile string

	startHTML string
	opt       *Optimizer

	benchMu sync.Mutex
	bench   []benchPoint
}

func (b *browser) benchLog(label string) {
	b.benchMu.Lock()
	b.bench = append(b.bench, benchPoint{label: label, t: time.Now()})
	if len(b.bench) > 50 {
		b.bench = b.bench[len(b.bench)-50:]
	}
	b.benchMu.Unlock()
}

func (b *browser) benchDump() {
	b.benchMu.Lock()
	pts := make([]benchPoint, len(b.bench))
	copy(pts, b.bench)
	b.benchMu.Unlock()
	if len(pts) < 2 {
		return
	}
	base := pts[0].t
	log.Printf("[BENCH] === Benchmark (%d points) ===", len(pts))
	for i, p := range pts {
		fromStart := p.t.Sub(base).Milliseconds()
		var gap string
		if i > 0 {
			gap = fmt.Sprintf(" gap=%dms", p.t.Sub(pts[i-1].t).Milliseconds())
		}
		log.Printf("[BENCH]   %s: +%dms%s", p.label, fromStart, gap)
	}
}

func generateRandomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func main() {
	// Minimize WebView2 memory: per-site process + trim background services
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS",
		"--process-per-site "+
			"--disable-extensions --disable-sync --disable-background-networking "+
			"--disable-component-extensions-with-background-pages "+
			"--disable-features=TranslateUI,InterestFeedContentSuggestions,ChromeWhatsNewUI "+
			"--disk-cache-size=10485760 --max-decoded-image-size-mb=50")
	os.Setenv("WEBVIEW2_BROWSER_EXE_PATH", "")

	w := webview.New(false)
	defer w.Destroy()

	w.SetSize(1024, 768, webview.HintNone)

	startURL := "hyperspeed://console"
	app := &browser{
		w:        w,
		history:  []string{startURL},
		idx:      0,
		curr:     startURL,
		evalReqs: make(map[int]chan string),
		evalPool: sync.Pool{New: func() interface{} { return make(chan string, 1) }},
		apiToken: generateRandomToken(),
	}

	app.opt = NewOptimizer(app)
	TuneCompiler("balanced")

	must(w.Bind("goNavigate", app.navigate))
	must(w.Bind("goBack", app.goBack))
	must(w.Bind("goForward", app.goForward))
	must(w.Bind("goReload", app.reload))
	must(w.Bind("__evalCb", app.evalCallback))
	must(w.Bind("getNavState", func() string {
		app.mu.Lock()
		defer app.mu.Unlock()
		canBack := app.idx > 0
		canFwd := app.idx < len(app.history)-1
		bs := func(v bool) string { if v { return "true" }; return "false" }
		return `{"b":` + bs(canBack) + `,"f":` + bs(canFwd) + `}`
	}))

	apiReady := make(chan struct{})
	go app.startAPI(apiReady)
	<-apiReady
	app.startHTML = strings.ReplaceAll(startPageHTML, "{{APITOKEN}}", app.apiToken)
	app.startHTML = strings.ReplaceAll(app.startHTML, "{{APIPORT}}", fmt.Sprintf("%d", app.apiPort))
	app.opt.uhe.Start()
	app.opt.hlrc.Start()
	app.opt.lod.Start()
	app.opt.ehs.Start()
	app.opt.rpc.Start()
	app.opt.rhd.Start()
	app.opt.pvc.Start()
	w.SetTitle(fmt.Sprintf("Hyperspeed Browser [:%d] Genesis", app.apiPort))

	// Single Init call — all JS merged (navBarJS runs on every document creation)
	var initJS strings.Builder
	initJS.WriteString(fmt.Sprintf(`window.__mbPort=%d;window.__mbToken=%q;window.__mbBase='http://127.0.0.1:'+window.__mbPort;`, app.apiPort, app.apiToken))
	initJS.WriteString(overlayJS)
	initJS.WriteString(runtimeJS)
	initJS.WriteString(popupBlockerJS)
	initJS.WriteString(optimizerInitJS)
	initJS.WriteString(optimizerGUIJS)
	initJS.WriteString(lodJS)
	initJS.WriteString(navBarJS)
	w.Init(initJS.String())
	navTo(app, app.curr)
	go app.injectTurboLoop()
	go app.lazyStartEngines()
	go func() {
		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("[MEM] Alloc=%dMB Heap=%dMB Sys=%dMB Stack=%dMB Goroutines=%d WorkingSet≈%dMB",
				m.Alloc/(1024*1024), m.HeapAlloc/(1024*1024), m.Sys/(1024*1024),
				m.StackInuse/(1024*1024), runtime.NumGoroutine(),
				(m.Alloc+m.Sys)/(1024*1024))
			time.Sleep(30 * time.Second)
		}
	}()
	w.Run()

	app.stopAPI()
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Injected via w.Eval after page loads (document.head exists then).
const turboDOM = `if(document.head&&!window.__turbDOMDone){window.__turbDOMDone=true;
var ka=function(){var e=document.querySelectorAll('[class*=ad-],[class*=banner],[id*=ad_],[id*=banner],.popup,.adsbygoogle');for(var i=0;i<e.length;i++){if(e[i]&&e[i].parentNode)e[i].remove()}};ka();
var mo=new MutationObserver(function(){ka()});mo.observe(document.body,{childList:true,subtree:true});
setTimeout(function(){mo.disconnect()},10000);
window.__turboState='done';
}`

// SPB UI - injected after page load when document.head exists
const spbUI = `if(document.head&&window.__mbSPB&&!window.__mbSPBUI){
window.__mbSPBUI=true;
var c=window.__mbSPBConfig||{};
var isDark=(c.theme||'dark')==='dark';
var text=isDark?'#e0e0e0':'#333333';
var accent='#4fc3f7';
var danger='#ef5350';
var success='#66bb6a';
var warning='#ffa726';
var border=isDark?'rgba(255,255,255,0.08)':'rgba(0,0,0,0.1)';
var shadow=isDark?'0 8px 32px rgba(0,0,0,0.5)':'0 8px 32px rgba(0,0,0,0.15)';
var glassBg=isDark?'rgba(26,26,46,0.95)':'rgba(255,255,255,0.95)';
var inputBg=isDark?'rgba(255,255,255,0.05)':'rgba(0,0,0,0.03)';
var css='#spb-notification-container{position:fixed;z-index:2147483647;pointer-events:none;display:flex;flex-direction:column;gap:10px;max-width:420px;width:100%;font-family:-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Oxygen,Ubuntu,sans-serif;font-size:13px;line-height:1.5}';
css+='.spb-pos-br{bottom:20px;right:20px;align-items:flex-end}';
css+='.spb-pos-bl{bottom:20px;left:20px;align-items:flex-start}';
css+='.spb-pos-tr{top:20px;right:20px;align-items:flex-end}';
css+='.spb-pos-tl{top:20px;left:20px;align-items:flex-start}';
css+='.spb-pos-tc{top:20px;left:50%;transform:translateX(-50%);align-items:center}';
css+='.spb-pos-bc{bottom:20px;left:50%;transform:translateX(-50%);align-items:center}';
css+='.spb-toast{pointer-events:auto;background:'+glassBg+';backdrop-filter:blur(20px);-webkit-backdrop-filter:blur(20px);border:1px solid '+border+';border-radius:14px;padding:14px 16px;box-shadow:'+shadow+';color:'+text+';min-width:320px;max-width:420px;animation:spb-si 0.3s cubic-bezier(0.16,1,0.3,1);transition:all 0.25s ease;position:relative;overflow:hidden}';
css+='.spb-toast.removing{animation:spb-so 0.25s cubic-bezier(0.16,1,0.3,1) forwards;opacity:0}';
css+='.spb-toast::before{content:"";position:absolute;top:0;left:0;width:100%;height:3px;background:linear-gradient(90deg,'+accent+','+warning+');border-radius:14px 14px 0 0}';
css+='.spb-toast-header{display:flex;align-items:center;gap:10px;margin-bottom:8px}';
css+='.spb-toast-title{font-weight:600;font-size:13px;color:'+text+';word-break:break-word}';
css+='.spb-toast-domain{font-size:11px;color:'+(isDark?'#888':'#999')+';margin-top:2px}';
css+='.spb-toast-url{font-size:11px;color:'+accent+';word-break:break-all;margin:6px 0;padding:6px 10px;background:'+inputBg+';border-radius:8px;max-height:32px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;cursor:pointer;transition:max-height 0.3s}';
css+='.spb-toast-url:hover{max-height:200px;white-space:normal;overflow-y:auto}';
css+='.spb-toast-actions{display:flex;gap:6px;margin-top:10px;flex-wrap:wrap}';
css+='.spb-btn{padding:7px 13px;border-radius:8px;border:none;cursor:pointer;font-size:12px;font-weight:500;transition:all 0.2s;font-family:inherit;letter-spacing:0.01em;white-space:nowrap;outline:none}';
css+='.spb-btn:active{transform:scale(0.95)}';
css+='.spb-btn-accept{background:'+success+';color:#fff}';
css+='.spb-btn-deny{background:'+(isDark?'rgba(255,255,255,0.1)':'rgba(0,0,0,0.06)')+';color:'+text+'}';
css+='.spb-btn-trust{background:'+accent+';color:#000}';
css+='.spb-btn-block{background:'+danger+';color:#fff}';
css+='#spb-trigger{position:fixed;z-index:2147483646;bottom:20px;right:20px;width:40px;height:40px;border-radius:50%;background:'+glassBg+';backdrop-filter:blur(20px);-webkit-backdrop-filter:blur(20px);border:1px solid '+border+';box-shadow:'+shadow+';cursor:pointer;display:flex;align-items:center;justify-content:center;font-size:18px;color:'+text+';transition:all 0.3s}';
css+='#spb-trigger:hover{transform:scale(1.08);border-color:'+accent+'}';
css+='#spb-badge{position:absolute;top:-4px;right:-4px;min-width:18px;height:18px;border-radius:9px;background:'+danger+';color:#fff;font-size:10px;font-weight:700;display:flex;align-items:center;justify-content:center;padding:0 5px;box-shadow:0 2px 8px rgba(239,83,80,0.4)}';
css+='#spb-panel{position:fixed;z-index:2147483646;bottom:70px;right:20px;width:380px;max-height:70vh;background:'+glassBg+';backdrop-filter:blur(24px);-webkit-backdrop-filter:blur(24px);border:1px solid '+border+';border-radius:16px;box-shadow:'+shadow+';color:'+text+';font-family:-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Oxygen,Ubuntu,sans-serif;font-size:13px;overflow:hidden;display:none;flex-direction:column}';
css+='#spb-panel.show{display:flex;animation:spb-fi 0.25s ease}';
css+='.spb-ph{padding:16px 18px;border-bottom:1px solid '+border+';display:flex;align-items:center;justify-content:space-between;font-weight:700}';
css+='.spb-pb{padding:14px 18px;overflow-y:auto;flex:1;display:flex;flex-direction:column;gap:14px}';
css+='.spb-pb::-webkit-scrollbar{width:4px}';
css+='.spb-pb::-webkit-scrollbar-thumb{background:'+border+';border-radius:2px}';
css+='.spb-sg{display:flex;flex-direction:column;gap:6px}';
css+='.spb-sl{font-weight:600;font-size:12px;text-transform:uppercase;letter-spacing:0.04em;color:'+(isDark?'#aaa':'#666')+'}';
css+='.spb-tr{display:flex;align-items:center;justify-content:space-between;padding:8px 0}';
css+='.spb-tg{position:relative;width:48px;height:26px;flex-shrink:0}';
css+='.spb-tg input{opacity:0;width:0;height:0}';
css+='.spb-tgs{position:absolute;cursor:pointer;top:0;left:0;right:0;bottom:0;background:'+(isDark?'rgba(255,255,255,0.15)':'rgba(0,0,0,0.2)')+';border-radius:26px;transition:all 0.3s}';
css+='.spb-tgs::before{content:"";position:absolute;height:20px;width:20px;left:3px;bottom:3px;background:white;border-radius:50%;transition:all 0.3s}';
css+='.spb-tg input:checked+.spb-tgs{background:'+accent+'}';
css+='.spb-tg input:checked+.spb-tgs::before{transform:translateX(22px)}';
css+='.spb-sel,.spb-inp{width:100%;padding:8px 12px;border-radius:8px;border:1px solid '+border+';background:'+inputBg+';color:'+text+';font-family:inherit;font-size:12px;outline:none}';
css+='.spb-sel:focus,.spb-inp:focus{border-color:'+accent+'}';
css+='@keyframes spb-si{from{opacity:0;transform:translateY(20px) scale(0.95)}to{opacity:1;transform:translateY(0) scale(1)}}';
css+='@keyframes spb-so{from{opacity:1;transform:translateY(0) scale(1)}to{opacity:0;transform:translateY(-20px) scale(0.9)}}';
css+='@keyframes spb-fi{from{opacity:0;transform:translateY(10px)}to{opacity:1;transform:translateY(0)}}';
var st=document.createElement('style');st.textContent=css;document.head.appendChild(st);
var nc=document.createElement('div');nc.id='spb-notification-container';document.body.appendChild(nc);
var tr=document.createElement('div');tr.id='spb-trigger';tr.innerHTML='\\u2699\\ufe0f';
var bd=document.createElement('span');bd.id='spb-badge';bd.textContent=c.blockedCount>0?c.blockedCount:'';tr.appendChild(bd);
document.body.appendChild(tr);
}`
const spbUICode = `var q=window.__mbSPBQueue;if(q&&q.length>0&&document.getElementById('spb-notification-container')){var r=q.shift();if(r){
var nc=document.getElementById('spb-notification-container');
var t=document.createElement('div');t.className='spb-toast';
t.innerHTML='<div class="spb-toast-header"><div class="spb-toast-title">Popup blocked</div><div class="spb-toast-domain">'+r.targetDomain+'</div></div><div class="spb-toast-url">'+r.url.substring(0,80)+'</div><div class="spb-toast-actions"><button class="spb-btn spb-btn-accept" data-a="a">Allow</button><button class="spb-btn spb-btn-deny" data-a="d">Deny</button></div>';
t.querySelector('.spb-btn-accept').onclick=function(){t.remove();window._originalOpen(r.url,r.target,r.features)};
t.querySelector('.spb-btn-deny').onclick=function(){t.remove();var cfg=window.__mbSPBConfig;cfg.blockedCount++;try{localStorage.setItem('spb_config',JSON.stringify(cfg))}catch(e){}};
nc.appendChild(t);
setTimeout(function(){if(t.parentNode)t.classList.add('removing');setTimeout(function(){if(t.parentNode)t.remove()},300)},8000);
}}`

// snapshot JS — returns flat array of actionable DOM nodes with unique IDs
const snapshotJS = `(function(){
var nodes=[],uid=0;
function walk(el,d){
if(d>12||!el||el.nodeType!==1)return;
uid++;var n={uid:'s_'+uid,tag:(el.tagName||'?').toLowerCase()};
var r=el.getAttribute('role');if(r)n.role=r;
if(el.id)n.id=el.id;
if(el.type&&el.type!=='text')n.type=el.type;
if(el.placeholder)n.placeholder=el.placeholder;
if(el.value&&el.value.length<100&&(el.tagName==='INPUT'||el.tagName==='TEXTAREA'))n.value=el.value;
if(typeof el.href==='string')n.href=el.href.substring(0,300);
if(typeof el.src==='string')n.src=el.src.substring(0,300);
if(el.alt)n.text=el.alt.substring(0,100);
if(!n.text){var t=(el.innerText||'').trim().substring(0,120);if(t)n.text=t;}
n.cc=el.children.length;
if(el.tagName==='A'||el.tagName==='BUTTON'||r==='button'||el.tagName==='INPUT'||el.tagName==='TEXTAREA'||el.tagName==='SELECT')n.a=true;
try{el.dataset.si=n.uid}catch(e){}
if(n.cc>0||n.a||n.text)nodes.push(n);
for(var i=0;i<el.children.length;i++)walk(el.children[i],d+1);
}
walk(document.body,0);
return nodes;
})()`

// Optimizer GUI loaded via //go:embed optimizer-gui.js

func (b *browser) injectTurboLoop() {
	// Single attempt — JS guards prevent re-execution
	time.Sleep(100 * time.Millisecond)
	b.w.Dispatch(func() {
		b.w.Eval(turboDOM)
		b.w.Eval(spbUI)
		b.w.Eval(spbUICode)
	})
	// Inject QuickOpt engines after page settles
	time.Sleep(200 * time.Millisecond)
	if b.opt != nil && b.opt.quick != nil {
		b.opt.quick.InjectAll()
	}
}

// ---------------------------------------------------------------------------
// HTTP API server

func (b *browser) startAPI(ready chan<- struct{}) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("[api] listen error: %v", err)
		close(ready)
		return
	}
	b.apiPort = listener.Addr().(*net.TCPAddr).Port

	b.portFile = filepath.Join(os.TempDir(), "hyperspeed-browser.port")
	os.WriteFile(b.portFile, []byte(fmt.Sprintf("%d\n%s", b.apiPort, b.apiToken)), 0644)
	log.Printf("[api] listening on 127.0.0.1:%d (port file: %s)", b.apiPort, b.portFile)
	close(ready)

	mux := http.NewServeMux()
	mux.HandleFunc("/api", b.handleAPIRoot)

	// navigation
	mux.HandleFunc("/api/navigate", b.handleNavigate)
	mux.HandleFunc("/api/back", b.handleBack)
	mux.HandleFunc("/api/forward", b.handleForward)
	mux.HandleFunc("/api/reload", b.handleReload)
	mux.HandleFunc("/api/bench", b.handleBench)

	// inspection
	mux.HandleFunc("/api/eval", b.handleEval)
	mux.HandleFunc("/api/source", b.handleSource)
	mux.HandleFunc("/api/network", b.handleNetwork)
	mux.HandleFunc("/api/ws", b.handleWS)
	mux.HandleFunc("/api/sse", b.handleSSE)

	// DOM interaction
	mux.HandleFunc("/api/click", b.handleClick)
	mux.HandleFunc("/api/fill", b.handleFill)
	mux.HandleFunc("/api/snapshot", b.handleSnapshot)

	// source analysis
	mux.HandleFunc("/api/scripts", b.handleScripts)
	mux.HandleFunc("/api/runtime", b.handleRuntime)
	mux.HandleFunc("/api/storage", b.handleStorage)
	mux.HandleFunc("/api/cookies", b.handleCookies)

	// runtime hooks
	mux.HandleFunc("/api/hook", b.handleHook)
	mux.HandleFunc("/api/unhook", b.handleUnhook)

	// state
	mux.HandleFunc("/api/info", b.handleInfo)

	// screenshot
	mux.HandleFunc("/api/screenshot", b.handleScreenshot)

	// optimizer
	mux.HandleFunc("/api/opt", b.handleOptimizerInfo)
	mux.HandleFunc("/api/opt/metrics", b.handleOptimizerMetrics)
	mux.HandleFunc("/api/opt/profile", b.handleOptimizerProfile)
	mux.HandleFunc("/api/opt/run", b.handleOptimizerRunAll)
	mux.HandleFunc("/api/opt/tune", b.handleOptimizerTune)
	// PVDS endpoints
	mux.HandleFunc("/api/vd/snapshot", b.handleVDSnapshot)
	mux.HandleFunc("/api/vd/optimize", b.handleVDOptimize)

	// CRG endpoints
	mux.HandleFunc("/api/crg/snapshot", b.handleCRGSnapshot)
	mux.HandleFunc("/api/crg/optimize", b.handleCRGOptimize)

	// QuickOpt endpoints (5 engines)
	mux.HandleFunc("/api/quick/inject", b.handleQuickOptInject)
	mux.HandleFunc("/api/quick/stats", b.handleQuickOptStats)

	// RHD-GC endpoints
	mux.HandleFunc("/api/rhd/start", b.handleRHDGCStart)
	mux.HandleFunc("/api/rhd/stats", b.handleRHDGCStats)

	// PVC endpoints
	mux.HandleFunc("/api/pvc/start", b.handlePVCStart)
	mux.HandleFunc("/api/pvc/stats", b.handlePVCStats)

	// EHS endpoints
	mux.HandleFunc("/api/ehs/start", b.handleEHSStart)
	mux.HandleFunc("/api/ehs/stats", b.handleEHSStats)

	// RPC endpoints
	mux.HandleFunc("/api/rpc/start", b.handleRPCStart)
	mux.HandleFunc("/api/rpc/stats", b.handleRPCStats)

	// QSE endpoints
	mux.HandleFunc("/api/qse/start", b.handleQSEStart)
	mux.HandleFunc("/api/qse/stats", b.handleQSEStats)
	mux.HandleFunc("/api/qse/add", b.handleQSEAdd)

	mux.HandleFunc("/api/lod/start", b.handleLODStart)
	mux.HandleFunc("/api/lod/stats", b.handleLODStats)
	mux.HandleFunc("/api/lod/toggle", b.handleLODToggle)
	mux.HandleFunc("/api/lod", b.handleLOD)
	mux.HandleFunc("/api/ioc/stats", b.handleIOCStats)

	// UHE endpoints
	// Browse history endpoints
	mux.HandleFunc("/api/browse/last", b.handleBrowseLast)
	mux.HandleFunc("/api/browse/history", b.handleBrowseHistory)

	// NDF endpoints
	mux.HandleFunc("/api/ndf/stats", b.handleNDFStats)
	mux.HandleFunc("/api/ndf/clear", b.handleNDFClear)

	// AutoTune endpoints
	mux.HandleFunc("/api/autotune/profiles", b.handleAutoTuneProfiles)
	mux.HandleFunc("/api/autotune/metrics", b.handleAutoTuneMetrics)

	// GC Controller endpoints
	mux.HandleFunc("/api/gc/stats", b.handleGCStats)

	// Adapt engine orchestration
	mux.HandleFunc("/api/adapt/stats", b.handleAdaptStats)

	// v3.2.0 Genesis engine endpoints
	mux.HandleFunc("/api/dna/fingerprint", b.handleDNAFingerprint)
	mux.HandleFunc("/api/dna/stats", b.handleDNAStats)
	mux.HandleFunc("/api/dna/clear", b.handleDNAClear)
	mux.HandleFunc("/api/hbm/stats", b.handleHBMStats)
	mux.HandleFunc("/api/avp/stats", b.handleAVPStats)
	mux.HandleFunc("/api/domcompress/stats", b.handleDOMCompressStats)
	mux.HandleFunc("/api/ncg/stats", b.handleNCGStats)
	mux.HandleFunc("/api/pce/stats", b.handlePCEStats)
	mux.HandleFunc("/api/upm/stats", b.handleUPMStats)
	mux.HandleFunc("/api/dra/stats", b.handleDRAStats)
	mux.HandleFunc("/api/mcs/stats", b.handleMCSStats)
	mux.HandleFunc("/api/cbl/stats", b.handleCBLStats)
	mux.HandleFunc("/api/uee/stats", b.handleUEEStats)
	mux.HandleFunc("/api/hfs/stats", b.handleHFSStats)
	mux.HandleFunc("/api/rcm/stats", b.handleRCMStats)

	mux.HandleFunc("/api/uhe/start", b.handleUHEStart)
	mux.HandleFunc("/api/uhe/stats", b.handleUHEStats)
	mux.HandleFunc("/api/uhe/access", b.handleUHEAccess)
	mux.HandleFunc("/api/uhe/top", b.handleUHETop)
	mux.HandleFunc("/api/uhe", b.handleUHE)

	// HLRC endpoints (v3.2 Paradigm)
	mux.HandleFunc("/api/hlrc/stats", b.handleHLRCStats)
	mux.HandleFunc("/api/hlrc/objects", b.handleHLRCObjects)
	mux.HandleFunc("/api/hlrc/access", b.handleHLRCAccess)
	mux.HandleFunc("/api/hlrc/register", b.handleHLRCRegister)
	mux.HandleFunc("/api/hlrc/config", b.handleHLRCConfig)

	b.srv = &http.Server{
		Handler:      corsMiddleware(authMiddleware(b, mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}
	b.srv.Serve(listener)
}

func (b *browser) stopAPI() {
	if b.portFile != "" {
		os.Remove(b.portFile)
	}
	if b.srv != nil {
		b.srv.Close()
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Token")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(b *browser, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Token") != b.apiToken {
			writeError(w, 401, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": msg})
}

// ---------------------------------------------------------------------------
// Sync eval helpers

func (b *browser) evalCallback(id int, result string) {
	b.mu.Lock()
	ch := b.evalReqs[id]
	delete(b.evalReqs, id)
	b.mu.Unlock()
	if ch != nil {
		select {
		case ch <- result:
		default:
		}
	}
}

// syncEval eval JS and return JSON-stringified result.
func (b *browser) syncEval(js string, timeout time.Duration) (string, error) {
	ch := b.evalPool.Get().(chan string)
	b.mu.Lock()
	id := b.evalID
	b.evalID++
	b.evalReqs[id] = ch
	b.mu.Unlock()

	code := fmt.Sprintf(`__evalCb(%d,(function(){try{return JSON.stringify(%s)}catch(e){return JSON.stringify({error:e.message})}})())`, id, js)
	b.w.Dispatch(func() { b.w.Eval(code) })

	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	select {
	case result := <-ch:
		b.evalPool.Put(ch)
		return result, nil
	case <-time.After(timeout):
		b.mu.Lock()
		delete(b.evalReqs, id)
		b.mu.Unlock()
		b.evalPool.Put(ch)
		return "", fmt.Errorf("eval timeout after %v", timeout)
	}
}

// syncUnwrap eval JS, JSON-parse once to undo the automatic stringify.
func (b *browser) syncUnwrap(js string, timeout time.Duration) (interface{}, error) {
	raw, err := b.syncEval(js, timeout)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal([]byte(raw), &val); err != nil {
		return raw, nil
	}
	return val, nil
}

// syncExec fire-and-forget eval (no result).
func (b *browser) syncExec(js string) {
	b.w.Dispatch(func() { b.w.Eval(js) })
}

// syncUnwrapInto eval JS and unmarshal directly into target struct — avoids the
// json.Marshal→json.Unmarshal double-conversion pattern seen throughout the codebase.
func (b *browser) syncUnwrapInto(js string, timeout time.Duration, target interface{}) error {
	raw, err := b.syncEval(js, timeout)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), target)
}

// ---------------------------------------------------------------------------
// API handlers

func (b *browser) handleAPIRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"ok":      true,
		"name":    "Mini Browser Deep Inspect API",
		"version": "3.2.0-genesis",
		"endpoints": []map[string]string{
			{"path": "GET  /api", "desc": "API info"},
			{"path": "POST /api/navigate", "desc": "Navigate to URL {\"url\":\"...\"}"},
			{"path": "POST /api/back", "desc": "Go back"},
			{"path": "POST /api/forward", "desc": "Go forward"},
			{"path": "POST /api/reload", "desc": "Reload current page"},
			{"path": "POST /api/eval", "desc": "Run arbitrary JS {\"js\":\"...\"}"},
			{"path": "GET  /api/source", "desc": "Page HTML source"},
			{"path": "GET  /api/network", "desc": "Captured network requests (fetch+XHR)"},
			{"path": "GET  /api/ws", "desc": "WebSocket message log"},
			{"path": "GET  /api/sse", "desc": "SSE/EventSource message log"},
			{"path": "POST /api/click", "desc": "Click element {\"selector\":\"...\"} or {\"uid\":\"s_5\"}"},
			{"path": "POST /api/fill", "desc": "Fill input {\"selector\":\"...\",\"value\":\"...\"} or {\"uid\":\"s_5\",\"value\":\"...\"}"},
			{"path": "GET  /api/snapshot", "desc": "DOM snapshot tree with unique uids (s_1..s_N)"},
			{"path": "GET  /api/scripts", "desc": "Script inventory (all <script> tags)"},
			{"path": "GET  /api/runtime", "desc": "Detected JS framework"},
			{"path": "GET  /api/storage", "desc": "localStorage + sessionStorage"},
			{"path": "GET  /api/cookies", "desc": "document.cookie"},
			{"path": "POST /api/hook", "desc": "Inject custom JS hook {\"js\":\"...\"}"},
			{"path": "POST /api/unhook", "desc": "Restore original hooks"},
			{"path": "GET  /api/info", "desc": "Browser state (URL, history, etc.)"},
			{"path": "GET  /api/screenshot", "desc": "Capture screenshot (PNG, base64)"},
			{"path": "GET  /api/opt", "desc": "Optimizer info + stats"},
			{"path": "GET  /api/opt/metrics", "desc": "Collect page performance metrics"},
			{"path": "POST /api/opt/profile", "desc": "Switch profile: balanced|speed|compat"},
			{"path": "POST /api/opt/run", "desc": "Run full optimization pipeline"},
			{"path": "GET  /api/opt/tune", "desc": "Auto-tune results + trend"},
			{"path": "GET  /api/ioc/stats", "desc": "IO Cascade stats (v3.2 alpha)"},
			{"path": "GET  /api/dna/fingerprint", "desc": "Page DNA fingerprint (v3.2 Genesis)"},
			{"path": "GET  /api/dna/stats", "desc": "Page DNA stats"},
			{"path": "POST /api/dna/clear", "desc": "Clear DNA cache"},
			{"path": "GET  /api/hbm/stats", "desc": "Heat-Based Memory allocator stats"},
			{"path": "GET  /api/avp/stats", "desc": "Adaptive Viewport Predictor stats"},
			{"path": "GET  /api/domcompress/stats", "desc": "DOM Compression stats"},
			{"path": "GET  /api/ncg/stats", "desc": "Network Cost Graph stats"},
			{"path": "GET  /api/pce/stats", "desc": "Page Change Engine stats"},
			{"path": "GET  /api/upm/stats", "desc": "User Presence Model stats"},
			{"path": "GET  /api/dra/stats", "desc": "Dynamic Resource Adjustment stats"},
			{"path": "GET  /api/mcs/stats", "desc": "Micro-Controller Scheduler stats"},
			{"path": "GET  /api/cbl/stats", "desc": "Content-Based Loading stats"},
			{"path": "GET  /api/uee/stats", "desc": "Unified Event Engine stats"},
			{"path": "GET  /api/hfs/stats", "desc": "Heat-File System stats"},
			{"path": "GET  /api/rcm/stats", "desc": "Resource Cost Model stats"},
		},
	})
}

func (b *browser) handleNavigate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var body struct{ URL string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return
	}
	if b.opt != nil && b.opt.qse != nil {
		if resolved, ok := b.opt.qse.Resolve(body.URL); ok {
			b.navigate(resolved)
			writeJSON(w, map[string]interface{}{"ok": true, "shortcut": true, "resolved": resolved})
			return
		}
	}
	b.navigate(body.URL)
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (b *browser) handleBack(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	b.goBack()
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (b *browser) handleForward(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	b.goForward()
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (b *browser) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	b.reload()
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (b *browser) handleEval(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var body struct{ JS string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return
	}
	result, err := b.syncEval(body.JS, 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": result})
}

func (b *browser) handleSource(w http.ResponseWriter, r *http.Request) {
	val, err := b.syncUnwrap("document.documentElement.outerHTML", 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": val})
}

func (b *browser) handleNetwork(w http.ResponseWriter, r *http.Request) {
	val, err := b.syncUnwrap("(window.__networkLog||[])", 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": val})
}

func (b *browser) handleWS(w http.ResponseWriter, r *http.Request) {
	val, err := b.syncUnwrap("(window.__wsLog||[])", 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": val})
}

func (b *browser) handleSSE(w http.ResponseWriter, r *http.Request) {
	val, err := b.syncUnwrap("(window.__sseLog||[])", 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": val})
}

func (b *browser) handleClick(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var body struct {
		Selector string `json:"selector"`
		UID      string `json:"uid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return
	}
	sel := body.Selector
	if body.UID != "" {
		sel = "[data-si=\"" + body.UID + "\"]"
	}
	js := fmt.Sprintf(`(function(){
var el=document.querySelector(%q);
if(!el)throw new Error('element not found: %s');
if(typeof el.click=='function'){el.click();return'clicked'}
el.dispatchEvent(new MouseEvent('click',{bubbles:true,cancelable:true,view:window}));
return'clicked'
})()`, sel, sel)
	result, err := b.syncEval(js, 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": result})
}

func (b *browser) handleFill(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var body struct {
		Selector string `json:"selector"`
		UID      string `json:"uid"`
		Value    string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return
	}
	sel := body.Selector
	if body.UID != "" {
		sel = "[data-si=\"" + body.UID + "\"]"
	}
	js := fmt.Sprintf(`(function(){
var el=document.querySelector(%q);
if(!el)throw new Error('element not found: %s');
el.focus();el.select();
var ns=Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype,'value').set;
if(!ns&&el.tagName==='TEXTAREA')ns=Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype,'value').set;
var val=%q;
if(ns){ns.call(el,val);el.dispatchEvent(new Event('input',{bubbles:true,data:val}))}
else{el.value=val;el.dispatchEvent(new Event('input',{bubbles:true}))}
el.dispatchEvent(new Event('change',{bubbles:true}));
return'filled'
})()`, sel, sel, body.Value)
	result, err := b.syncEval(js, 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": result})
}

func (b *browser) handleScripts(w http.ResponseWriter, r *http.Request) {
	js := `Array.from(document.scripts).map(function(s){
return{src:s.src||null,inline:!s.src,type:s.type||'text/javascript',content:s.src?null:s.textContent.substring(0,5000)}})`
	val, err := b.syncUnwrap(js, 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": val})
}

func (b *browser) handleRuntime(w http.ResponseWriter, r *http.Request) {
	js := `(function(){
if(window.__NEXT_DATA__||window.next) return "Next.js";
if(window.__NUXT__) return "Nuxt";
if(window.React&&window.React.Fragment) return "React";
if(window.Vue) return "Vue";
if(window.Angular) return "Angular";
if(document.querySelector('[ng-app]')) return "AngularJS";
return "Unknown";
})()`
	val, err := b.syncUnwrap(js, 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": val})
}

func (b *browser) handleStorage(w http.ResponseWriter, r *http.Request) {
	js := `(function(){
try{var ls=Object.entries(localStorage).map(function(e){return{key:e[0],value:e[1]}})
}catch(e){var ls=[]}
try{var ss=Object.entries(sessionStorage).map(function(e){return{key:e[0],value:e[1]}})
}catch(e){var ss=[]}
return{localStorage:ls,sessionStorage:ss}
})()`
	val, err := b.syncUnwrap(js, 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": val})
}

func (b *browser) handleCookies(w http.ResponseWriter, r *http.Request) {
	js := `document.cookie.split('; ').filter(Boolean).map(function(c){
var i=c.indexOf('=');return{key:c.substring(0,i),value:c.substring(i+1)}})`
	val, err := b.syncUnwrap(js, 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": val})
}

func (b *browser) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	val, err := b.syncUnwrap(snapshotJS, 0)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "result": val})
}

func (b *browser) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	winTitle := fmt.Sprintf("Hyperspeed Browser [:%d]", b.apiPort)
	psScript := fmt.Sprintf(`
Add-Type -AssemblyName System.Drawing, System.Windows.Forms;
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class Win32 {
	[DllImport("user32.dll")]
	public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
	[DllImport("user32.dll")]
	public static extern bool PrintWindow(IntPtr hWnd, IntPtr hdcBlt, int nFlags);
}
public struct RECT { public int Left; public int Top; public int Right; public int Bottom; }
"@;
$hwnd = (Get-Process | Where-Object { $_.MainWindowTitle -like '*%s*' }).MainWindowHandle;
if (-not $hwnd) { Write-Error 'no window'; exit 1 }
$rect = New-Object RECT;
[Win32]::GetWindowRect($hwnd, [ref]$rect);
$w = $rect.Right - $rect.Left;
$h = $rect.Bottom - $rect.Top;
if ($w -le 0 -or $h -le 0) { Write-Error 'invalid window rect'; exit 1 }
$bmp = New-Object System.Drawing.Bitmap $w, $h;
$gfx = [System.Drawing.Graphics]::FromImage($bmp);
$gfx.CopyFromScreen($rect.Left, $rect.Top, 0, 0, [System.Drawing.Size]::new($w, $h));
$ms = New-Object System.IO.MemoryStream;
$bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png);
$ms.Close();
[System.Convert]::ToBase64String($ms.ToArray())
`, winTitle)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		writeError(w, 500, "screenshot failed: "+out.String())
		return
	}
	b64 := strings.TrimSpace(out.String())
	writeJSON(w, map[string]interface{}{
		"ok":       true,
		"image":    b64,
		"mime":     "image/png",
		"encoding": "base64",
	})
}

func (b *browser) handleHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var body struct{ JS string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return
	}
	b.syncExec(body.JS)
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (b *browser) handleUnhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	b.syncExec(`
if(window.__origFetch)window.fetch=window.__origFetch;
if(window.__origXHR)XMLHttpRequest=window.__origXHR;
if(window.__origWS)WebSocket=window.__origWS;
if(window.__origES)EventSource=window.__origES;
window.__mbHooks=false`)
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (b *browser) handleBench(w http.ResponseWriter, r *http.Request) {
	b.benchMu.Lock()
	pts := make([]benchPoint, len(b.bench))
	copy(pts, b.bench)
	b.benchMu.Unlock()

	type benchEntry struct {
		Label string `json:"label"`
		Ms    int64  `json:"ms"`
	}
	var entries []benchEntry
	for _, p := range pts {
		entries = append(entries, benchEntry{
			Label: p.label,
			Ms:    p.t.UnixMilli(),
		})
	}
	writeJSON(w, map[string]interface{}{"ok": true, "points": entries})
}

func (b *browser) handleInfo(w http.ResponseWriter, r *http.Request) {
	_, err := b.syncEval("1", 5*time.Second)
	pageReady := err == nil

	var currURL string
	if pageReady {
		res, _ := b.syncEval("location.href", 5*time.Second)
		currURL = strings.Trim(res, `"`)
	}

	b.mu.Lock()
	historySize := len(b.history)
	historyIdx := b.idx
	canBack := b.idx > 0
	canFwd := b.idx < len(b.history)-1
	b.mu.Unlock()

	writeJSON(w, map[string]interface{}{
		"ok":           true,
		"currentURL":   currURL,
		"pageReady":    pageReady,
		"historySize":  historySize,
		"historyIdx":   historyIdx,
		"canGoBack":    canBack,
		"canGoForward": canFwd,
		"apiPort":      b.apiPort,
	})
}

// ---------------------------------------------------------------------------
// Navigation helpers

func navTo(b *browser, urlStr string) {
	b.benchLog("navTo start")
	if urlStr == "hyperspeed://console" || urlStr == "" {
		b.w.Navigate("data:text/html," + url.PathEscape(b.startHTML))
	} else {
		b.w.Navigate(urlStr)
	}
	b.benchLog("w.Navigate called")
	// Inject nav bar on every navigation (guard in JS prevents duplicates)
	go func() {
		time.Sleep(800 * time.Millisecond)
		b.w.Dispatch(func() {
			b.benchLog("navBarJS eval start")
			b.w.Eval(navBarJS)
			b.benchLog("navBarJS eval done")
		})
	}()
}

func (b *browser) navigate(rawURL string) {
	b.bench = nil
	b.benchLog("navigate")
	u := normalizeURL(rawURL)
	urlStr := u.String()
	log.Printf("[NAV] Navigate: %s", urlStr)
	b.mu.Lock()
	if b.curr == urlStr {
		b.mu.Unlock()
		return
	}
	if b.idx < len(b.history)-1 {
		b.history = b.history[:b.idx+1]
	}
	b.history = append(b.history, urlStr)
	if len(b.history) > 100 {
		b.history = b.history[len(b.history)-100:]
		b.idx = len(b.history) - 1
	} else {
		b.idx = len(b.history) - 1
	}
	b.curr = urlStr
	
	// Track non-console URLs for resume
	if urlStr != "hyperspeed://console" {
		b.lastBrowseURL = urlStr
		if len(b.browseHistory) == 0 || b.browseHistory[len(b.browseHistory)-1] != urlStr {
			b.browseHistory = append(b.browseHistory, urlStr)
			if len(b.browseHistory) > 10 {
				b.browseHistory = b.browseHistory[1:]
			}
		}
	}
	// Adapt: classify site and disable unnecessary engines
	if b.opt != nil && b.opt.adapt != nil {
		b.opt.adapt.OnNavigate(urlStr)
	}

	b.mu.Unlock()
	b.w.Dispatch(func() { navTo(b, urlStr) })
}

func (b *browser) goBack() {
	b.bench = nil
	b.benchLog("goBack")
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.idx > 0 {
		b.idx--
		b.curr = b.history[b.idx]
		log.Printf("[NAV] Back: idx=%d url=%s", b.idx, b.curr)
		b.w.Dispatch(func() { navTo(b, b.curr) })
	}
}

func (b *browser) goForward() {
	b.bench = nil
	b.benchLog("goForward")
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.idx < len(b.history)-1 {
		b.idx++
		b.curr = b.history[b.idx]
		log.Printf("[NAV] Forward: idx=%d url=%s", b.idx, b.curr)
		b.w.Dispatch(func() { navTo(b, b.curr) })
	}
}

func (b *browser) reload() {
	if b.curr != "" {
		log.Printf("[NAV] Reload: %s", b.curr)
		b.w.Dispatch(func() { navTo(b, b.curr) })
	}
}

func (b *browser) handleBrowseLast(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	last := b.lastBrowseURL
	b.mu.Unlock()
	if last == "" {
		last = "https://google.com"
	}
	writeJSON(w, map[string]interface{}{"ok": true, "url": last})
}

func (b *browser) handleBrowseHistory(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	hist := make([]string, len(b.browseHistory))
	copy(hist, b.browseHistory)
	b.mu.Unlock()
	writeJSON(w, map[string]interface{}{"ok": true, "history": hist})
}

func (b *browser) handleNDFStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.ndf == nil {
		writeError(w, 503, "NDF not init")
		return
	}
	stats := b.opt.ndf.Stats()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": stats})
}

func (b *browser) handleNDFClear(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.ndf == nil {
		writeError(w, 503, "NDF not init")
		return
	}
	b.opt.ndf.Clear()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "NDF cache cleared"})
}

func (b *browser) handleAutoTuneProfiles(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.autotune == nil {
		writeError(w, 503, "AutoTune not init")
		return
	}
	profiles := b.opt.autotune.AllProfiles()
	writeJSON(w, map[string]interface{}{"ok": true, "profiles": profiles})
}

func (b *browser) handleAutoTuneMetrics(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.autotune == nil {
		writeError(w, 503, "AutoTune not init")
		return
	}
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var req struct {
		Domain string  `json:"domain"`
		CPU    float64 `json:"cpu"`
		Memory float64 `json:"memory"`
		Network float64 `json:"network"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	b.opt.autotune.RecordMetrics(req.Domain, req.CPU, req.Memory, req.Network)
	rec := b.opt.autotune.Recommend(req.Domain)
	writeJSON(w, map[string]interface{}{"ok": true, "recommendation": rec})
}

func (b *browser) handleGCStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.gcctl == nil {
		writeError(w, 503, "GC controller not init")
		return
	}
	stats := b.opt.gcctl.Stats()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": stats})
}

func normalizeURL(raw string) *url.URL {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "console" {
		return &url.URL{Scheme: "hyperspeed", Host: "console"}
	}
	if raw == "hyperspeed://console" {
		return &url.URL{Scheme: "hyperspeed", Host: "console"}
	}
	if !strings.Contains(raw, "://") && !strings.HasPrefix(raw, "about:") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return &url.URL{Scheme: "https", Host: "google.com", RawQuery: "q=" + url.QueryEscape(raw)}
	}
	return u
}

func (b *browser) lazyStartEngines() {
	time.Sleep(2 * time.Second)
	if b.opt == nil {
		return
	}
	engines := []func(){
		b.opt.qse.Start,
		b.opt.dna.Start,
		b.opt.hbm.Start,
		b.opt.avp.Start,
		b.opt.domCompress.Start,
		b.opt.ncg.Start,
		b.opt.pce.Start,
		b.opt.upm.Start,
		b.opt.dra.Start,
		b.opt.mcs.Start,
		b.opt.cbl.Start,
		b.opt.uee.Start,
		b.opt.hfs.Start,
		b.opt.rcm.Start,
		b.opt.autotune.Start,
	}
	for _, fn := range engines {
		go fn()
	}
}
