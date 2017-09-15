package webview

/*
// shamelessly inspired by https://github.com/sourcegraph/go-webkit2

#include <stdlib.h>
#include <gio/gio.h>
extern void _go_callback(void* data, void *res);

static inline void _gasync_callback(GObject *source_object, GAsyncResult *res, gpointer user_data) {
	(void)source_object;
	_go_callback(user_data, res);
}

static inline void _callback(gpointer user_data) {
	_go_callback(user_data, NULL);
}

static inline void idle_add_cb(gpointer data) {
	g_idle_add((GSourceFunc)_callback, data);
}

static inline GAsyncReadyCallback _get_gasync_callback() {
	return (GAsyncReadyCallback)_gasync_callback;
}
*/
import "C"

import (
	"unsafe"
)

var gasyncCallback = C._get_gasync_callback()

type refCallback struct {
	fn func(p unsafe.Pointer)
}

//export _go_callback
func _go_callback(data unsafe.Pointer, res unsafe.Pointer) {
	defer C.free(data)
	cb := (*refCallback)(data)
	// log.Printf("%p %p", data, res)
	cb.fn(res)
}

func funcToPtr(fn func(p unsafe.Pointer)) C.gpointer {
	data := C.malloc(C.size_t(unsafe.Sizeof(refCallback{})))
	cb := (*refCallback)(data)
	cb.fn = fn
	return C.gpointer(data)
}

func newGAsyncCallback(fn func(p unsafe.Pointer)) (C.GAsyncReadyCallback, C.gpointer) {
	return gasyncCallback, funcToPtr(fn)
}

func gtk_idle_add(fn func(p unsafe.Pointer)) {
	C.idle_add_cb(funcToPtr(fn))
}
