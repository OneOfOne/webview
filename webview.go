package gtkwebview

/*
#cgo linux CFLAGS: -DWEBVIEW_GTK=1 -Wall -O2 -Wno-unused-function -Wno-unused-variable -Werror
#cgo linux pkg-config: gtk+-3.0 webkit2gtk-4.0

#include <stdlib.h>
#include "webview.h"
*/
import "C"
import (
	"errors"
	"log"
	"runtime"
	"sync"
	"unsafe"
)

var (
	AutoQuitGTK = true

	Debug = true

	ErrWindowIsClosed = errors.New("WebView is already closed.")
)

type WebKitSettings struct {
	EnableJava            bool
	EnablePlugins         bool
	EnableFrameFlattening bool
	EnableSmoothScrolling bool
	EnableSpellChecking   bool

	EnableJavaScript               bool
	EnableJavaScriptCanOpenWindows bool
	AllowModalDialogs              bool

	EnableWriteConsoleMessagesToStdout bool

	EnableWebGL bool
}

type WebKitBoolProperty struct {
	Name  string
	Value bool
}

type Settings struct {
	*WebKitSettings

	Decorated bool
	Resizable bool

	Width  int
	Height int

	UserAgent string
}

var (
	DefaultWebKitSettings = WebKitSettings{
		EnableJava:            false,
		EnablePlugins:         false,
		EnableFrameFlattening: true,
		EnableSmoothScrolling: true,
		EnableSpellChecking:   true,

		EnableJavaScript:               true,
		EnableJavaScriptCanOpenWindows: true,
		AllowModalDialogs:              true,

		EnableWriteConsoleMessagesToStdout: true,

		EnableWebGL: false,
	}

	DefaultSettings = Settings{
		WebKitSettings: &DefaultWebKitSettings,

		Decorated: true,
		Resizable: true,

		Width:  1024,
		Height: 768,

		UserAgent: "webkit2gtk/" + runtime.Version(),
	}
)

func (s *Settings) c() *C.settings_t {
	var v C.settings_t

	if ws := s.WebKitSettings; ws != nil {
		v.EnableJava = cbool(ws.EnableJava)
		v.EnablePlugins = cbool(ws.EnablePlugins)
		v.EnableFrameFlattening = cbool(ws.EnableFrameFlattening)
		v.EnableSmoothScrolling = cbool(ws.EnableSmoothScrolling)
		v.EnableJavaScript = cbool(ws.EnableJavaScript)
		v.EnableJavaScriptCanOpenWindows = cbool(ws.EnableJavaScriptCanOpenWindows)
		v.AllowModalDialogs = cbool(ws.AllowModalDialogs)
		v.EnableWriteConsoleMessagesToStdout = cbool(ws.EnableWriteConsoleMessagesToStdout)
		v.EnableWebGL = cbool(ws.EnableWebGL)
	}
	v.Decorated = cbool(s.Decorated)
	v.Resizable = cbool(s.Resizable)
	v.Width = C.int(s.Width)
	v.Height = C.int(s.Height)

	return &v
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

func cbool(v bool) C.gboolean {
	if v {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}

type WebView struct {
	id uint32
	q  chan func()

	done    chan struct{}
	started chan struct{}

	win *C.GtkWidget
	wv  *C.WebKitWebView

	OnPageLoad func(uri string)
}

func New(windowTitle string, s *Settings) *WebView {
	wv := &WebView{
		q:       make(chan func(), 1),
		done:    make(chan struct{}),
		started: make(chan struct{}),
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
		wv.win = C.create_window()
		wv.wv = C.init_window(wv.win, title, ua, s.c(), C.guint64(wv.id))
	})

	<-wv.started
	return wv
}

func (wv *WebView) LoadHTML(html string) {
	wv.exec(func() {
		html := C.CString(html)
		defer C.free(unsafe.Pointer(html))
		C.load_html(wv.wv, html)
	})
}

func (wv *WebView) LoadURI(uri string) {
	wv.exec(func() {
		uri := C.CString(uri)
		defer C.free(unsafe.Pointer(uri))
		C.load_uri(wv.wv, uri)
	})
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
	return nil
}

func (wv *WebView) Done() <-chan struct{} { return wv.done }

func (wv *WebView) exec(fn func()) {
	ch := make(chan struct{})
	wv.q <- func() {
		fn()
		close(ch)
	}
	C.idle_add(C.guint64(wv.id))
	<-ch
}

func (wv *WebView) WithGtkContext(fn func(win *C.GtkWidget, wv *C.WebKitWebView)) {
	wv.exec(func() {
		fn(wv.win, wv.wv)
	})
}

var views = struct {
	sync.RWMutex
	m          map[uint32]*WebView
	calledMain bool
	counter    uint32
}{
	m: map[uint32]*WebView{},
}

func CloseAll() {
	views.Lock()
	wvs := make([]*WebView, 0, len(views.m))
	for _, wv := range views.m {
		wvs = append(wvs, wv)
	}
	views.Unlock()
	for _, wv := range wvs {
		wv.Close()
	}
	if AutoQuitGTK {
		return
	}
	views.Lock()
	defer views.Unlock()
	destoryGUI()
	views.calledMain = false
}

func addView(wv *WebView) uint32 {
	views.Lock()
	defer views.Unlock()
	if !views.calledMain {
		startGUI()
		views.calledMain = true
	}
	id := views.counter
	views.counter++
	views.m[id] = wv
	return id
}

func delView(id uint32) {
	views.Lock()
	defer views.Unlock()
	delete(views.m, id)
	if len(views.m) == 0 && AutoQuitGTK {
		destoryGUI()
		views.calledMain = false
	}
}

func getView(id uint32) *WebView {
	views.RLock()
	defer views.RUnlock()
	return views.m[id]
}

//export in_gtk_main
func in_gtk_main(p C.guint64) {
	if Debug {
		log.Printf("in_gtk_main (%d)", p)
	}

	if wv := getView(uint32(p)); wv != nil {
		select {
		case fn := <-wv.q:
			fn()
		default:
			return
		}
	}

}

//export close_handler
func close_handler(p C.guint64) {
	if Debug {
		log.Printf("close_handler (%d)", p)
	}
	if wv := getView(uint32(p)); wv != nil {
		wv.Close()
	}
}

//export start_handler
func start_handler(p C.guint64) {
	if Debug {
		log.Printf("start_handler (%d)", p)
	}
	if wv := getView(uint32(p)); wv != nil {
		close(wv.started)
	}
}

//export wv_load_finished
func wv_load_finished(p C.guint64, url *C.char) {
	if Debug {
		log.Printf("wv_load_finished (%d): %s", p, C.GoString(url))
	}
	if wv := getView(uint32(p)); wv != nil && wv.OnPageLoad != nil {
		wv.OnPageLoad(C.GoString(url))
	}
}
