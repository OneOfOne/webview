package webview

import "sync/atomic"

var idCounter uint64

func nextID() uint64 {
	return atomic.AddUint64(&idCounter, 1)
}
