package webview

/*
#cgo CFLAGS: -Wall -O2 -Wno-unused-function -Wno-unused-variable -Werror -std=c11
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.0

#include <stdlib.h>
#include "helpers.h"
*/
import "C"
import (
	"errors"
	"log"
	"runtime"
	"sync"
	"unsafe"

	"github.com/OneOfOne/webview/internal/cache"
)

var (
	AutoQuitGTK = true

	Debug = false

	ErrWindowIsClosed = errors.New("WebView is already closed.")
)

type WebKitSettings struct {
	EnableJava            bool
	EnablePlugins         bool
	EnableFrameFlattening bool
	EnableSmoothScrolling bool
	EnableSpellChecking   bool
	EnableFullscreen      bool
	EnableLocalFileAccess bool

	EnableJavaScript               bool
	EnableJavaScriptCanOpenWindows bool
	AllowModalDialogs              bool

	EnableWriteConsoleMessagesToStdout bool

	EnableWebGL bool

	IgnoreTLSErrors bool
}

type WebKitBoolProperty struct {
	Name  string
	Value bool
}

type Settings struct {
	Offscreen bool

	Decorated  bool
	Resizable  bool
	Fullscreen bool

	Width  int
	Height int

	UserAgent string

	WebKit WebKitSettings
}

var (
	DefaultWebKitSettings = WebKitSettings{
		EnableJava:            false,
		EnablePlugins:         false,
		EnableFrameFlattening: true,
		EnableSmoothScrolling: true,
		EnableSpellChecking:   true,
		EnableFullscreen:      true,

		EnableJavaScript:               true,
		EnableJavaScriptCanOpenWindows: true,
		AllowModalDialogs:              true,

		EnableWriteConsoleMessagesToStdout: true,

		EnableWebGL: false,

		IgnoreTLSErrors: true,
	}

	DefaultSettings = Settings{
		WebKit: DefaultWebKitSettings,

		Decorated: true,
		Resizable: true,

		Width:  1024,
		Height: 768,

		UserAgent: "OneOfOne-Go-WebView/" + runtime.Version(),
	}
)

func (s *Settings) c() *C.settings_t {
	var v C.settings_t

	ws := s.WebKit
	v.EnableJava = cbool(ws.EnableJava)
	v.EnablePlugins = cbool(ws.EnablePlugins)

	v.EnableFrameFlattening = cbool(ws.EnableFrameFlattening)
	v.EnableSmoothScrolling = cbool(ws.EnableSmoothScrolling)
	v.EnableSpellChecking = cbool(ws.EnableSpellChecking)

	v.EnableFullscreen = cbool(ws.EnableFullscreen)
	v.EnableLocalFileAccess = cbool(ws.EnableFullscreen)

	v.EnableJavaScript = cbool(ws.EnableJavaScript)
	v.EnableJavaScriptCanOpenWindows = cbool(ws.EnableJavaScriptCanOpenWindows)
	v.AllowModalDialogs = cbool(ws.AllowModalDialogs)

	v.EnableWriteConsoleMessagesToStdout = cbool(ws.EnableWriteConsoleMessagesToStdout)
	v.EnableWebGL = cbool(ws.EnableWebGL)
	v.IgnoreTLSErrors = cbool(ws.IgnoreTLSErrors)

	v.Decorated = cbool(s.Decorated)
	v.Resizable = cbool(s.Resizable)

	if s.Fullscreen {
		v.Width, v.Height = -1, -1
	} else {
		v.Width = C.int(s.Width)
		v.Height = C.int(s.Height)
	}

	return &v
}

