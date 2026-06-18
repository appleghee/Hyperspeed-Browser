package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed lod.js
var lodJS string

type LODEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	stats   LODStats
}

type LODStats struct {
	Total   int     `json:"total"`
	Level0  int     `json:"level0"`
	Level1  int     `json:"level1"`
	Level2  int     `json:"level2"`
	Level3  int     `json:"level3"`
	Memory  float64 `json:"memoryMB"`
	SavedKB float64 `json:"savedKB"`
	Inflate int     `json:"inflates"`
	Status  string  `json:"status"`
}

var lodGatherJS = `(function(){
var s=window.__mbLOD;if(!s)return{total:0,level0:0,level1:0,level2:0,level3:0,memoryMB:0,savedKB:0,inflates:0,status:'n/a'};
var l=s.levels,r=s.rects;
var total=l.length||0,le0=0,le1=0,le2=0,le3=0;
for(var i=0;i<total;i++){var v=l[i];if(v===0)le0++;else if(v===1)le1++;else if(v===2)le2++;else le3++}
var mem=performance.memory?Math.round(performance.memory.usedJSHeapSize/1048576*10)/10:0;
var kb=0;
for(var k in r){var o=r[k];if(o&&o.len)kb+=o.len}
return{total:total,level0:le0,level1:le1,level2:le2,level3:le3,memoryMB:mem,savedKB:Math.round(kb/1024),inflates:s.inflates||0,status:'active'}
})()`

var lodToggleJS = `(function(){var s=window.__mbLOD;if(!s)return 'no LOD';s.enabled=!s.enabled;return s.enabled?'enabled':'disabled'})()`

func NewLODEngine(b *browser) *LODEngine {
	return &LODEngine{b: b, enabled: true}
}

func (l *LODEngine) Start() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return
	}
	l.b.syncExec(lodJS)
}

func (l *LODEngine) Gather() *LODStats {
	var s LODStats
	if err := l.b.syncUnwrapInto(lodGatherJS, 5*time.Second, &s); err != nil {
		return &l.stats
	}
	l.mu.Lock()
	l.stats = s
	l.mu.Unlock()
	return &s
}

func (l *LODEngine) Toggle() string {
	v, err := l.b.syncUnwrap(lodToggleJS, 5*time.Second)
	if err != nil {
		return "error"
	}
	return fmt.Sprint(v)
}

func (l *LODEngine) HandleAPI(prefix string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, prefix)
		switch path {
		case "/start":
			l.Start()
			writeJSON(w, map[string]interface{}{"ok": true, "msg": "LOD started"})
		case "/stats":
			s := l.Gather()
			writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
		case "/toggle":
			msg := l.Toggle()
			writeJSON(w, map[string]interface{}{"ok": true, "msg": msg})
		default:
			writeJSON(w, map[string]interface{}{"ok": false, "error": "unknown LOD path"})
		}
	})
	return mux
}

func lodHandler(l *LODEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			s := l.Gather()
			writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
		case "POST":
			var req struct {
				Action string `json:"action"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, 400, "bad request")
				return
			}
			switch req.Action {
			case "start":
				l.Start()
				writeJSON(w, map[string]interface{}{"ok": true, "msg": "LOD started"})
			case "toggle":
				msg := l.Toggle()
				writeJSON(w, map[string]interface{}{"ok": true, "msg": msg})
			default:
				writeError(w, 400, "unknown action")
			}
		}
	}
}

func (b *browser) handleLODStart(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.lod == nil {
		writeError(w, 503, "LOD not init")
		return
	}
	b.opt.lod.Start()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "LOD started"})
}

func (b *browser) handleLODStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.lod == nil {
		writeError(w, 503, "LOD not init")
		return
	}
	s := b.opt.lod.Gather()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}

func (b *browser) handleLODToggle(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.lod == nil {
		writeError(w, 503, "LOD not init")
		return
	}
	msg := b.opt.lod.Toggle()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": msg})
}

func (b *browser) handleLOD(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.lod == nil {
		writeError(w, 503, "LOD not init")
		return
	}
	lodHandler(b.opt.lod)(w, r)
}
