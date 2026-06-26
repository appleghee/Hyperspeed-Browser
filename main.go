package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net"
	"net/http"
	"strings"

	ultralightui "github.com/YindSoft/ultralight-ebitengine-port"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

const (
	navH = 40
	winW = 1280
	winH = 800
)

var (
	colNavBg     = color.RGBA{0xE8, 0xE8, 0xE8, 0xFF}
	colBorder    = color.RGBA{0xCC, 0xCC, 0xCC, 0xFF}
	colURLBg     = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	colURLText   = color.RGBA{0x22, 0x22, 0x22, 0xFF}
	colPlacehold = color.RGBA{0x99, 0x99, 0x99, 0xFF}
	colBtn       = color.RGBA{0x44, 0x44, 0x44, 0xFF}
	colBtnOff    = color.RGBA{0xBB, 0xBB, 0xBB, 0xFF}
	whitePix     *ebiten.Image
	navBgImg     *ebiten.Image
	urlBgImg     *ebiten.Image
)

func init() {
	whitePix = ebiten.NewImage(1, 1)
	whitePix.Fill(color.White)
	navBgImg = ebiten.NewImage(winW, navH)
	navBgImg.Fill(colNavBg)
	urlBgImg = ebiten.NewImage(1, 1)
	urlBgImg.Fill(colURLBg)
}

type App struct {
	ui         *ultralightui.UltralightUI
	urlInput   string
	urlDisplay string
	urlFocused bool
	history    []string
	histIdx    int
	navChan    chan string
	apiPort    int
	font       font.Face
}

type Game struct{ app *App }

func main() {
	app := &App{
		urlDisplay: "https://www.google.com",
		history:    []string{},
		navChan:    make(chan string, 32),
	}

	tt, _ := opentype.Parse(goregular.TTF)
	app.font, _ = opentype.NewFace(tt, &opentype.FaceOptions{Size: 13, DPI: 96})

	ui, err := ultralightui.NewFromURL(winW, winH-navH, app.urlDisplay, nil)
	if err != nil {
		log.Fatalf("ui: %v", err)
	}
	ui.SetBounds(0, navH, winW, winH-navH)
	app.ui = ui

	go app.startAPI()

	ebiten.SetWindowSize(winW, winH)
	ebiten.SetWindowTitle("Ultra-Browser v4.0.0-ultra")
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGame(&Game{app: app}); err != nil {
		log.Fatal(err)
	}
}

func (g *Game) Update() error {
	app := g.app

	select {
	case cmd := <-app.navChan:
		switch cmd {
		case "__back__":
			app.goBack()
		case "__forward__":
			app.goForward()
		case "__reload__":
			app.navigate(app.urlDisplay)
		default:
			app.navigate(cmd)
		}
	default:
	}

	if app.ui != nil {
		if app.urlFocused {
			ultralightui.ClearFocus()
		} else {
			app.ui.SetFocus()
		}
		app.ui.Update()
	}

	app.handleInput()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xF5, 0xF5, 0xF5, 0xFF})
	if app := g.app; app.ui != nil {
		if tex := app.ui.GetTexture(); tex != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(0, float64(navH))
			screen.DrawImage(tex, op)
		}
	}
	g.app.drawNavBar(screen)
}

func (g *Game) Layout(int, int) (int, int) { return winW, winH }

// --- Navigation ---

func (app *App) navigate(raw string) {
	url := normalizeURL(strings.TrimSpace(raw))
	if url == "" {
		return
	}
	if app.histIdx < len(app.history)-1 {
		app.history = app.history[:app.histIdx+1]
	}
	app.history = append(app.history, url)
	if len(app.history) > 100 {
		app.history = app.history[50:]
		app.histIdx -= 50
	}
	app.histIdx = len(app.history) - 1
	app.loadURL(url)
}

func (app *App) loadURL(url string) {
	if app.ui != nil {
		app.ui.Close()
	}
	ui, err := ultralightui.NewFromURL(winW, winH-navH, url, nil)
	if err != nil {
		log.Printf("[NAV] error: %v", err)
		return
	}
	ui.SetBounds(0, navH, winW, winH-navH)
	app.ui = ui
	app.urlDisplay = url
	app.urlInput = ""
	app.urlFocused = false
	log.Printf("[NAV] %s", url)
}

func (app *App) goBack() {
	if app.histIdx <= 0 {
		return
	}
	app.histIdx--
	app.loadURL(app.history[app.histIdx])
}

func (app *App) goForward() {
	if app.histIdx >= len(app.history)-1 {
		return
	}
	app.histIdx++
	app.loadURL(app.history[app.histIdx])
}

// --- Input ---

