package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// NetworkDeltaFetch (NDF) - cache only changed bytes
type NDFCache struct {
	mu       sync.RWMutex
	entries  map[string]*NDFEntry
	maxSize  int64
	currSize int64
}

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
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if cached != nil {
			cached.Accesses++
			cached.LastAccess = time.Now()
			return cached, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	// 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		if cached != nil {
			cached.Accesses++
			cached.LastAccess = time.Now()
			cached.HitCount++
			if onData != nil {
				onData(cached.Data, true)
			}
			return cached, nil
		}
	}

	// Download full response
	if resp.StatusCode == http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		hash := fmt.Sprintf("%x", md5.Sum(data))

		// Only cache if different from cached
		if cached != nil && cached.Hash == hash {
			cached.Accesses++
			cached.LastAccess = time.Now()
			cached.HitCount++
			return cached, nil
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

		n.mu.Lock()
		if cached != nil {
			n.currSize -= cached.Size
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

	// Sort by last access time
	var coldest []kv
	for _, e := range entries {
		coldest = append(coldest, e)
	}
	for i := 0; i < len(coldest); i++ {
		for j := i + 1; j < len(coldest); j++ {
			if coldest[j].v.LastAccess.Before(coldest[i].v.LastAccess) {
				coldest[i], coldest[j] = coldest[j], coldest[i]
			}
		}
	}

	for i := 0; i < toEvict && i < len(coldest); i++ {
		n.currSize -= coldest[i].v.Size
		delete(n.entries, coldest[i].k)
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
