package webview

/*
// shamelessly inspired by https://github.com/sourcegraph/go-webkit2

#include <stdlib.h>
#include <gio/gio.h>
extern void _go_callback(void* data, void *res);

static inline void _gasync_callback(GObject *source_object, GAsyncResult *res, void* user_data) {
	(void)source_object;
	_go_callback(user_data, res);
}

static inline void _callback(void* user_data) {
	_go_callback(user_data, NULL);
}

static inline void idle_add_cb(void* data) {
	g_idle_add((GSourceFunc)_callback, data);
}
*/
import "C"

import (
	"log"
	"unsafe"
)

type refCallback struct {
	fn func(p unsafe.Pointer)
}

//export _go_callback
func _go_callback(data unsafe.Pointer, res unsafe.Pointer) {
	defer C.free(data)
	cb := (*refCallback)(data)
	log.Printf("%p %p", data, res)
	cb.fn(res)
}

func funcToCallback(fn func(p unsafe.Pointer)) unsafe.Pointer {
	data := C.malloc(C.size_t(unsafe.Sizeof(refCallback{})))
	cb := (*refCallback)(data)
	cb.fn = fn
	return data
}

func newGAsyncCallback(fn func(p unsafe.Pointer)) (C.GAsyncReadyCallback, C.gpointer) {
	return C.GAsyncReadyCallback(C._gasync_callback), C.gpointer(funcToCallback(fn))
}

func gtk_idle_add(fn func(p unsafe.Pointer)) {
	C.idle_add_cb(funcToCallback(fn))
}
