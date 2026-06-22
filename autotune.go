package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// AutoTuneUHE - automatically adjust UHE thresholds based on site patterns
type AutoTuneUHE struct {
	mu              sync.RWMutex
	enabled         bool
	profilesByDomain map[string]*SiteProfile
	globalProfile   *SiteProfile
	stopCh          chan struct{}
}

type SiteProfile struct {
	Domain        string
	CPUUsage      float64
	MemUsage      float64
	NetworkUsage  float64
	AccessRate    float64
	DecayRate     float64
	HotThresh     float64
	WarmThresh    float64
	CoolThresh    float64
	Samples       int
	LastUpdate    time.Time
	Recommendations string
}

func NewAutoTuneUHE() *AutoTuneUHE {
	return &AutoTuneUHE{
		enabled:         true,
		profilesByDomain: make(map[string]*SiteProfile),
		globalProfile:   &SiteProfile{Domain: "global"},
	}
}

func (a *AutoTuneUHE) Start() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.enabled || a.stopCh != nil {
		return
	}
	a.stopCh = make(chan struct{})
	go a.tuneLoop()
}

func (a *AutoTuneUHE) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopCh != nil {
		close(a.stopCh)
		a.stopCh = nil
	}
}

func (a *AutoTuneUHE) tuneLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.analyze()
		case <-a.stopCh:
			return
		}
	}
}

func (a *AutoTuneUHE) RecordMetrics(domain string, cpu, mem, net float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.profilesByDomain[domain]; !ok {
		a.profilesByDomain[domain] = &SiteProfile{
			Domain:     domain,
			HotThresh:  0.6,
			WarmThresh: 0.25,
			CoolThresh: 0.1,
			DecayRate:  0.02,
		}
	}

	p := a.profilesByDomain[domain]
	p.CPUUsage = (p.CPUUsage*float64(p.Samples) + cpu) / float64(p.Samples+1)
	p.MemUsage = (p.MemUsage*float64(p.Samples) + mem) / float64(p.Samples+1)
	p.NetworkUsage = (p.NetworkUsage*float64(p.Samples) + net) / float64(p.Samples+1)
	p.Samples++
	p.LastUpdate = time.Now()
}

func (a *AutoTuneUHE) analyze() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Evict profiles not seen in 1 hour
	cutoff := time.Now().Add(-1 * time.Hour)
	for domain, p := range a.profilesByDomain {
		if p.LastUpdate.Before(cutoff) {
			delete(a.profilesByDomain, domain)
		}
	}

	// Cap at 200 domains
	const maxProfiles = 200
	if len(a.profilesByDomain) > maxProfiles {
		var oldest string
		var oldestTime time.Time
		first := true
		for d, p := range a.profilesByDomain {
			if first || p.LastUpdate.Before(oldestTime) {
				oldest = d
				oldestTime = p.LastUpdate
				first = false
			}
		}
		delete(a.profilesByDomain, oldest)
	}

	// Analyze each domain profile
	for domain, p := range a.profilesByDomain {
		if p.Samples < 5 {
			continue
		}

		old := p.DecayRate
		a.tuneProfile(p)
		if old != p.DecayRate {
			fmt.Printf("[AutoTune] %s: decay %.3f → %.3f (cpu:%.1f%% mem:%.1f%% net:%.1f%%)\n",
				domain, old, p.DecayRate, p.CPUUsage, p.MemUsage, p.NetworkUsage)
		}
	}

	// Update global profile
	if len(a.profilesByDomain) > 0 {
		a.tuneProfile(a.globalProfile)
	}
}

func (a *AutoTuneUHE) tuneProfile(p *SiteProfile) {
	// Normalize metrics to 0-1
	cpu := math.Min(p.CPUUsage/100, 1.0)
	mem := math.Min(p.MemUsage/100, 1.0)
	net := math.Min(p.NetworkUsage/100, 1.0)

	// Adjust decay rate based on resource usage
	// High CPU/Mem → reduce decay (keep hot items longer)
	// Low CPU/Mem → increase decay (clean up faster)
	avgUsage := (cpu + mem) / 2
	baseDecay := 0.02
	if avgUsage > 0.7 {
		// Heavy site: aggressive heat retention
		p.DecayRate = baseDecay * 0.5
		p.HotThresh = 0.5
		p.WarmThresh = 0.2
		p.CoolThresh = 0.05
		p.Recommendations = "Heavy site: aggressive retention"
	} else if avgUsage > 0.4 {
		// Medium site: balanced
		p.DecayRate = baseDecay
		p.HotThresh = 0.6
		p.WarmThresh = 0.25
		p.CoolThresh = 0.1
		p.Recommendations = "Medium site: balanced"
	} else {
		// Light site: aggressive cleanup
		p.DecayRate = baseDecay * 2
		p.HotThresh = 0.7
		p.WarmThresh = 0.35
		p.CoolThresh = 0.15
		p.Recommendations = "Light site: aggressive cleanup"
	}

	// Adjust for network usage
	if net > 0.3 {
		p.DecayRate *= 0.75 // Slow down decay on heavy network
	}

	// Adjust for memory pressure
	if mem > 0.8 {
		p.DecayRate *= 1.5 // Speed up decay under mem pressure
	}
}

func (a *AutoTuneUHE) GetProfile(domain string) *SiteProfile {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if p, ok := a.profilesByDomain[domain]; ok {
		return p
	}
	return nil
}

func (a *AutoTuneUHE) GetGlobalProfile() *SiteProfile {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.globalProfile
}

func (a *AutoTuneUHE) AllProfiles() map[string]*SiteProfile {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	m := make(map[string]*SiteProfile)
	for k, v := range a.profilesByDomain {
		m[k] = v
	}
	return m
}

func (a *AutoTuneUHE) Recommend(domain string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if p, ok := a.profilesByDomain[domain]; ok {
		return p.Recommendations
	}
	return "collecting metrics..."
}

func (a *AutoTuneUHE) String() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return fmt.Sprintf("AutoTuneUHE: %d domains profiled, global decay=%.3f", 
		len(a.profilesByDomain), a.globalProfile.DecayRate)
}
