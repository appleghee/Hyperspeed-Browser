package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	webview "github.com/webview/webview_go"
)

//go:embed popup-blocker.js
var popupBlockerJS string

//go:embed optimizer-gui.js
var optimizerGUIJS string

// minimized toolbar JS (~1KB)
const toolbarJS = `(function(){
var h='<div id="__mb_bar" style="position:fixed;top:0;left:0;right:0;height:32px;z-index:2147483647;background:#1e1e1e;display:flex;align-items:center;padding:4px 8px;font-family:Segoe UI,sans-serif;box-shadow:0 2px 4px rgba(0,0,0,.5);box-sizing:border-box;gap:4px;border-bottom:1px solid #333"><button id="__mb_b" style="background:none;border:none;color:#aaa;font-size:16px;cursor:pointer;width:28px;height:24px;border-radius:3px">\u2039</button><button id="__mb_f" style="background:none;border:none;color:#aaa;font-size:16px;cursor:pointer;width:28px;height:24px;border-radius:3px">\u203A</button><button id="__mb_r" style="background:none;border:none;color:#aaa;font-size:16px;cursor:pointer;width:28px;height:24px;border-radius:3px">\u21bb</button><input id="__mb_u" style="flex:1;height:24px;padding:0 8px;background:#2d2d2d;border:1px solid #444;border-radius:3px;color:#ddd;font-size:13px;outline:none;min-width:0" placeholder="Nhap URL..."></div>';
document.documentElement.insertAdjacentHTML('beforeend',h);
function ns(){try{return JSON.parse(window.name||'{}')}catch(e){return{}}}
var o=new MutationObserver(function(){
var u=document.getElementById('__mb_u'),b=document.getElementById('__mb_b'),f=document.getElementById('__mb_f'),r=document.getElementById('__mb_r');
if(!u)return;
o.disconnect();
var s=ns();
u.onkeydown=function(e){if(e.key=='Enter'&&u.value.trim())window.goNavigate(u.value.trim())};u.value=location.href;
if(b){b.disabled=!s.b;b.onclick=function(){window.goBack()}}
if(f){f.disabled=!s.f;f.onclick=function(){window.goForward()}}
if(r)r.onclick=function(){window.goReload()};
});
o.observe(document.documentElement,{childList:true,subtree:true});
})();`

// runtime intercept JS - hooks fetch, XHR, WebSocket, EventSource for deep capture
const runtimeJS = `(function(){
if(window.__mbHooks)return;
window.__mbHooks=true;
var L=[],WL=[],SL=[];
window.__networkLog=L;window.__wsLog=WL;window.__sseLog=SL;
var _f=window.fetch,_X=XMLHttpRequest,_W=WebSocket,_E=EventSource;
function tr(s,m){return s&&typeof s=='string'?s.length<=m?s:s.slice(0,m)+' [truncated]':s}
function rl(r,p){r.status=p.status;r.statusText=p.statusText;r.endTime=Date.now();r.contentType=p.headers.get('content-type')||'';
r.responseHeaders={};p.headers.forEach(function(v,k){r.responseHeaders[k]=v});
var ct=r.contentType;if(ct&&ct.match(/json|text|html|xml|javascript/)){
var c=p.clone();c.text().then(function(t){r.responseBody=tr(t,10240);r.bodyLength=t.length})['catch'](function(){r.responseBody='[body read failed]'})}}
window.fetch=function(u,o){var r={url:(typeof u=='string'?u:(u&&u.url)||''),method:(o&&o.method)||'GET',requestBody:(o&&o.body)?String(o.body):null,type:'fetch',startTime:Date.now()};L.push(r);
return _f.call(this,u,o).then(function(p){rl(r,p);return p})['catch'](function(e){r.error=e.message;r.endTime=Date.now();throw e})};
window.XMLHttpRequest=function(){var x=new _X(),r={type:'xhr',startTime:Date.now()};L.push(r);
var o=x.open.bind(x);x.open=function(m,u){r.method=m;r.url=u;return o(m,u)};
var s=x.send.bind(x);x.send=function(b){r.requestBody=b?String(b):null;r.startTime=Date.now();
x.addEventListener('readystatechange',function(){if(x.readyState==4){
r.status=x.status;r.statusText=x.statusText;r.endTime=Date.now();r.contentType=x.getResponseHeader('content-type')||'';
try{var t=x.responseText;if(t){r.responseBody=tr(t,10240);r.bodyLength=t.length}}catch(e){}}});
return s(b)};return x};
window.WebSocket=function(url,p){var ws=new _W(url,p),en={url:url,type:'websocket',messages:[],readyState:ws.readyState};WL.push(en);
var s=ws.send.bind(ws);ws.send=function(d){en.messages.push({direction:'outgoing',payload:String(d),time:Date.now()});return s(d)};
ws.addEventListener('open',function(){en.readyState=ws.readyState});
ws.addEventListener('message',function(e){en.messages.push({direction:'incoming',payload:String(e.data),time:Date.now()})});
ws.addEventListener('close',function(){en.readyState=ws.readyState});return ws};
window.EventSource=function(url,c){var es=new _E(url,c),en={url:url,type:'eventsource',messages:[],readyState:es.readyState};SL.push(en);
es.addEventListener('open',function(){en.readyState=es.readyState});
es.addEventListener('message',function(e){en.messages.push({event:e.type,data:String(e.data),time:Date.now()})});
es.addEventListener('error',function(){en.readyState=es.readyState});return es};
try{!function(){
window.__turboState='started';
var B=['google-analytics.com','googletagmanager.com','googleadservices.com','pagead2.googlesyndication.com','doubleclick.net','adservice.google.com'];
function m(u){return u?B.some(function(b){return u.indexOf(b)>=0}):false}
var of=window.fetch;window.fetch=function(i,o){var u=typeof i=='string'?i:(i&&i.url)||'';return m(u)?Promise.resolve(new Response('',{status:204})):of.call(this,i,o)};
window.__turboState='fetch ok';
var Ox=XMLHttpRequest;XMLHttpRequest=function(){var x=new Ox(),bl=false;var op=x.open.bind(x);x.open=function(m,u){bl=m(u);if(!bl)op(m,u)};var sd=x.send.bind(x);x.send=function(b){if(!bl)sd(b)};return x};
var os=Storage.prototype.setItem;Storage.prototype.setItem=function(k,v){if(k[0]=='_')return;return os.call(this,k,v)};
window.__turboState='hooks ok'}();
}catch(e){window.__turboErr=String(e)}
})()`

