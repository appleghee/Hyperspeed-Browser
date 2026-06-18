package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type HeatEntry struct {
	Key        string    `json:"key"`
	Kind       string    `json:"kind"`
	Heat       float64   `json:"heat"`
	Accesses   uint64    `json:"accesses"`
	LastAccess time.Time `json:"-"`
	Created    time.Time `json:"-"`
}

type UHEngine struct {
	mu         sync.RWMutex
	enabled    bool
	entries    map[string]*HeatEntry
	decayRate  float64
	accessHeat float64
	maxHeat    float64
	coolThresh float64
	hotThresh  float64
	stopCh     chan struct{}
	stats      UHEStats
}

type UHEStats struct {
	Total   int                `json:"total"`
	Hot     int                `json:"hot"`
	Warm    int                `json:"warm"`
	Cool    int                `json:"cool"`
	Dom     int                `json:"dom"`
	Script  int                `json:"script"`
	Cache   int                `json:"cache"`
	Network int                `json:"network"`
	Image   int                `json:"image"`
	Tab     int                `json:"tab"`
	Top5    []HeatSummary      `json:"top5"`
	Status  string             `json:"status"`
}

type HeatSummary struct {
	Key  string  `json:"key"`
	Kind string  `json:"kind"`
	Heat float64 `json:"heat"`
}

func NewUHEngine() *UHEngine {
	return &UHEngine{
		entries:    make(map[string]*HeatEntry),
		decayRate:  0.02,
		accessHeat: 0.15,
		maxHeat:    1.0,
		coolThresh: 0.15,
		hotThresh:  0.6,
		enabled:    true,
	}
}

func (u *UHEngine) Start() {
	u.mu.Lock()
	defer u.mu.Unlock()
	if !u.enabled || u.stopCh != nil {
		return
	}
	u.stopCh = make(chan struct{})
	go u.loop()
}

func (u *UHEngine) Stop() {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.stopCh != nil {
		close(u.stopCh)
		u.stopCh = nil
	}
}

func (u *UHEngine) SetEnabled(v bool) {
	u.mu.Lock()
	u.enabled = v
	u.mu.Unlock()
	if v {
		u.Start()
	} else {
		u.Stop()
	}
}

func (u *UHEngine) loop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			u.decayAll()
		case <-u.stopCh:
			return
		}
	}
}

func (u *UHEngine) decayAll() {
	u.mu.Lock()
	defer u.mu.Unlock()
	now := time.Now()
	for _, e := range u.entries {
		elapsed := now.Sub(e.LastAccess).Seconds()
		if elapsed < 1 {
			continue
		}
		decay := u.decayRate * elapsed
		e.Heat = max(0, e.Heat-decay)
	}
}

func (u *UHEngine) Access(key, kind string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	e, ok := u.entries[key]
	if !ok {
		e = &HeatEntry{
			Key:        key,
			Kind:       kind,
			Heat:       0,
			Accesses:   0,
			LastAccess: time.Now(),
			Created:    time.Now(),
		}
		u.entries[key] = e
	}
	e.Heat = min(u.maxHeat, e.Heat+u.accessHeat)
	e.Accesses++
	e.LastAccess = time.Now()
}

func (u *UHEngine) getHeat(key string) float64 {
	u.mu.RLock()
	defer u.mu.RUnlock()
	if e, ok := u.entries[key]; ok {
		return e.Heat
	}
	return 0
}

func (u *UHEngine) IsHot(key string) bool {
	return u.getHeat(key) >= u.hotThresh
}

func (u *UHEngine) IsCool(key string) bool {
	return u.getHeat(key) <= u.coolThresh
}

func (u *UHEngine) HotCount() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	c := 0
	for _, e := range u.entries {
		if e.Heat >= u.hotThresh {
			c++
		}
	}
	return c
}

func (u *UHEngine) TopN(n int) []HeatEntry {
	u.mu.RLock()
	defer u.mu.RUnlock()
	if len(u.entries) == 0 {
		return nil
	}
	sorted := make([]*HeatEntry, 0, len(u.entries))
	for _, e := range u.entries {
		sorted = append(sorted, e)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Heat > sorted[j].Heat
	})
	if n > len(sorted) {
		n = len(sorted)
	}
	out := make([]HeatEntry, n)
	for i := 0; i < n; i++ {
		out[i] = *sorted[i]
	}
	return out
}

