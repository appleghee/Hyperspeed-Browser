// NCG — Network Cost Graph
package main

import (
	"net/http"
	"sort"
	"sync"
	"time"
)

type NCGEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	stats   NCGStats
	costs   map[string]*NCGEntry
	maxDomains int
}

type NCGEntry struct {
	Domain    string
	TotalSize int64
	Requests  int
	AvgLatency time.Duration
	LastSeen  time.Time
}

type NCGStats struct {
	Tracked   int     `json:"tracked"`
	TotalSize int64   `json:"totalSizeKB"`
	TotalReqs int     `json:"totalReqs"`
	TopDomain string  `json:"topDomain"`
	Status    string  `json:"status"`
}

func NewNCGEngine(b *browser) *NCGEngine {
	return &NCGEngine{b: b, enabled: true, costs: make(map[string]*NCGEntry), maxDomains: 500}
}

func (n *NCGEngine) Start() {}

func (n *NCGEngine) Track(domain string, size int64, latency time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	e, ok := n.costs[domain]
	if !ok {
		if len(n.costs) >= n.maxDomains {
			n.evictOldest()
		}
		e = &NCGEntry{Domain: domain}
		n.costs[domain] = e
	}
	e.TotalSize += size
	e.Requests++
	e.AvgLatency = (e.AvgLatency*time.Duration(e.Requests-1) + latency) / time.Duration(e.Requests)
	e.LastSeen = time.Now()
}

func (n *NCGEngine) evictOldest() {
	var oldest string
	var oldestTime time.Time
	first := true
	for k, e := range n.costs {
		if first || e.LastSeen.Before(oldestTime) {
			oldest = k
			oldestTime = e.LastSeen
			first = false
		}
	}
	if oldest != "" {
		delete(n.costs, oldest)
	}
}

func (n *NCGEngine) Stats() NCGStats {
	n.mu.Lock()
	defer n.mu.Unlock()
	s := NCGStats{Tracked: len(n.costs), Status: "active"}
	var top string
	var maxSize int64
	for _, e := range n.costs {
		s.TotalSize += e.TotalSize
		s.TotalReqs += e.Requests
		if e.TotalSize > maxSize {
			maxSize = e.TotalSize
			top = e.Domain
		}
	}
	s.TopDomain = top
	return s
}

func (n *NCGEngine) HeavyDomains(limit int) []NCGEntry {
	n.mu.Lock()
	defer n.mu.Unlock()
	var list []NCGEntry
	for _, e := range n.costs {
		list = append(list, *e)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalSize > list[j].TotalSize
	})
	if len(list) > limit {
		list = list[:limit]
	}
	return list
}

func (b *browser) handleNCGStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.ncg == nil {
		writeError(w, 503, "NCG not init")
		return
	}
	s := b.opt.ncg.Stats()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}
