package webview

/*
#include <stdlib.h>
#include "helpers.h"
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/OneOfOne/webview/internal/cache"
)

var jsCB = cache.NewLMap()

func newJSValue(typ C.gint8, str *C.char, num C.double) *JSValue {
	v := &JSValue{
		typ: JSType(typ),
	}

	switch v.typ {
	case JSString, JSError:
		v.val = C.GoString(str)
	case JSObject:
		v.val = json.RawMessage(C.GoString(str))
	case JSNumber:
		v.val = float64(num)
	case JSBoolean:
		v.val = num == 1
	}
	return v
}

type JSValue struct {
	val interface{}
	typ JSType
}

func (v *JSValue) Type() JSType {
	if v == nil {
		return -128
	}
	return v.typ
}

func (v *JSValue) AsObject(out interface{}) error {
	if t := v.Type(); t != JSObject {
		return fmt.Errorf("unexpected %s, got : %s", JSObject, t)
	}
	j, _ := v.val.(json.RawMessage)
	return json.Unmarshal(j, out)
}

func (v *JSValue) AsString() string {
	s, _ := v.val.(string)
	return s
}

func (v *JSValue) AsJSON() json.RawMessage {
	s, _ := v.val.(json.RawMessage)
	return s
}

func (v *JSValue) AsNumber() float64 {
	s, _ := v.val.(float64)
	return s
}

func (v *JSValue) AsBool() bool {
	s, _ := v.val.(bool)
	return s
}

func (v *JSValue) Err() error {
	if v.Type() != JSError {
		return nil
	}
	return errors.New(v.AsString())
}

func (v *JSValue) String() string {
	return fmt.Sprintf("[%s] %v", v.typ, v.val)
}

//export getSystemScript
func getSystemScript() *C.char {
	return C.CString(sysScript)
}

const sysScript = `
(function(win) {
	const callbacks = {};
	let cbID = 0;

	win._replyToSystemMessage = (m) => {
		if(!m) return;
		const cb = callbacks[m.cbID];
		if (!!cb) {
			delete callbacks[m.cbID];
			return cb(m.val, m.err);
		}
	};

	win.postSystemMessage = (msg, cb) => {
		const m = { val: msg, cbID: 0 };
		if (typeof cb === 'function') {
			m.cbID = ++cbID;
			callbacks[m.cbID] = cb;
		}
		return webkit.messageHandlers.system.postMessage(m);
	};
})(window);
`
