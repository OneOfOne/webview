package webview

/*
#include <stdlib.h>
#include "helpers.h"
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
	p := C.CString(script)
	defer C.free(unsafe.Pointer(p))
	id := nextID()
	ch := make(chan *JSValue, 1)
	jsCB.Set(id, func(v *JSValue) {
		ch <- v
	})
	C.execute_javascript(wv.wv, C.guint64(id), p)
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

//export jsCallback
func jsCallback(cbID C.guint64, typ C.gint8, str *C.char, num C.double) {
	id := uint64(cbID)
	fn, _ := jsCB.DeleteAndGet(id).(func(*JSValue))

	if fn == nil {
		return
	}
	fn(newJSValue(typ, str, num))
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
