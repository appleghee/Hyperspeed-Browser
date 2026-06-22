package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// AdaptiveGCController - dynamically adjust GC pressure based on runtime heap pressure
type AdaptiveGCController struct {
	mu                 sync.RWMutex
	enabled            bool
	stopCh             chan struct{}
	
	// EWMA state
	smoothedHeap       uint64
	prevSmoothedHeap   uint64
	alpha              float64  // EWMA smoothing factor (0.2)
	
	// Current state
	currentGCPercent   int
	lastSetTime        time.Time
	
	// Thresholds
	aggressiveThresh   float64  // >15% growth → GC aggressively
	relaxThresh        float64  // <2% growth → relax GC
	minGCPercent       int      // floor: 20
	maxGCPercent       int      // ceil: 150
	
	// System memory
	memoryLimitBytes   int64
	dynamicMemLimit    bool
	
	// Stats
	samples            int
	lastGCPercent      int
	heapGrowthRate     float64
}

func NewAdaptiveGCController() *AdaptiveGCController {
	return &AdaptiveGCController{
		enabled:          true,
		alpha:            0.2,
		aggressiveThresh: 0.15,
		relaxThresh:      0.02,
		minGCPercent:     20,
		maxGCPercent:     150,
		currentGCPercent: 100,
		dynamicMemLimit:  true,
	}
}

func (agc *AdaptiveGCController) Start() {
	agc.mu.Lock()
	defer agc.mu.Unlock()
	if !agc.enabled || agc.stopCh != nil {
		return
	}
	
	// Initialize system memory limit
	agc.setDynamicMemoryLimit()
	
	agc.stopCh = make(chan struct{})
	go agc.monitorLoop()
}

func (agc *AdaptiveGCController) Stop() {
	agc.mu.Lock()
	defer agc.mu.Unlock()
	if agc.stopCh != nil {
		close(agc.stopCh)
		agc.stopCh = nil
	}
}

func (agc *AdaptiveGCController) monitorLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			agc.evaluate()
		case <-agc.stopCh:
			return
		}
	}
}

func (agc *AdaptiveGCController) evaluate() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	agc.mu.Lock()
	defer agc.mu.Unlock()
	
	// Calculate EWMA of heap
	currentHeap := m.Alloc
	if agc.smoothedHeap == 0 {
		agc.smoothedHeap = currentHeap
	} else {
		agc.smoothedHeap = uint64(float64(agc.alpha)*float64(currentHeap) + 
			float64(1-agc.alpha)*float64(agc.smoothedHeap))
	}
	
	// Calculate growth rate
	var growthRate float64
	if agc.prevSmoothedHeap > 0 {
		growthRate = float64(int64(agc.smoothedHeap)-int64(agc.prevSmoothedHeap)) / 
			float64(agc.prevSmoothedHeap)
	}
	agc.heapGrowthRate = growthRate
	agc.prevSmoothedHeap = agc.smoothedHeap
	agc.samples++
	
	// Determine new GCPercent
	var newGCPercent int
	if growthRate > agc.aggressiveThresh {
		// Heap growing >15% → aggressive GC
		newGCPercent = agc.minGCPercent
	} else if growthRate < agc.relaxThresh {
		// Heap stable (<2%) → relax GC to reduce CPU
		newGCPercent = agc.maxGCPercent
	} else {
		// Linear interpolation between relax and aggressive
		// growthRate ∈ [0.02, 0.15] → GCPercent ∈ [150, 20]
		ratio := (growthRate - agc.relaxThresh) / (agc.aggressiveThresh - agc.relaxThresh)
		newGCPercent = agc.maxGCPercent - int(ratio*float64(agc.maxGCPercent-agc.minGCPercent))
	}
	
	// Apply adjustment if changed
	if newGCPercent != agc.currentGCPercent {
		debug.SetGCPercent(newGCPercent)
		agc.currentGCPercent = newGCPercent
		agc.lastGCPercent = newGCPercent
		agc.lastSetTime = time.Now()
		
		if agc.samples%15 == 0 {
			fmt.Printf("[AdaptiveGC] heap=%dMB growth=%.2f%% GCPercent=%d\n",
				agc.smoothedHeap/(1024*1024), growthRate*100, newGCPercent)
		}
	}
	
	// Update memory limit if enabled
	if agc.dynamicMemLimit && agc.samples%6 == 0 {
		agc.setDynamicMemoryLimit()
	}
}

func (agc *AdaptiveGCController) setDynamicMemoryLimit() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Dynamic limit = 40% of system memory obtained from OS, not less than 96MB
	proposed := int64(m.Sys) / 100 * 40
	if proposed < 96*1024*1024 {
		proposed = 96 * 1024 * 1024
	}
	
	// Cap at 512MB to avoid runaway growth
	if proposed > 512*1024*1024 {
		proposed = 512 * 1024 * 1024
	}
	
	debug.SetMemoryLimit(proposed)
	agc.memoryLimitBytes = proposed
}

func (agc *AdaptiveGCController) Stats() map[string]interface{} {
	agc.mu.RLock()
	defer agc.mu.RUnlock()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]interface{}{
		"heap_mb":           m.Alloc / (1024 * 1024),
		"smoothed_heap_mb":  agc.smoothedHeap / (1024 * 1024),
		"growth_rate":       fmt.Sprintf("%.2f%%", agc.heapGrowthRate*100),
		"gc_percent":        agc.currentGCPercent,
		"memory_limit_mb":   agc.memoryLimitBytes / (1024 * 1024),
		"gc_runs":           m.NumGC,
		"pause_ms":          fmt.Sprintf("%.2f", float64(m.PauseNs[(m.NumGC+255)%256])/1e6),
		"samples":           agc.samples,
	}
}

func (agc *AdaptiveGCController) String() string {
	agc.mu.RLock()
	defer agc.mu.RUnlock()
	return fmt.Sprintf("AdaptiveGC: heap=%.0fMB growth=%.1f%% GCPercent=%d samples=%d",
		float64(agc.smoothedHeap)/(1024*1024), agc.heapGrowthRate*100, agc.currentGCPercent, agc.samples)
}
