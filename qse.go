package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type QSEEngine struct {
	b       *browser
	mu      sync.RWMutex
	enabled bool
	shortcuts map[string]string
	clickHeat map[string]int
	stats    QSEStats
}

type QSEStats struct {
	Shortcuts   int    `json:"shortcuts"`
	Resolved    int    `json:"resolved"`
	ClickHeat   int    `json:"clickHeat"`
	TopDomain   string `json:"topDomain"`
	Status      string `json:"status"`
}

var defaultShortcuts = map[string]string{
	"yt":    "https://youtube.com/results?search_query=%s",
	"gh":    "https://github.com/search?q=%s",
	"w":     "https://en.wikipedia.org/wiki/%s",
	"g":     "https://www.google.com/search?q=%s",
	"reddit": "https://www.reddit.com/search/?q=%s",
	"tw":    "https://twitter.com/search?q=%s",
	"x":     "https://x.com/search?q=%s",
	"amz":   "https://www.amazon.com/s?k=%s",
	"so":    "https://stackoverflow.com/search?q=%s",
	"maps": "https://www.google.com/maps/search/%s",
	"img":   "https://www.google.com/search?tbm=isch&q=%s",
	"news":  "https://news.google.com/search?q=%s",
	"npm":   "https://www.npmjs.com/search?q=%s",
	"pypi":  "https://pypi.org/search/?q=%s",
	"wiki":  "https://en.wikipedia.org/wiki/%s",
}

const qseClickHeatJS = `(function(){
if(window.__mbQSE)return;
var Q=window.__mbQSE={clicks:{}};
try{var s=localStorage.getItem('__qse_clicks');if(s)Q.clicks=JSON.parse(s);}catch(e){}
document.addEventListener('click',function(e){
var a=e.target.closest('a');
if(!a||!a.href)return;
try{
var u=new URL(a.href);
var d=u.hostname.replace('www.','');
Q.clicks[d]=(Q.clicks[d]||0)+1;
try{localStorage.setItem('__qse_clicks',JSON.stringify(Q.clicks));}catch(ex){}
}catch(ex){}
},true);
})()`

func NewQSEEngine(b *browser) *QSEEngine {
	q := &QSEEngine{
		b:         b,
		enabled:   true,
		shortcuts: make(map[string]string),
		clickHeat: make(map[string]int),
	}
	for k, v := range defaultShortcuts {
		q.shortcuts[k] = v
	}
	return q
}

func (q *QSEEngine) Start() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.enabled {
		return
	}
	q.b.syncExec(qseClickHeatJS)
}

func (q *QSEEngine) Resolve(input string) (string, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	input = strings.TrimSpace(input)
	parts := strings.SplitN(input, " ", 2)
	if len(parts) < 2 {
		return "", false
	}
	key := strings.ToLower(parts[0])
	query := strings.TrimSpace(parts[1])
	tpl, ok := q.shortcuts[key]
	if !ok {
		return "", false
	}
	q.stats.Resolved++
	return strings.ReplaceAll(tpl, "%s", query), true
}

func (q *QSEEngine) GatherClickHeat() {
	q.mu.Lock()
	defer q.mu.Unlock()
	var clicks map[string]int
	if err := q.b.syncUnwrapInto("(function(){var q=window.__mbQSE;if(!q)return{};try{return q.clicks;}catch(e){return{};};})()", 5*time.Second, &clicks); err != nil {
		return
	}
	// Cap at 500 domains — evict latest merged entries if full
	for d, c := range clicks {
		if len(q.clickHeat) >= 500 {
			break
		}
		q.clickHeat[d] = c
	}
}

func (q *QSEEngine) Stats() *QSEStats {
	q.mu.RLock()
	defer q.mu.RUnlock()
	q.stats.Shortcuts = len(q.shortcuts)
	q.stats.ClickHeat = len(q.clickHeat)
	var top string
	var maxC int
	for d, c := range q.clickHeat {
		if c > maxC {
			maxC = c
			top = d
		}
	}
	q.stats.TopDomain = top
	q.stats.Status = "ok"
	return &q.stats
}

func (b *browser) handleQSEStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.qse == nil {
		writeError(w, 503, "QSE not init")
		return
	}
	s := b.opt.qse.Stats()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}

func (b *browser) handleQSEStart(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.qse == nil {
		writeError(w, 503, "QSE not init")
		return
	}
	b.opt.qse.Start()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "QSE click heat started"})
}

func (b *browser) handleQSEAdd(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.qse == nil {
		writeError(w, 503, "QSE not init")
		return
	}
	var req struct {
		Key  string `json:"key"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	req.Key = strings.TrimSpace(strings.ToLower(req.Key))
	if req.Key == "" {
		writeError(w, 400, "key required")
		return
	}
	b.opt.qse.mu.Lock()
	if len(b.opt.qse.shortcuts) >= 100 {
		for k := range b.opt.qse.shortcuts {
			delete(b.opt.qse.shortcuts, k)
			break
		}
	}
	b.opt.qse.shortcuts[req.Key] = req.URL
	b.opt.qse.mu.Unlock()
	writeJSON(w, map[string]interface{}{"ok": true, "shortcuts": len(b.opt.qse.shortcuts)})
}
