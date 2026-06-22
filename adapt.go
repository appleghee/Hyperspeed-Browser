package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// SiteType classifies what kind of page the user is on.
type SiteType int

const (
	SiteGeneral SiteType = iota
	SiteSearch
	SiteCode
	SiteVideo
	SiteSocial
	SiteNews
	SiteEcommerce
	SiteEmail
)

func (s SiteType) String() string {
	switch s {
	case SiteSearch:
		return "search"
	case SiteCode:
		return "code"
	case SiteVideo:
		return "video"
	case SiteSocial:
		return "social"
	case SiteNews:
		return "news"
	case SiteEcommerce:
		return "ecommerce"
	case SiteEmail:
		return "email"
	default:
		return "general"
	}
}

// Adapt is the intelligent site-aware engine orchestrator.
type Adapt struct {
	mu         sync.RWMutex
	currentURL string
	siteType   SiteType

	disabled        map[string]bool
	totalClassified int
}

// per-site engine blacklists — engines that contribute nothing to that site type.
var siteEngineBlacklist = map[SiteType][]string{
	SiteSearch: {
		"dna", "hbm", "avp", "pce", "upm", "ncg", "dra", "mcs", "cbl", "uee", "hfs", "rcm",
		"lod", "pvc", "rhd", "ehs", "rpc", "crg", "domCompress",
	},
	SiteCode: {
		"avp", "ehs", "rpc", "crg", "pce",
	},
	SiteVideo: {
		"domCompress", "lod", "pvc",
	},
	SiteSocial: {
		"pce", "ncg",
	},
	SiteNews: {
		"pce", "ncg",
	},
	SiteEcommerce: {
		// everything useful
	},
	SiteEmail: {
		"dna", "hbm", "avp", "pce", "upm", "ncg", "dra", "mcs", "cbl", "uee", "hfs", "rcm",
		"lod", "pvc", "rhd", "ehs", "rpc", "crg", "domCompress", "avp",
	},
}

func newAdapt() *Adapt {
	return &Adapt{
		disabled: make(map[string]bool),
	}
}

func classifySite(rawURL string) SiteType {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return SiteGeneral
	}
	host := strings.ToLower(u.Host)

	// Search engines
	if strings.Contains(host, "google.") ||
		strings.Contains(host, "duckduckgo.") ||
		strings.Contains(host, "bing.") ||
		strings.Contains(host, "yahoo.") ||
		strings.Contains(host, "baidu.") ||
		strings.Contains(host, "yandex.") ||
		strings.Contains(host, "ecosia.") ||
		strings.Contains(host, "qwant.") ||
		host == "search.brave.com" ||
		host == "www.startpage.com" {
		return SiteSearch
	}

	// Code/Dev
	if strings.Contains(host, "github.") ||
		strings.Contains(host, "gitlab.") ||
		strings.Contains(host, "bitbucket.") ||
		strings.Contains(host, "stackoverflow.") ||
		strings.Contains(host, "stackexchange.") ||
		strings.Contains(host, "npmjs.") ||
		strings.Contains(host, "pypi.") ||
		strings.Contains(host, "docs.") ||
		strings.Contains(host, "developer.") ||
		strings.Contains(host, "codepen.") ||
		strings.Contains(host, "codesandbox.") ||
		strings.Contains(host, "replit.") {
		return SiteCode
	}

	// Video
	if strings.Contains(host, "youtube.") ||
		strings.Contains(host, "youtu.be") ||
		strings.Contains(host, "twitch.") ||
		strings.Contains(host, "vimeo.") ||
		strings.Contains(host, "dailymotion.") ||
		strings.Contains(host, "netflix.") ||
		strings.Contains(host, "hulu.") ||
		strings.Contains(host, "spotify.") ||
		strings.Contains(host, "tiktok.") {
		return SiteVideo
	}

	// Social
	if strings.Contains(host, "facebook.") ||
		strings.Contains(host, "twitter.") ||
		strings.Contains(host, "x.com") ||
		strings.Contains(host, "reddit.") ||
		strings.Contains(host, "instagram.") ||
		strings.Contains(host, "linkedin.") ||
		strings.Contains(host, "discord.") ||
		strings.Contains(host, "telegram.") ||
		strings.Contains(host, "whatsapp.") ||
		strings.Contains(host, "t.me") {
		return SiteSocial
	}

	// News
	if strings.Contains(host, "news.") ||
		strings.Contains(host, "cnn.") ||
		strings.Contains(host, "bbc.") ||
		strings.Contains(host, "nytimes.") ||
		strings.Contains(host, "reuters.") ||
		strings.Contains(host, "bloomberg.") ||
		strings.Contains(host, "medium.") ||
		strings.Contains(host, "substack.") {
		return SiteNews
	}

	// E-commerce
	if strings.Contains(host, "amazon.") ||
		strings.Contains(host, "ebay.") ||
		strings.Contains(host, "etsy.") ||
		strings.Contains(host, "shopify.") ||
		strings.Contains(host, "walmart.") ||
		strings.Contains(host, "bestbuy.") ||
		strings.Contains(host, "target.") ||
		strings.Contains(host, "alibaba.") ||
		strings.Contains(host, "aliexpress.") ||
		strings.Contains(host, "shopee.") ||
		strings.Contains(host, "lazada.") ||
		strings.Contains(host, "taobao.") ||
		strings.Contains(host, "tmall.") {
		return SiteEcommerce
	}

	// Email
	if strings.Contains(host, "mail.") ||
		strings.Contains(host, "gmail.") ||
		strings.Contains(host, "outlook.") ||
		strings.Contains(host, "protonmail.") ||
		strings.Contains(host, "zoho.") ||
		strings.Contains(host, "fastmail.") ||
		host == "mail.google.com" {
		return SiteEmail
	}

	return SiteGeneral
}

