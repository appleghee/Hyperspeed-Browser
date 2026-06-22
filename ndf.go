package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

// NetworkDeltaFetch (NDF) - cache only changed bytes
type NDFCache struct {
	mu       sync.RWMutex
	entries  map[string]*NDFEntry
	maxSize  int64
	currSize int64
	hlrc     *HLRC
}

func (n *NDFCache) SetHLRC(h *HLRC) { n.hlrc = h }

type NDFEntry struct {
	URL          string
	ETag         string
	LastModified string
	Hash         string
	Data         []byte
	Size         int64
	Accesses     uint64
	LastAccess   time.Time
	Created      time.Time
	HitCount     int64
}

func NewNDFCache(maxMB int64) *NDFCache {
	return &NDFCache{
		entries: make(map[string]*NDFEntry),
		maxSize: maxMB * 1024 * 1024,
	}
}

func (n *NDFCache) Fetch(url string, onData func([]byte, bool) error) (*NDFEntry, error) {
	n.mu.RLock()
	cached := n.entries[url]
	n.mu.RUnlock()

	// Build request headers
	headers := make(http.Header)
	if cached != nil && cached.ETag != "" {
		headers.Set("If-None-Match", cached.ETag)
	}
	if cached != nil && cached.LastModified != "" {
		headers.Set("If-Modified-Since", cached.LastModified)
	}

	// Fetch from remote
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	for k, v := range headers {
		req.Header[k] = v
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if cached != nil {
			n.mu.Lock()
			e, ok := n.entries[url]
			n.mu.Unlock()
			if ok {
				e.Accesses++
				e.LastAccess = time.Now()
				return e, nil
			}
			return cached, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	// 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		if cached != nil {
			n.mu.Lock()
			e, ok := n.entries[url]
			n.mu.Unlock()
			if ok {
				e.Accesses++
				e.LastAccess = time.Now()
				e.HitCount++
				if n.hlrc != nil {
					n.hlrc.Access(url)
				}
				if onData != nil {
					onData(e.Data, true)
				}
				return e, nil
			}
			if onData != nil {
				onData(cached.Data, true)
			}
			return cached, nil
		}
	}

	// Download full response
	if resp.StatusCode == http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			if cached != nil {
				n.mu.Lock()
				e, ok := n.entries[url]
				n.mu.Unlock()
				if ok {
					e.Accesses++
					e.LastAccess = time.Now()
					e.HitCount++
					return e, nil
				}
				return cached, nil
			}
			return nil, err
		}
		hash := fmt.Sprintf("%x", md5.Sum(data))

		n.mu.Lock()
		existing, exists := n.entries[url]
		if exists && existing.Hash == hash {
			existing.Accesses++
			existing.LastAccess = time.Now()
			existing.HitCount++
			n.mu.Unlock()
			if onData != nil {
				onData(existing.Data, true)
			}
			return existing, nil
		}
		entry := &NDFEntry{
			URL:          url,
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
			Hash:         hash,
			Data:         data,
			Size:         int64(len(data)),
			Accesses:     1,
			LastAccess:   time.Now(),
			Created:      time.Now(),
			HitCount:     0,
		}
		if n.hlrc != nil {
			n.hlrc.Register(url, HLRCKindCache, uint32(len(data)/1024))
			n.hlrc.UpdateCosts(url, 0.7, 0.5, int64(len(data)), 10, 0, int64(len(data)))
			n.hlrc.Access(url)
		}
		if exists {
			n.currSize -= existing.Size
		}
		n.entries[url] = entry
		n.currSize += entry.Size
		n.evictIfNeeded()
		n.mu.Unlock()

		if onData != nil {
			onData(entry.Data, false)
		}
		return entry, nil
	}

	return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
}

func (n *NDFCache) evictIfNeeded() {
	if n.currSize <= n.maxSize {
		return
	}

	// Sort by access time, evict coldest
	type kv struct {
		k string
		v *NDFEntry
	}
	var entries []kv
	for k, v := range n.entries {
		entries = append(entries, kv{k, v})
	}

	// Evict 20% of entries
	toEvict := len(entries) / 5
	if toEvict < 1 {
		toEvict = 1
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].v.LastAccess.Before(entries[j].v.LastAccess)
	})
	for i := 0; i < toEvict && i < len(entries); i++ {
		n.currSize -= entries[i].v.Size
		delete(n.entries, entries[i].k)
	}
}

func (n *NDFCache) Stats() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	totalHits := int64(0)
	totalAccesses := uint64(0)
	for _, e := range n.entries {
		totalHits += e.HitCount
		totalAccesses += e.Accesses
	}

	return map[string]interface{}{
		"cached":        len(n.entries),
		"size_mb":       n.currSize / (1024 * 1024),
		"max_mb":        n.maxSize / (1024 * 1024),
		"total_accesses": totalAccesses,
		"cache_hits":    totalHits,
		"hit_rate":      fmt.Sprintf("%.1f%%", float64(totalHits)*100/float64(totalAccesses+1)),
	}
}

func (n *NDFCache) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.entries = make(map[string]*NDFEntry)
	n.currSize = 0
}
