package gtkwebview

/*
#cgo linux CFLAGS: -DWEBVIEW_GTK=1
#cgo linux pkg-config: gtk+-3.0 webkit2gtk-4.0

#include <stdlib.h>
#include "webview.h"
*/
import "C"
import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/OneOfOne/cmap"
)

func Open(title, url string, w, h int, resizable bool) error {
	New(title, url, w, h, resizable)
	<-gtkCtx.Done()
	return nil
}

var (
	started     = make(chan struct{})
	gtkCtx      context.Context
	gtkMainOnce sync.Once

	cbHandles = cmap.New()
	counter   uint64
)

func startGTK(ua string) {
	gtkMainOnce.Do(func() {
		var cancel func()
		gtkCtx, cancel = context.WithCancel(context.Background())
		go func() {
			runtime.LockOSThread()
			defer cancel()
			defer runtime.UnlockOSThread()

			ua := C.CString(ua)
			defer C.free(unsafe.Pointer(ua))
			C.create_window(ua)
		}()
	})
	<-started
}

func init() {
	log.SetFlags(log.Lshortfile)
	startGTK("user agent")
}

func main_exec(fn func()) {
	id := atomic.AddUint64(&counter, 1)
	cbHandles.Set(id, fn)
	C.idle_add(C.guint64(id))
}

func GTKCtx() context.Context { return gtkCtx }

type WebView struct {
	h unsafe.Pointer
}

type Settings struct {
}

func New(title, url string, w, h int, resizable bool) *WebView {
	var wv WebView
	main_exec(func() {
		titleStr := C.CString(title)
		defer C.free(unsafe.Pointer(titleStr))
		urlStr := C.CString(url)
		defer C.free(unsafe.Pointer(urlStr))
		resize := C.int(0)
		if resizable {
			resize = C.int(1)
		}
		C.setWebView(titleStr, urlStr, C.int(w), C.int(h), resize)
	})
	return &wv
}

type action struct {
	Exec func()
}

//export in_gtk_main
func in_gtk_main(v C.guint64) {
	if fn, ok := cbHandles.Get(uint64(v)).(func()); ok {
		fn()
	}

}

//export close_handler
func close_handler() {
	log.Printf("close_handler")
}

//export start_handler
func start_handler() {
	log.Printf("start_handler")
	close(started)
}
