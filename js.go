package webview

/*
#include <stdlib.h>
#include "helpers.h"
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/OneOfOne/webview/internal/cache"
)

type JSType uint8

const (
	JSUndefined = JSType(C.kJSTypeUndefined)
	JSNull      = JSType(C.kJSTypeNull)
	JSBoolean   = JSType(C.kJSTypeBoolean)
	JSNumber    = JSType(C.kJSTypeNumber)
	JSString    = JSType(C.kJSTypeString)
	JSObject    = JSType(C.kJSTypeObject)
)

func (t JSType) String() string {
	switch t {
	case JSUndefined:
		return "undefined"
	case JSNull:
		return "null"
	case JSBoolean:
		return "boolean"
	case JSNumber:
		return "number"
	case JSString:
		return "string"
	case JSObject:
		return "object"
	default:
		return "<invalid>"
	}
}

type JavascriptCallback func(JSValue, error)

var jsCB = cache.NewLMap()

type JSValue struct {
	ctx C.JSGlobalContextRef
	val C.JSValueRef
}

func (v JSValue) Type() JSType {
	return JSType(C.JSValueGetType(v.ctx, v.val))
}

func (v JSValue) AsString() string {
	p := C.js_get_str(v.ctx, v.val)
	s := C.GoString(p)
	C.g_free(C.gpointer(p))
	return s
}

//export jsCallback
func jsCallback(cbID C.guint64, ctx C.JSGlobalContextRef, v C.JSValueRef, errMsg *C.char) {
	id := uint64(cbID)
	fn, _ := jsCB.DeleteAndGet(id).(JavascriptCallback)

	if fn == nil {
		return
	}
	if errMsg != nil {
		fn(JSValue{}, errors.New(C.GoString(errMsg)))
	} else {
		fn(JSValue{ctx, v}, nil)
	}
}

func (wv *WebView) RunJavaScript(script string, fn func(JSValue, error)) {
	id := nextID()
	jsCB.Set(id, JavascriptCallback(fn))
	p := C.CString(script)
	defer C.free(unsafe.Pointer(p))
	C.execute_javascript(wv.wv, C.guint64(id), p)
}

// 	"unsafe"
// 	"unsafe"
// 	"errors"

// 	"github.com/sqs/gojs"
// )

// func (wv *WebView) RunJavaScript(script string, resultCallback func(result *gojs.Value, err error)) {
// 	var cCallback C.GAsyncReadyCallback
// 	var userData C.gpointer
// 	var err error
// 	if resultCallback != nil {
// 		callback := func(result *C.GAsyncResult) {
// 			C.free(unsafe.Pointer(userData))
// 			var jserr *C.GError
// 			jsResult := C.webkit_web_view_run_javascript_finish(v.webView, result, &jserr)
// 			if jsResult == nil {
// 				defer C.g_error_free(jserr)
// 				msg := C.GoString((*C.char)(jserr.message))
// 				resultCallback(nil, errors.New(msg))
// 				return
// 			}
// 			ctxRaw := gojs.RawGlobalContext(unsafe.Pointer(C.webkit_javascript_result_get_global_context(jsResult)))
// 			jsValRaw := gojs.RawValue(unsafe.Pointer(C.webkit_javascript_result_get_value(jsResult)))
// 			ctx := (*gojs.Context)(gojs.NewGlobalContextFrom(ctxRaw))
// 			jsVal := ctx.NewValueFrom(jsValRaw)
// 			resultCallback(jsVal, nil)
// 		}
// 		cCallback, userData, err = newGAsyncReadyCallback(callback)
// 		if err != nil {
// 			panic(err)
// 		}
// 	}
// 	C.webkit_web_view_run_javascript(v.webView, (*C.gchar)(C.CString(script)), nil, cCallback, userData)
// }