func (app *App) handleInput() {
	// Nav bar click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if my < navH {
			app.handleNavBarClick(mx, my)
		} else {
			app.urlFocused = false
		}
	}

	// URL bar keyboard
	if app.urlFocused {
		for _, r := range ebiten.AppendInputChars(nil) {
			app.urlInput += string(r)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			app.navigate(app.urlInput)
			return
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			app.urlFocused = false
			app.urlInput = ""
			return
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(app.urlInput) > 0 {
			app.urlInput = app.urlInput[:len(app.urlInput)-1]
		}
		return
	}

	// Global shortcuts
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		if inpututil.IsKeyJustPressed(ebiten.KeyL) {
			app.urlFocused = true
			app.urlInput = app.urlDisplay
			return
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			app.navigate(app.urlDisplay)
			return
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			app.goBack()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			app.goForward()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		app.navigate(app.urlDisplay)
	}
}

func (app *App) handleNavBarClick(mx, my int) {
	_ = my
	const (
		backX = 8
		btnW  = 24
		fwdX  = backX + btnW + 8
		relX  = fwdX + btnW + 8
		urlX  = relX + btnW + 12
		urlW  = winW - urlX - 8
	)

	if mx >= backX && mx < backX+btnW {
		app.goBack()
		return
	}
	if mx >= fwdX && mx < fwdX+btnW {
		app.goForward()
		return
	}
	if mx >= relX && mx < relX+btnW {
		app.navigate(app.urlDisplay)
		return
	}
	if mx >= urlX && mx < urlX+urlW {
		app.urlFocused = true
		app.urlInput = app.urlDisplay
	} else {
		app.urlFocused = false
	}
}

// --- Nav Bar Rendering ---

func (app *App) drawNavBar(screen *ebiten.Image) {
	screen.DrawImage(navBgImg, nil)
	fillRect(screen, 0, float64(navH-1), winW, 1, colBorder)

	canBack := app.histIdx > 0
	canFwd := app.histIdx < len(app.history)-1

	drawBtn(screen, 8, "◀", canBack, app.font)
	drawBtn(screen, 40, "▶", canFwd, app.font)
	drawBtn(screen, 72, "↻", true, app.font)

	urlX, urlW := 108, winW-116
	var gm ebiten.GeoM
	gm.Scale(float64(urlW), float64(navH-12))
	gm.Translate(float64(urlX), 6)
	screen.DrawImage(urlBgImg, &ebiten.DrawImageOptions{GeoM: gm})
	fillRect(screen, float64(urlX), 6, float64(urlW), float64(navH-12), colBorder)

	display := app.urlDisplay
	if app.urlFocused {
		display = app.urlInput
	}
	txtCol := colURLText
	if app.urlFocused && app.urlInput == "" {
		txtCol = colPlacehold
		display = "Type URL..."
	}
	text.Draw(screen, display, app.font, urlX+6, navH-14, txtCol)
}

func drawBtn(screen *ebiten.Image, x int, label string, enabled bool, f font.Face) {
	c := colBtn
	if !enabled {
		c = colBtnOff
	}
	text.Draw(screen, label, f, x, navH-14, c)
}

func fillRect(screen *ebiten.Image, x, y, w, h float64, cl color.Color) {
	r, g, b, a := cl.RGBA()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(x, y)
	op.ColorScale.Scale(float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535)
	screen.DrawImage(whitePix, op)
}

// --- URL Normalization ---

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw == "about:blank" {
		return raw
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") && !strings.HasPrefix(raw, "about:") {
		if strings.Contains(raw, ".") || strings.Contains(raw, "/") || strings.Contains(raw, ":") && !strings.Contains(raw, " ") {
			return "https://" + raw
		}
		return "https://www.google.com/search?q=" + strings.ReplaceAll(raw, " ", "+")
	}
	return raw
}

// --- HTTP API ---

func (app *App) startAPI() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("[api] error: %v", err)
		return
	}
	app.apiPort = listener.Addr().(*net.TCPAddr).Port
	log.Printf("[api] listening on 127.0.0.1:%d", app.apiPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/navigate", app.apiNavigate)
	mux.HandleFunc("/api/back", app.apiBack)
	mux.HandleFunc("/api/forward", app.apiForward)
	mux.HandleFunc("/api/reload", app.apiReload)
	mux.HandleFunc("/api/info", app.apiInfo)
	mux.HandleFunc("/api/eval", app.apiEval)
	http.Serve(listener, cors(mux))
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Token")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *App) apiNavigate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "POST required"})
		return
	}
	var b struct{ URL string }
	json.NewDecoder(r.Body).Decode(&b)
	app.navChan <- b.URL
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "url": b.URL})
}

func (app *App) apiBack(w http.ResponseWriter, r *http.Request) {
	app.navChan <- "__back__"
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (app *App) apiForward(w http.ResponseWriter, r *http.Request) {
	app.navChan <- "__forward__"
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (app *App) apiReload(w http.ResponseWriter, r *http.Request) {
	app.navChan <- "__reload__"
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (app *App) apiInfo(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":    true,
		"url":   app.urlDisplay,
		"title": app.urlDisplay,
	})
}

func (app *App) apiEval(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "POST required"})
		return
	}
	var b struct{ JS string }
	json.NewDecoder(r.Body).Decode(&b)
	if app.ui != nil {
		app.ui.Eval(b.JS)
	}
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func randToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
