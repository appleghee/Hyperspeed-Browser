package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HLRC object kinds
const (
	HLRCKindTab     = "tab"
	HLRCKindDOM     = "dom"
	HLRCKindCache   = "cache"
	HLRCKindNetwork = "network"
	HLRCKindScript  = "script"
	HLRCKindImage   = "image"
	HLRCKindTimer   = "timer"
	HLRCKindFont    = "font"
)

// HLRC LOD levels
const (
	HLRCLODFull    = 0
	HLRCLODReduced = 1
	HLRCLODSkeleton = 2
	HLRCLODCold    = 3
	HLRCLODEvicted = 4
)

// Heat constants (uint8 0-255)
const (
	HLRCHeatMax       = 255
	HLRCHeatDecay     = 8
	HLRCHeatAccess    = 32
	HLRCHeatHotThresh = 180
	HLRCHeatWarmThresh = 80
)

var hlrcLODNames = []string{"full", "reduced", "skeleton", "cold", "evicted"}

// RTObject is the compact runtime object header (16 bytes critical, ~72 total)
type RTObject struct {
	Heat        uint8   `json:"heat"`
	LODLevel    uint8   `json:"lodLevel"`
	Flags       uint16  `json:"flags"`
	LastAccess  uint32  `json:"lastAccessTick"`
	RestoreCost uint32  `json:"restoreCostHint"`
	DepCount    uint16  `json:"depCount"`

	key       string
	kind      string
	createdAt time.Time

	reuseProb  float64
	interactProb float64
	ramCost    int64
	cpuCost    int64
	gpuCost    int64
	netCost    int64
}

const (
	RTFlagActive  = 1 << 0
	RTFlagPinned  = 1 << 1
	RTFlagDirty   = 1 << 2
)

// HLRC is the Heat-LOD Runtime Core
type HLRC struct {
	mu      sync.RWMutex
	enabled bool

	// 256 bucket queue indexed by heat
	buckets [HLRCHeatMax + 1]map[string]int

	// All objects
	objects map[string]*RTObject

	// Conversion from float64 heat (UHE-style) to uint8
	heatScale float64

	tick       uint32
	stopCh     chan struct{}
	tickRate   time.Duration
	lastTick   time.Time

	// Budget targets
	ramBudgetMB  int64
	cpuBudgetPct float64

	// Hysteresis map — object keys with demotion timestamps
	hysteresis map[string]time.Time

	// Hard safety cap to prevent OOM
	maxObjects int

	stats HLRCStats
}

type HLRCStats struct {
	Objects     int    `json:"objects"`
	Tick        uint32 `json:"tick"`
	Enabled     bool   `json:"enabled"`
	RAMBudgetMB int64  `json:"ramBudgetMB"`
	RAMUsedMB   int64  `json:"ramUsedMB"`
	LODCounts   [5]int `json:"lodCounts"`
	Demotions   int64  `json:"demotions"`
	Promotions  int64  `json:"promotions"`
	Evictions   int64  `json:"evictions"`
	Status      string `json:"status"`
}

func NewHLRC() *HLRC {
	h := &HLRC{
		enabled:    true,
		objects:    make(map[string]*RTObject),
		heatScale:  255.0,
		tickRate:   10 * time.Second,
		ramBudgetMB: 50,
		cpuBudgetPct: 80,
		hysteresis: make(map[string]time.Time),
		maxObjects: 2000,
	}
	for i := 0; i <= HLRCHeatMax; i++ {
		h.buckets[i] = make(map[string]int)
	}
	return h
}

func (h *HLRC) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.enabled || h.stopCh != nil {
		return
	}
	h.stopCh = make(chan struct{})
	go h.loop()
}

func (h *HLRC) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.stopCh != nil {
		close(h.stopCh)
		h.stopCh = nil
	}
}

func (h *HLRC) loop() {
	ticker := time.NewTicker(h.tickRate)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			h.tick++
			h.decayAll()
			h.enforceBudgets()
			h.cleanHysteresis()
		case <-h.stopCh:
			return
		}
	}
}

func (h *HLRC) decayAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for _, obj := range h.objects {
		if obj.Flags&RTFlagPinned != 0 {
			continue
		}
		oldHeat := obj.Heat
		if obj.Heat > HLRCHeatDecay {
			obj.Heat -= HLRCHeatDecay
		} else {
			obj.Heat = 0
		}
		if oldHeat != obj.Heat {
			h.moveBucket(obj.key, oldHeat, obj.Heat)
		}
	}
	h.lastTick = now
	h.updateStats()
}

// Register adds a new runtime object to HLRC
func (h *HLRC) Register(key, kind string, restoreCost uint32) *RTObject {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.objects) >= h.maxObjects {
		h.evictColdest(50)
	}

	obj := &RTObject{
		Heat:        0,
		LODLevel:    HLRCLODFull,
		LastAccess:  h.tick,
		RestoreCost: restoreCost,
		key:         key,
		kind:        kind,
		createdAt:   time.Now(),
		reuseProb:   0.5,
		interactProb: 0.5,
		ramCost:   int64(restoreCost),
	}
	if obj.ramCost < 100 {
		obj.ramCost = 100
	}
	h.objects[key] = obj
	h.buckets[0][key] = 1
	return obj
}