// OnNavigate classifies a URL and updates the disabled engine set.
// Returns the new site type.
func (a *Adapt) OnNavigate(rawURL string) SiteType {
	rawURL = strings.TrimSpace(rawURL)
	st := classifySite(rawURL)

	a.mu.Lock()
	defer a.mu.Unlock()

	a.currentURL = rawURL
	a.siteType = st
	a.totalClassified++

	// Rebuild disabled set
	a.disabled = make(map[string]bool)
	if blacklist, ok := siteEngineBlacklist[st]; ok {
		for _, name := range blacklist {
			a.disabled[name] = true
		}
	}

	log.Printf("[ADAPT] site=%s url=%s disabled=%d/%d engines",
		st, truncate(rawURL, 80), len(a.disabled), len(engineNames))

	return st
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ShouldRun returns true if an engine should execute for the current page.
func (a *Adapt) ShouldRun(engine string) bool {
	a.mu.RLock()
	disabled := a.disabled[engine]
	a.mu.RUnlock()
	return !disabled
}

// Profile returns the current adapt state.
func (a *Adapt) Profile() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	disabledList := make([]string, 0, len(a.disabled))
	for name := range a.disabled {
		disabledList = append(disabledList, name)
	}

	return map[string]interface{}{
		"siteType":   a.siteType.String(),
		"currentURL": a.currentURL,
		"disabled":   disabledList,
		"count":      len(disabledList),
		"classified": a.totalClassified,
	}
}

// engineNames is the canonical list of all known engines.
var engineNames = []string{
	"hlrc", "uhe", "autotune", "gcCtl",
	"dna", "hbm", "avp", "domCompress", "ncg", "pce", "upm",
	"dra", "mcs", "cbl", "uee", "hfs", "rcm",
	"lod", "pvc", "rhd", "ehs", "rpc", "crg",
	"qse", "vd", "quick", "netq", "csso", "media", "cache", "tuner",
}

// ---------------------------------------------------------------------------
// HTTP handler
// ---------------------------------------------------------------------------

func (b *browser) handleAdaptStats(w http.ResponseWriter, r *http.Request) {
	if b.opt == nil || b.opt.adapt == nil {
		writeError(w, 503, "adapt not initialized")
		return
	}
	writeJSON(w, map[string]interface{}{
		"ok":      true,
		"adapt":   b.opt.adapt.Profile(),
	})
}


