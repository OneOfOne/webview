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
	"sync/atomic"
	"unsafe"

	"github.com/OneOfOne/cmap"
)

var (
	Debug = true

	ErrWindowIsClosed = errors.New("WebView is already closed.")

	started  = make(chan struct{})
	done     = make(chan struct{})
	mainOnce sync.Once
	quitOnce sync.Once

	cbHandles = cmap.NewSize(8)
	counter   uint64
)

type Settings struct {
	EnableJava            bool
	EnablePlugins         bool
	EnableFrameFlattening bool
	EnableSmoothScrolling bool

	EnableJavaScript               bool
	EnableJavaScriptCanOpenWindows bool
	AllowModalDialogs              bool

	EnableWriteConsoleMessagesToStdout bool

	Resizable bool

	Width  int
	Height int

	UserAgent string
}

var DefaultSettings = Settings{
	EnableJava:            false,
	EnablePlugins:         false,
	EnableFrameFlattening: true,
	EnableSmoothScrolling: true,

	EnableJavaScript:               true,
	EnableJavaScriptCanOpenWindows: true,
	AllowModalDialogs:              true,

	EnableWriteConsoleMessagesToStdout: true,

	Resizable: true,

	Width:  800,
	Height: 600,

	UserAgent: "webkit2gtk/" + runtime.Version(),
}

func StartGUI() {
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		close(done)
		C.gtk_main()
	}()
	<-done

}

func DestoryGUI() {
	C.gtk_main_quit()
}

func (s *Settings) c() *C.settings_t {
	var v C.settings_t

	v.EnableJava = cbool(s.EnableJava)
	v.EnablePlugins = cbool(s.EnablePlugins)
	v.EnableFrameFlattening = cbool(s.EnableFrameFlattening)
	v.EnableSmoothScrolling = cbool(s.EnableSmoothScrolling)
	v.EnableJavaScript = cbool(s.EnableJavaScript)
	v.EnableJavaScriptCanOpenWindows = cbool(s.EnableJavaScriptCanOpenWindows)
	v.AllowModalDialogs = cbool(s.AllowModalDialogs)
	v.EnableWriteConsoleMessagesToStdout = cbool(s.EnableWriteConsoleMessagesToStdout)
	v.Resizable = cbool(s.Resizable)
	v.Width = C.int(s.Width)
	v.Height = C.int(s.Height)

	return &v
}

func cbool(v bool) C.gboolean {
	if v {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}

type WebView struct {
	id      C.guint64
	main    chan func()
	done    chan struct{}
	started chan struct{}

	win *C.GtkWidget
	wv  *C.WebKitWebView

	OnPageLoad func(uri string)
}

func New(windowTitle string, s *Settings) *WebView {
	wv := &WebView{
		main:    make(chan func(), 10),
		done:    make(chan struct{}),
		started: make(chan struct{}),
	}

	if s == nil {
		s = &DefaultSettings
	}

	ua := C.CString(s.UserAgent)
	defer C.free(unsafe.Pointer(ua))

	title := C.CString(windowTitle)
	defer C.free(unsafe.Pointer(title))

	wv.id = C.guint64(addToCache(wv))

	wv.exec(func() {
		wv.win = C.create_window()
		wv.wv = C.init_window(wv.win, title, ua, s.c(), wv.id)
	})
	runtime.SetFinalizer(wv, func(wv *WebView) { wv.Close() })
	<-wv.started
	return wv
}

func (wv *WebView) LoadHTML(html string) {
	wv.exec(func() {
		html := C.CString(html)
		defer C.free(unsafe.Pointer(html))
		C.loadHTML(wv.wv, html)
		log.Println("html")
	})
}

func (wv *WebView) LoadURI(uri string) {
	wv.exec(func() {
		uri := C.CString(uri)
		defer C.free(unsafe.Pointer(uri))
		C.loadURI(wv.wv, uri)
		log.Println("uri")
	})
}

func (wv *WebView) Close() error {
	select {
	case <-wv.done:
		return ErrWindowIsClosed
	default:
	}
	C.close_window(wv.wv, wv.win)
	cbHandles.Delete(uint64(wv.id))
	close(wv.done)
	return nil
}

func (wv *WebView) Done() chan struct{} { return wv.done }

func (wv *WebView) exec(fn func()) {
	ch := make(chan struct{})
	wv.main <- func() {
		fn()
		close(ch)
	}
	C.idle_add(wv.id)
	<-ch
}

// func LoadURI(uri string) {
// 	gtk_exec(func() {
// 		uri := C.CString(uri)
// 		defer C.free(unsafe.Pointer(uri))
// 		C.loadURI(uri)
// 	})
// }

// func LoadHTML(html string) {
// 	gtk_exec(func() {
// 		html := C.CString(html)
// 		defer C.free(unsafe.Pointer(html))
// 		C.loadHTML(html)
// 	})
// }

func gtk_exec(fn func()) {
	// Init("forgot to call Init", nil)

	id := atomic.AddUint64(&counter, 1)
	ch := make(chan struct{})
	cbHandles.Set(id, func() {
		fn()
		close(ch)
	})
	C.idle_add(C.guint64(id))
	<-ch
}

func addToCache(wv *WebView) uint64 {
	id := atomic.AddUint64(&counter, 1)
	cbHandles.Set(id, wv)
	return id
}

//export in_gtk_main
func in_gtk_main(p C.guint64) {
	if Debug {
		log.Printf("in_gtk_main (%d)", p)
	}
	id := uint64(p)
	if wv, ok := cbHandles.Get(id).(*WebView); ok {
		for {
			select {
			case fn := <-wv.main:
				fn()
			default:
				return
			}
		}
	}

}

//export close_handler
func close_handler(p C.guint64) {
	if Debug {
		log.Printf("close_handler (%d)", p)
	}
	if wv, ok := cbHandles.Get(uint64(p)).(*WebView); ok {
		wv.Close()
	}
}

//export start_handler
func start_handler(p C.guint64) {
	if Debug {
		log.Printf("start_handler (%d)", p)
	}
	if wv, ok := cbHandles.Get(uint64(p)).(*WebView); ok {
		close(wv.started)
	}
	// close(started)
}

//export wv_load_finished
func wv_load_finished(p C.guint64, url *C.char) {
	if Debug {
		log.Printf("wv_load_finished (%d): %s", p, C.GoString(url))
	}
	if wv, ok := cbHandles.Get(uint64(p)).(*WebView); ok {
		if wv.OnPageLoad != nil {
			wv.OnPageLoad(C.GoString(url))
		}
	}
}
