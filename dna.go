// Tab/Page DNA — per-site behavioral fingerprint
package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type DNAEngine struct {
	b       *browser
	mu      sync.Mutex
	enabled bool
	stats   DNAStats
	fingerprints map[string]*PageDNA
}

type PageDNA struct {
	Domain        string   `json:"domain"`
	ColorScheme   string   `json:"colorScheme"`
	LayoutPattern string   `json:"layoutPattern"`
	Interactives  []string `json:"interactives"`
	Scripts       []string `json:"scripts"`
	Fonts         []string `json:"fonts"`
	AvgDOMDepth   float64  `json:"avgDOMDepth"`
	LastSeen      time.Time `json:"-"`
}

type DNAStats struct {
	Profiled  int    `json:"profiled"`
	CacheSize int    `json:"cacheSize"`
	Status    string `json:"status"`
}

const dnaGatherJS = `(function(){
var d={domain:location.hostname};
var bg=getComputedStyle(document.body).backgroundColor;
var isDark=function(c){if(!c)return false;var m=c.match(/rgba?\((\d+),(\d+),(\d+)/);if(!m)return false;return(parseInt(m[1])*299+parseInt(m[2])*587+parseInt(m[3])*114)/1000<128;}
d.colorScheme=isDark(bg)?'dark':'light';
var layout='unknown';var tags={};
document.querySelectorAll('body *').forEach(function(el){var t=el.tagName.toLowerCase();tags[t]=(tags[t]||0)+1;});
if(tags.nav&&tags.main)layout='has-nav-main';
else if(tags.header&&tags.footer)layout='has-header-footer';
else if(tags.section)layout='section-heavy';
d.layoutPattern=layout;
var ints=[];document.querySelectorAll('a,button,input,select,textarea,[tabindex]').forEach(function(el,i){
if(i<10&&el.textContent&&el.textContent.trim().length<50)ints.push((el.tagName||'').toLowerCase()+':'+(el.textContent||'').trim().slice(0,20));});
d.interactives=ints;
var scrs=[];document.querySelectorAll('script[src]').forEach(function(s,i){if(i<10)scrs.push(s.src.split('/').pop()||'');});
d.scripts=scrs;
var fts=[];document.fonts.forEach(function(f,i){if(i<10)fts.push(f.family);});
d.fonts=fts;
var dep=0,c=0;function w(e,d){if(d>20||!e)return;c++;dep+=d;for(var i=0;i<e.children.length;i++)w(e.children[i],d+1);}
w(document.body,0);d.avgDOMDepth=c>0?Math.round(dep/c*10)/10:0;
return d;
})()`

func NewDNAEngine(b *browser) *DNAEngine {
	return &DNAEngine{
		b:            b,
		enabled:      true,
		fingerprints: make(map[string]*PageDNA),
	}
}

func (d *DNAEngine) Start() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.enabled {
		return
	}
	// DNA runs on-demand via Fingerprint() — no persistent JS needed
}

func (d *DNAEngine) evictOldestDNA() {
	var oldest string
	var oldestTime time.Time
	for domain, p := range d.fingerprints {
		if oldest == "" || p.LastSeen.Before(oldestTime) {
			oldest = domain
			oldestTime = p.LastSeen
		}
	}
	delete(d.fingerprints, oldest)
}

func (d *DNAEngine) Fingerprint() (*PageDNA, error) {
	var dna PageDNA
	if err := d.b.syncUnwrapInto(dnaGatherJS, 10*time.Second, &dna); err != nil {
		return nil, fmt.Errorf("dna scan: %w", err)
	}
	dna.LastSeen = time.Now()
	d.mu.Lock()
	if len(d.fingerprints) >= 500 {
		d.evictOldestDNA()
	}
	d.fingerprints[dna.Domain] = &dna
	d.stats = DNAStats{
		Profiled:  len(d.fingerprints),
		CacheSize: len(d.fingerprints),
		Status:    "active",
	}
	d.mu.Unlock()
	return &dna, nil
}

func (d *DNAEngine) Get(domain string) *PageDNA {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.fingerprints[domain]
}

func (d *DNAEngine) Stats() DNAStats {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stats.Profiled = len(d.fingerprints)
	d.stats.CacheSize = len(d.fingerprints)
	return d.stats
}

func (b *browser) handleDNAFingerprint(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.dna == nil {
		writeError(w, 503, "DNA not init")
		return
	}
	dna, err := b.opt.dna.Fingerprint()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true, "dna": dna})
}

func (b *browser) handleDNAStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.dna == nil {
		writeError(w, 503, "DNA not init")
		return
	}
	s := b.opt.dna.Stats()
	writeJSON(w, map[string]interface{}{"ok": true, "stats": s})
}

func (b *browser) handleDNAClear(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.dna == nil {
		writeError(w, 503, "DNA not init")
		return
	}
	b.opt.dna.mu.Lock()
	b.opt.dna.fingerprints = make(map[string]*PageDNA)
	b.opt.dna.mu.Unlock()
	writeJSON(w, map[string]interface{}{"ok": true, "msg": "DNA cache cleared"})
}