func cbool(v bool) C.gboolean {
	if v {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}

type WebView struct {
	id uint64
	q  chan func()

	done    chan struct{}
	started chan struct{}
	msgs    chan *JSValue
	// snapshots chan *snapshot

	win *C.GtkWidget
	wv  *C.WebKitWebView

	loadMux sync.Mutex
	loadCh  chan string

	OnMessage func(j *JSValue, cb func(ret interface{}) *JSValue)
}

func New(windowTitle string, s *Settings) *WebView {
	wv := &WebView{
		q:       make(chan func(), 1),
		done:    make(chan struct{}),
		started: make(chan struct{}),
		msgs:    make(chan *JSValue, 10),
		loadCh:  make(chan string),
	}

	runtime.SetFinalizer(wv, func(wv *WebView) { wv.Close() })

	wv.id = addView(wv)

	if s == nil {
		s = &DefaultSettings
	}

	ua := C.CString(s.UserAgent)
	defer C.free(unsafe.Pointer(ua))

	title := C.CString(windowTitle)
	defer C.free(unsafe.Pointer(title))

	wv.exec(func() {
		wv.win = C.create_window(cbool(s.Offscreen))
		wv.wv = C.init_window(wv.win, title, ua, s.c(), C.guint64(wv.id))
	})

	<-wv.started
	go wv.watchMessages()
	return wv
}

func (wv *WebView) LoadHTML(html string) string {
	wv.loadMux.Lock()
	defer wv.loadMux.Unlock()

	wv.exec(func() {
		html := C.CString(html)
		defer C.free(unsafe.Pointer(html))
		C.load_html(wv.wv, html)
	})

	return <-wv.loadCh
}

func (wv *WebView) LoadURI(uri string) string {
	wv.loadMux.Lock()
	defer wv.loadMux.Unlock()
	wv.exec(func() {
		uri := C.CString(uri)
		defer C.free(unsafe.Pointer(uri))
		C.load_uri(wv.wv, uri)
	})

	return <-wv.loadCh
}

func (wv *WebView) Close() error {
	select {
	case <-wv.done:
		return ErrWindowIsClosed
	default:
	}
	C.close_window(wv.wv, wv.win)
	delView(wv.id)
	close(wv.done)
	close(wv.msgs)
	wv.loadMux.Lock()
	close(wv.loadCh)
	wv.loadMux.Unlock()
	return nil
}

func (wv *WebView) Done() <-chan struct{} { return wv.done }

func (wv *WebView) exec(fn func()) {
	ch := make(chan struct{})
	gtk_idle_add(func(_ unsafe.Pointer) {
		fn()
		close(ch)
	})
	<-ch
}

func (wv *WebView) WithGtkContext(fn func(win *C.GtkWidget, wv *C.WebKitWebView)) {
	wv.exec(func() {
		fn(wv.win, wv.wv)
	})
}

var (
	views      = cache.NewLMap()
	mainMux    sync.Mutex
	calledMain bool
)

func CloseAll() {
	views.ForEach(nil, func(key uint64, val interface{}) bool {
		if wv, _ := val.(*WebView); wv != nil {
			wv.Close()
		}
		return true
	})
	if AutoQuitGTK {
		return
	}
	mainMux.Lock()
	destoryGUI()
	calledMain = false
	mainMux.Unlock()
}

func startGUI() {
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		close(done)
		C.gtk_main()
	}()
	<-done

}

func destoryGUI() {
	C.gtk_main_quit()
}

func checkStartGUI() {
	mainMux.Lock()
	if !calledMain {
		startGUI()
		calledMain = true
	}
	mainMux.Unlock()
}

func addView(wv *WebView) uint64 {
	checkStartGUI()
	id := nextID()
	views.Set(id, wv)
	return id
}

func delView(id uint64) {
	views.Delete(id)
	if views.Len() > 0 || !AutoQuitGTK {
		return
	}
	mainMux.Lock()
	destoryGUI()
	calledMain = false
	mainMux.Unlock()
}

func getView(id uint64) *WebView {
	wv, _ := views.Get(id).(*WebView)
	return wv
}

//export inGtkMain
func inGtkMain(p C.guint64) {
	if Debug {
		log.Printf("inGtkMain (%d)", p)
	}

	if wv := getView(uint64(p)); wv != nil {
		select {
		case fn := <-wv.q:
			fn()
		default:
			return
		}
	}

}

//export closeHandler
func closeHandler(p C.guint64) {
	if Debug {
		log.Printf("closeHandler (%d)", p)
	}
	if wv := getView(uint64(p)); wv != nil {
		wv.Close()
	}
}

//export startHandler
func startHandler(p C.guint64) {
	if Debug {
		log.Printf("startHandler (%d)", p)
	}
	if wv := getView(uint64(p)); wv != nil {
		close(wv.started)
	}
}

//export wvLoadFinished
func wvLoadFinished(p C.guint64, url *C.char) {
	if Debug {
		log.Printf("wvLoadFinished (%d): %s", p, C.GoString(url))
	}
	if wv := getView(uint64(p)); wv != nil {
		wv.loadCh <- C.GoString(url)
	}
}
