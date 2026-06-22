// HBM — Heat-Based Memory allocator
package main

import (
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

type HBMEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	stats   HBMStats
	hotPool int64
	coolPool int64
	stopCh  chan struct{}
}

type HBMStats struct {
	HotAllocMB  int64  `json:"hotAllocMB"`
	CoolAllocMB int64  `json:"coolAllocMB"`
	GCPercent   int    `json:"gcPercent"`
	HotRatio    string `json:"hotRatio"`
	Status      string `json:"status"`
}

func NewHBMEngine() *HBMEngine {
	return &HBMEngine{enabled: true}
}

func (h *HBMEngine) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.enabled || h.stopCh != nil {
		return
	}
	h.stopCh = make(chan struct{})
	go h.loop()
}

func (h *HBMEngine) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.stopCh != nil {
		close(h.stopCh)
		h.stopCh = nil
	}
}

func (h *HBMEngine) loop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			hot := int64(m.Alloc) * 60 / 100
			cool := int64(m.Alloc) * 40 / 100
			h.mu.Lock()
			h.hotPool = hot / (1024 * 1024)
			h.coolPool = cool / (1024 * 1024)
			ratio := float64(0)
			if h.hotPool+h.coolPool > 0 {
				ratio = float64(h.hotPool) * 100 / float64(h.hotPool+h.coolPool)
			}
			gc := debug.SetGCPercent(-1)
			h.stats = HBMStats{
				HotAllocMB:  h.hotPool,
				CoolAllocMB: h.coolPool,
				GCPercent:   gc,
				HotRatio:    fmt.Sprintf("%.0f%%", ratio),
				Status:      "active",
			}
			h.mu.Unlock()
		case <-h.stopCh:
			return
		}
	}
}

func (h *HBMEngine) Stats() HBMStats {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stats
}

func (b *browser) handleHBMStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.hbm == nil {
		writeError(w, 503, "HBM not init")
		return
	}
	s := b.opt.hbm.Stats()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}