type browser struct {
	w       webview.WebView
	history []string
	idx     int
	curr    string

	apiPort int

	mu       sync.Mutex
	evalID   int
	evalReqs map[int]chan string
	srv      *http.Server
	portFile string

	opt *Optimizer
}

func main() {
	w := webview.New(false)
	defer w.Destroy()

	w.SetTitle("Mini Browser")
	w.SetSize(1024, 768, webview.HintNone)

	startURL := "https://www.google.com"
	app := &browser{
		w:        w,
		history:  []string{startURL},
		idx:      0,
		curr:     startURL,
		evalReqs: make(map[int]chan string),
	}

	app.opt = NewOptimizer(app)
	TuneCompiler("balanced")

	must(w.Bind("goNavigate", app.navigate))
	must(w.Bind("goBack", app.goBack))
	must(w.Bind("goForward", app.goForward))
	must(w.Bind("goReload", app.reload))
	must(w.Bind("__evalCb", app.evalCallback))

	apiReady := make(chan struct{})
	go app.startAPI(apiReady)
	<-apiReady
	w.SetTitle(fmt.Sprintf("Hyperspeed Browser [:%d]", app.apiPort))
	w.Init(fmt.Sprintf(`window.__mbPort=%d;`, app.apiPort))

	w.Init(toolbarJS)
	w.Init(runtimeJS)
	w.Init(popupBlockerJS)
	w.Init(optimizerInitJS)
	w.Init(optimizerGUIJS)
	w.Navigate(app.curr)
	go app.injectTurboLoop()
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
var s=document.createElement('style');s.textContent='a.ads-banner,.banner-content,.banner-left,.banner-right,.banner-text,.banner-title,.banner-subtitle,#ad_info,.community-banner,#top-banner-info-container,#catfish-banner-info-container,.popup{display:none!important}';document.head.appendChild(s);
var m=document.createElement('meta');m.httpEquiv='Content-Security-Policy';m.content="script-src 'self' 'unsafe-inline' 'unsafe-eval' *.truyenqqko.com *.hinhhinh.com *.tintruyen.net ajax.googleapis.com *.gstatic.com; img-src *.truyenqqko.com *.hinhhinh.com data:; connect-src *; font-src * data:; frame-src 'none';";document.head.appendChild(m);
var ka=function(){var e=document.querySelectorAll('a.ads-banner,.banner-content,.banner-left,.banner-right,.banner-text,.banner-title,.banner-subtitle,#ad_info,.community-banner,#top-banner-info-container,#catfish-banner-info-container,.popup');for(var i=0;i<e.length;i++){if(e[i]&&e[i].parentNode)e[i].remove()}};ka();
(new MutationObserver(function(){ka()})).observe(document.body,{childList:true,subtree:true});
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
	for i := 0; ; i++ {
		b.w.Dispatch(func() {
			b.w.Eval(turboDOM)
			b.w.Eval(spbUI)
			if i%2 == 0 {
				b.w.Eval(spbUICode)
			}
		})
		if i < 10 {
			time.Sleep(200 * time.Millisecond)
		} else {
			time.Sleep(3 * time.Second)
		}
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

	b.portFile = filepath.Join(os.TempDir(), "mini-browser.port")
	os.WriteFile(b.portFile, []byte(fmt.Sprintf("%d", b.apiPort)), 0644)
	log.Printf("[api] listening on 127.0.0.1:%d (port file: %s)", b.apiPort, b.portFile)
	close(ready)

	mux := http.NewServeMux()
	mux.HandleFunc("/api", b.handleAPIRoot)

	// navigation
	mux.HandleFunc("/api/navigate", b.handleNavigate)
	mux.HandleFunc("/api/back", b.handleBack)
	mux.HandleFunc("/api/forward", b.handleForward)
	mux.HandleFunc("/api/reload", b.handleReload)

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

	b.srv = &http.Server{Handler: corsMiddleware(mux)}
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
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
	w.WriteHeader(code)
	writeJSON(w, map[string]interface{}{"ok": false, "error": msg})
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
	ch := make(chan string, 1)
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
		return result, nil
	case <-time.After(timeout):
		b.mu.Lock()
		delete(b.evalReqs, id)
		b.mu.Unlock()
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

// ---------------------------------------------------------------------------
// API handlers

func (b *browser) handleAPIRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"ok":      true,
		"name":    "Mini Browser Deep Inspect API",
		"version": "2.1",
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
	winTitle := "Mini Browser"
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
	b.syncExec(`window.fetch=window.__mbFetch||window.__origFetch;XMLHttpRequest=window.__origXHR;WebSocket=window.__origWS;EventSource=window.__origES`)
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (b *browser) handleInfo(w http.ResponseWriter, r *http.Request) {
	_, err := b.syncEval("1", 5*time.Second)
	pageReady := err == nil

	var currURL string
	if pageReady {
		res, _ := b.syncEval("location.href", 5*time.Second)
		currURL = strings.Trim(res, `"`)
	}

	writeJSON(w, map[string]interface{}{
		"ok":           true,
		"currentURL":   currURL,
		"pageReady":    pageReady,
		"historySize":  len(b.history),
		"historyIdx":   b.idx,
		"canGoBack":    b.idx > 0,
		"canGoForward": b.idx < len(b.history)-1,
		"apiPort":      b.apiPort,
	})
}

// ---------------------------------------------------------------------------
// Navigation helpers

func (b *browser) navigate(rawURL string) {
	u := normalizeURL(rawURL)
	urlStr := u.String()
	if b.curr == urlStr {
		return
	}
	if b.idx < len(b.history)-1 {
		b.history = b.history[:b.idx+1]
	}
	b.history = append(b.history, urlStr)
	b.idx = len(b.history) - 1
	b.curr = urlStr
	navJSON := fmt.Sprintf(`{"b":%t,"f":false}`, b.idx > 0)
	b.w.Dispatch(func() {
		b.w.Eval("window.name='" + navJSON + "'")
		b.w.Navigate(urlStr)
	})
}

func (b *browser) goBack() {
	if b.idx > 0 {
		b.idx--
		b.curr = b.history[b.idx]
		navJSON := fmt.Sprintf(`{"b":%t,"f":%t}`, b.idx > 0, b.idx < len(b.history)-1)
		b.w.Dispatch(func() {
			b.w.Eval("window.name='" + navJSON + "'")
			b.w.Navigate(b.curr)
		})
	}
}

func (b *browser) goForward() {
	if b.idx < len(b.history)-1 {
		b.idx++
		b.curr = b.history[b.idx]
		navJSON := fmt.Sprintf(`{"b":%t,"f":%t}`, b.idx > 0, b.idx < len(b.history)-1)
		b.w.Dispatch(func() {
			b.w.Eval("window.name='" + navJSON + "'")
			b.w.Navigate(b.curr)
		})
	}
}

func (b *browser) reload() {
	if b.curr != "" {
		b.w.Dispatch(func() { b.w.Navigate(b.curr) })
	}
}

func normalizeURL(raw string) *url.URL {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &url.URL{Scheme: "https", Host: "google.com"}
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