func (u *UHEngine) Gather() *UHEStats {
	u.mu.RLock()
	defer u.mu.RUnlock()
	s := &UHEStats{
		Total:  len(u.entries),
		Status: "active",
	}
	for _, e := range u.entries {
		if e.Heat >= u.hotThresh {
			s.Hot++
		} else if e.Heat > u.coolThresh {
			s.Warm++
		} else {
			s.Cool++
		}
		switch e.Kind {
		case "dom":
			s.Dom++
		case "script":
			s.Script++
		case "cache":
			s.Cache++
		case "network":
			s.Network++
		case "image":
			s.Image++
		case "tab":
			s.Tab++
		}
	}
	top := u.TopN(5)
	for _, t := range top {
		if len(s.Top5) < 5 {
			s.Top5 = append(s.Top5, HeatSummary{Key: t.Key, Kind: t.Kind, Heat: t.Heat})
		}
	}
	return s
}

func (u *UHEngine) RemoveStale(olderThan time.Duration) int {
	u.mu.Lock()
	defer u.mu.Unlock()
	cutoff := time.Now().Add(-olderThan)
	count := 0
	for k, e := range u.entries {
		if e.LastAccess.Before(cutoff) && e.Heat < u.coolThresh {
			delete(u.entries, k)
			count++
		}
	}
	return count
}

func (u *UHEngine) Clear() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.entries = make(map[string]*HeatEntry)
}

func (b *browser) handleUHEStart(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.uhe == nil {
		writeError(w, 503, "UHE not init")
		return
	}
	b.opt.uhe.Start()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "UHE started"})
}

func (b *browser) handleUHEStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.uhe == nil {
		writeError(w, 503, "UHE not init")
		return
	}
	s := b.opt.uhe.Gather()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}

func (b *browser) handleUHEAccess(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.uhe == nil {
		writeError(w, 503, "UHE not init")
		return
	}
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var req struct {
		Key  string `json:"key"`
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	b.opt.uhe.Access(req.Key, req.Kind)
	writeJSON(w, map[string]interface{}{"ok": true, "heat": b.opt.uhe.getHeat(req.Key)})
}

func (b *browser) handleUHETop(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.uhe == nil {
		writeError(w, 503, "UHE not init")
		return
	}
	top := b.opt.uhe.TopN(10)
	writeJSON(w, map[string]interface{}{"ok": true, "top": top})
}

func (b *browser) handleUHE(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.uhe == nil {
		writeError(w, 503, "UHE not init")
		return
	}
	switch r.Method {
	case "GET":
		s := b.opt.uhe.Gather()
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
			b.opt.uhe.Start()
			writeJSON(w, map[string]interface{}{"ok": true, "msg": "UHE started"})
		case "stop":
			b.opt.uhe.Stop()
			writeJSON(w, map[string]interface{}{"ok": true, "msg": "UHE stopped"})
		case "clear":
			b.opt.uhe.Clear()
			writeJSON(w, map[string]interface{}{"ok": true, "msg": "UHE cleared"})
		default:
			writeError(w, 400, "unknown action")
		}
	default:
		writeError(w, 405, "use GET or POST")
	}
}

func (u *UHEngine) HandleAPI(prefix string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, prefix)
		switch path {
		case "/start":
			u.Start()
			writeJSON(w, map[string]interface{}{"ok": true, "msg": "UHE started"})
		case "/stats":
			s := u.Gather()
			writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
		case "/access":
			handleJSON(w, r, func(req struct{ Key, Kind string }) {
				u.Access(req.Key, req.Kind)
				writeJSON(w, map[string]interface{}{"ok": true, "heat": u.getHeat(req.Key)})
			})
		default:
			writeJSON(w, map[string]interface{}{"ok": false, "error": "unknown UHE path"})
		}
	})
	return mux
}

func handleJSON[T any](w http.ResponseWriter, r *http.Request, fn func(T)) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	fn(req)
}

func (u *UHEngine) String() string {
	s := u.Gather()
	return fmt.Sprintf("UHE: %d entries (%d hot, %d warm, %d cool)", s.Total, s.Hot, s.Warm, s.Cool)
}