// evictColdest removes N coldest entries when object cap is hit
func (h *HLRC) evictColdest(n int) {
	if n < 1 {
		n = 1
	}
	var coldest []string
	for heat := uint8(0); heat <= HLRCHeatMax && len(coldest) < n; heat++ {
		for key := range h.buckets[heat] {
			coldest = append(coldest, key)
			if len(coldest) >= n {
				break
			}
		}
	}
	for _, key := range coldest {
		if obj, ok := h.objects[key]; ok {
			delete(h.buckets[obj.Heat], key)
			delete(h.objects, key)
			delete(h.hysteresis, key)
		}
	}
}

// cleanHysteresis removes hysteresis entries older than 60s
func (h *HLRC) cleanHysteresis() {
	cutoff := time.Now().Add(-60 * time.Second)
	for key, t := range h.hysteresis {
		if t.Before(cutoff) {
			delete(h.hysteresis, key)
		}
	}
}

// Access boosts an object's heat
func (h *HLRC) Access(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	obj, ok := h.objects[key]
	if !ok {
		return
	}
	oldHeat := obj.Heat
	if int(obj.Heat)+HLRCHeatAccess > HLRCHeatMax {
		obj.Heat = HLRCHeatMax
	} else {
		obj.Heat += HLRCHeatAccess
	}
	obj.LastAccess = h.tick
	if oldHeat != obj.Heat {
		h.moveBucket(key, oldHeat, obj.Heat)
	}
}

// UpdateCosts sets utility density parameters for an object
func (h *HLRC) UpdateCosts(key string, reuseProb, interactProb float64, ramCost, cpuCost, gpuCost, netCost int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	obj, ok := h.objects[key]
	if !ok {
		return
	}
	obj.reuseProb = reuseProb
	obj.interactProb = interactProb
	obj.ramCost = ramCost
	obj.cpuCost = cpuCost
	obj.gpuCost = gpuCost
	obj.netCost = netCost
}

// SetLODLevel changes an object's LOD level
func (h *HLRC) SetLODLevel(key string, level uint8) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	obj, ok := h.objects[key]
	if !ok || level > HLRCLODEvicted {
		return false
	}
	obj.LODLevel = level
	return true
}

// SetFlags sets bit flags on an object
func (h *HLRC) SetFlags(key string, flags uint16) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if obj, ok := h.objects[key]; ok {
		obj.Flags = flags
	}
}

// Unregister removes an object
func (h *HLRC) Unregister(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	obj, ok := h.objects[key]
	if !ok {
		return
	}
	delete(h.buckets[obj.Heat], key)
	delete(h.objects, key)
	delete(h.hysteresis, key)
}

// computeUtilityDensity returns (density, low) for an object
// low = true if density is below eviction threshold
func (h *HLRC) computeUtilityDensity(obj *RTObject) (float64, bool) {
	if obj.Flags&RTFlagPinned != 0 {
		return 1.0, false
	}

	expectedValue := float64(obj.Heat) / HLRCHeatMax

	reuseProb := obj.reuseProb
	if reuseProb < 0.01 {
		reuseProb = 0.01
	}
	interactProb := obj.interactProb
	if interactProb < 0.01 {
		interactProb = 0.01
	}

	numerator := expectedValue * reuseProb * interactProb

	totalCost := float64(obj.ramCost+1) +
		0.5*float64(obj.cpuCost+1) +
		0.3*float64(obj.gpuCost+1) +
		0.2*float64(obj.RestoreCost+1)

	density := numerator / totalCost
	return density, density < 0.001
}

// moveBucket moves an object from oldHeat to newHeat bucket
// Caller must hold h.mu.Lock()
func (h *HLRC) moveBucket(key string, oldHeat, newHeat uint8) {
	if oldHeat > HLRCHeatMax {
		oldHeat = HLRCHeatMax
	}
	if newHeat > HLRCHeatMax {
		newHeat = HLRCHeatMax
	}
	delete(h.buckets[oldHeat], key)
	h.buckets[newHeat][key] = 1
}

// enforceBudgets checks RAM and CPU budgets and demotes low-density objects
func (h *HLRC) enforceBudgets() {
	h.mu.Lock()
	defer h.mu.Unlock()

	ramUsed := int64(0)
	for _, obj := range h.objects {
		ramUsed += obj.ramCost
	}
	ramBudget := h.ramBudgetMB * 1024 * 1024

	if ramUsed <= ramBudget {
		return
	}

	excess := ramUsed - ramBudget
	demoted := 0
	now := time.Now()

	// Scan from coldest bucket upward
	for heat := uint8(0); heat <= HLRCHeatMax && excess > 0; heat++ {
		for key := range h.buckets[heat] {
			obj, ok := h.objects[key]
			if !ok || obj.Flags&RTFlagPinned != 0 {
				continue
			}

			// Hysteresis: skip if demoted within last 30s
			if t, ok := h.hysteresis[key]; ok && now.Sub(t) < 30*time.Second {
				continue
			}

			_, low := h.computeUtilityDensity(obj)
			if low {
				if obj.LODLevel < HLRCLODEvicted {
					obj.LODLevel++
					excess -= obj.ramCost / 2
					h.hysteresis[key] = now
					h.stats.Demotions++
					demoted++
				} else {
					delete(h.buckets[obj.Heat], key)
					delete(h.objects, key)
					delete(h.hysteresis, key)
					excess -= obj.ramCost
					h.stats.Evictions++
				}
			}
		}
	}

	h.updateStats()
}

