package webview

/*
#include <stdlib.h>
#include "helpers.h"

static inline gboolean jsbool(JSGlobalContextRef ctx, JSValueRef val) {
	return JSValueToBoolean(ctx, val);
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// this is implemented just so PostMessages wouldn't block the UI thread.
func (wv *WebView) watchMessages() {
	type sysMsgIn struct {
		CallbackID uint64          `json:"cbID"`
		Value      json.RawMessage `json:"val"`
	}
	type sysMsgOut struct {
		CallbackID uint64      `json:"cbID"`
		Value      interface{} `json:"val"`
	}
	for m := range wv.msgs {
		if wv.OnMessage == nil {
			continue
		}
		var (
			in sysMsgIn
			cb func(interface{}) *JSValue
		)

		if err := m.AsObject(&in); err != nil {
			m.typ, m.val = JSError, err.Error()
			goto SEND
		}

		if in.CallbackID == 0 {
			goto SEND
		}
		cb = func(v interface{}) *JSValue {
			var out sysMsgOut
			out.CallbackID, out.Value = in.CallbackID, v
			b, err := json.Marshal(out)

			if err != nil {
				b = []byte(fmt.Sprintf("{cbID:%d,err:%q}", in.CallbackID, err.Error()))
			}

			return wv.RunJS("window._replyToSystemMessage(" + string(b) + ");")
		}
		m.val = in.Value

	SEND:
		wv.OnMessage(m, cb)
	}
}

func (wv *WebView) RunJSAsync(script string) <-chan *JSValue {
	js := C.CString(script)
	defer C.free(unsafe.Pointer(js))

	ch := make(chan *JSValue, 1)
	cb := func(p unsafe.Pointer) {
		var gerr *C.GError
		jsres := C.webkit_web_view_run_javascript_finish(wv.wv, (*C.GAsyncResult)(p), &gerr)
		if gerr != nil {
			ch <- &JSValue{
				typ: JSError,
				val: C.GoString((*C.char)(gerr.message)),
			}
			C.g_error_free(gerr)
			return
		}
		defer C.webkit_javascript_result_unref(jsres)

		ctx := C.webkit_javascript_result_get_global_context(jsres)
		val := C.webkit_javascript_result_get_value(jsres)

		switch t := JSType(C.JSValueGetType(ctx, val)); t {
		case JSString:
			sv := C.js_get_str(C.JSValueToStringCopy(ctx, val, nil))
			ch <- &JSValue{
				typ: t,
				val: C.GoString(sv),
			}
			C.g_free(C.gpointer(sv))

		case JSObject:
			sv := C.js_get_str(C.JSValueCreateJSONString(ctx, val, 0, nil))
			ch <- &JSValue{
				typ: t,
				val: C.GoString(sv),
			}
			C.g_free(C.gpointer(sv))

		case JSBoolean:
			ch <- &JSValue{
				typ: t,
				val: C.jsbool(ctx, val) == 1,
			}

		case JSNumber:
			ch <- &JSValue{
				typ: t,
				val: float64(C.JSValueToNumber(ctx, val, nil)),
			}
		default: // void
			close(ch)
		}
	}
	ccb, data := newGAsyncCallback(cb)
	wv.exec(func() {
		C.webkit_web_view_run_javascript(wv.wv, (*C.gchar)(js), nil, ccb, data)
	})
	return ch
}

func (wv *WebView) RunJS(script string) *JSValue {
	return <-wv.RunJSAsync(script)
}

type JSType int8

const (
	JSError     = JSType(-1)
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
	case JSError:
		return "<error>"
	default:
		return "<invalid>"
	}
}

//export jsSystemMessage
func jsSystemMessage(viewID C.guint64, typ C.gint8, str *C.char, num C.double) {
	if wv := getView(uint64(viewID)); wv != nil && wv.OnMessage != nil {
		// TODO: optimize this, too many copies.
		// using a channel here because this is executed inside the UI thread.
		wv.msgs <- newJSValue(typ, str, num)
	}
}

// TODO: https://webkitgtk.org/reference/webkit2gtk/stable/WebKitUserContentManager.html