// Promote increases LOD level based on high heat
func (h *HLRC) Promote(key string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	obj, ok := h.objects[key]
	if !ok || obj.LODLevel == 0 {
		return false
	}
	obj.LODLevel--
	h.stats.Promotions++
	return true
}

func (h *HLRC) updateStats() {
	h.stats.Objects = len(h.objects)
	h.stats.Tick = h.tick
	h.stats.Enabled = h.enabled
	h.stats.RAMBudgetMB = h.ramBudgetMB
	h.stats.Status = "active"
	if !h.enabled {
		h.stats.Status = "disabled"
	}

	ramUsed := int64(0)
	for i := 0; i < 5; i++ {
		h.stats.LODCounts[i] = 0
	}
	for _, obj := range h.objects {
		ramUsed += obj.ramCost
		if obj.LODLevel <= HLRCLODEvicted {
			h.stats.LODCounts[obj.LODLevel]++
		}
	}
	h.stats.RAMUsedMB = ramUsed / (1024 * 1024)
}

// Gather returns current stats
func (h *HLRC) Gather() *HLRCStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s := h.stats
	s.LODCounts = [5]int{}
	for _, obj := range h.objects {
		if obj.LODLevel <= HLRCLODEvicted {
			s.LODCounts[obj.LODLevel]++
		}
	}
	return &s
}

// ObjectSnapshot returns all objects as a serializable slice
func (h *HLRC) ObjectSnapshot() []map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]map[string]interface{}, 0, len(h.objects))
	for _, obj := range h.objects {
		density, _ := h.computeUtilityDensity(obj)
		out = append(out, map[string]interface{}{
			"key":         obj.key,
			"kind":        obj.kind,
			"heat":        obj.Heat,
			"lodLevel":    obj.LODLevel,
			"lodLabel":    hlrcLODNames[obj.LODLevel],
			"flags":       obj.Flags,
			"lastAccess":  obj.LastAccess,
			"restoreCost": obj.RestoreCost,
			"depCount":    obj.DepCount,
			"utilityDensity": fmt.Sprintf("%.6f", density),
		})
	}
	return out
}

// SetConfig updates runtime budgets
func (h *HLRC) SetConfig(ramMB int64, cpuPct float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ramMB > 0 {
		h.ramBudgetMB = ramMB
	}
	if cpuPct > 0 && cpuPct <= 100 {
		h.cpuBudgetPct = cpuPct
	}
}

// ---------------------------------------------------------------------------
// API handlers
// ---------------------------------------------------------------------------

func (b *browser) handleHLRCStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.hlrc == nil {
		writeError(w, 503, "HLRC not init")
		return
	}
	s := b.opt.hlrc.Gather()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}

func (b *browser) handleHLRCObjects(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.hlrc == nil {
		writeError(w, 503, "HLRC not init")
		return
	}
	objs := b.opt.hlrc.ObjectSnapshot()
	writeJSON(w, map[string]interface{}{"ok": true, "objects": objs})
}

func (b *browser) handleHLRCAccess(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.hlrc == nil {
		writeError(w, 503, "HLRC not init")
		return
	}
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	b.opt.hlrc.Access(req.Key)
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (b *browser) handleHLRCConfig(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.hlrc == nil {
		writeError(w, 503, "HLRC not init")
		return
	}
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var req struct {
		RAMBudgetMB int64   `json:"ramBudgetMB"`
		CPUBudgetPct float64 `json:"cpuBudgetPct"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	b.opt.hlrc.SetConfig(req.RAMBudgetMB, req.CPUBudgetPct)
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (b *browser) handleHLRCRegister(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.hlrc == nil {
		writeError(w, 503, "HLRC not init")
		return
	}
	if r.Method != "POST" {
		writeError(w, 405, "POST required")
		return
	}
	var req struct {
		Key         string `json:"key"`
		Kind        string `json:"kind"`
		RestoreCost uint32 `json:"restoreCost"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "bad request")
		return
	}
	b.opt.hlrc.Register(req.Key, req.Kind, req.RestoreCost)
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (h *HLRC) String() string {
	s := h.Gather()
	return fmt.Sprintf("HLRC: %d objects, %d demotions, RAM %d/%dMB tick=%d",
		s.Objects, s.Demotions, s.RAMUsedMB, s.RAMBudgetMB, s.Tick)
}
