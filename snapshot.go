package webview

/*
#include <stdlib.h>
#include "helpers.h"
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"log"
	"unsafe"
)

func (wv *WebView) Snapshot() (img *image.RGBA, err error) {
	// http://cairographics.org/manual/cairo-cairo-surface-t.html#cairo-surface-type-t
	const cairoSurfaceTypeImage = 0
	// http://cairographics.org/manual/cairo-Image-Surfaces.html#cairo-format-t
	const cairoImageSurfaceFormatARGB32 = 0

	ch := make(chan struct{})
	cb := func(p unsafe.Pointer) {
		defer close(ch)

		var gerr *C.GError

		surface := C.webkit_web_view_get_snapshot_finish(wv.wv, (*C.GAsyncResult)(p), &gerr)
		if gerr != nil {
			err = fmt.Errorf("%s", gerr.message)
			C.g_error_free(gerr)
			return
		}
		defer C.cairo_surface_destroy(surface)

		if C.cairo_surface_get_type(surface) != cairoSurfaceTypeImage ||
			C.cairo_image_surface_get_format(surface) != cairoImageSurfaceFormatARGB32 {
			panic("Snapshot in unexpected format")
		}

		w := int(C.cairo_image_surface_get_width(surface))
		h := int(C.cairo_image_surface_get_height(surface))
		stride := int(C.cairo_image_surface_get_stride(surface))
		data := unsafe.Pointer(C.cairo_image_surface_get_data(surface))
		surfaceBytes := C.GoBytes(data, C.int(stride*h))
		// convert from b,g,r,a or a,r,g,b(local endianness) to r,g,b,a
		testint, _ := binary.ReadUvarint(bytes.NewBuffer([]byte{0x1, 0}))
		if testint == 0x1 {
			// Little: b,g,r,a -> r,g,b,a
			for i := 0; i < w*h; i++ {
				b := surfaceBytes[4*i+0]
				r := surfaceBytes[4*i+2]
				surfaceBytes[4*i+0] = r
				surfaceBytes[4*i+2] = b
			}
		} else {
			// Big: a,r,g,b -> r,g,b,a
			for i := 0; i < w*h; i++ {
				a := surfaceBytes[4*i+0]
				r := surfaceBytes[4*i+1]
				g := surfaceBytes[4*i+2]
				b := surfaceBytes[4*i+3]
				surfaceBytes[4*i+0] = r
				surfaceBytes[4*i+1] = g
				surfaceBytes[4*i+2] = b
				surfaceBytes[4*i+3] = a
			}
		}
		img = &image.RGBA{Pix: surfaceBytes, Stride: stride, Rect: image.Rect(0, 0, w, h)}
	}

	ccb, data := newGAsyncCallback(cb)
	wv.exec(func() {
		C.webkit_web_view_get_snapshot(wv.wv,
			C.WEBKIT_SNAPSHOT_REGION_FULL_DOCUMENT,
			C.WEBKIT_SNAPSHOT_OPTIONS_NONE, // WEBKIT_SNAPSHOT_OPTIONS_TRANSPARENT_BACKGROUND ?
			nil, ccb, data)
	})
	<-ch
	return
}

//export snapshotFinished
func snapshotFinished(p C.guint64, surface *C.cairo_surface_t, err *C.char) {
	if Debug && err != nil {
		log.Printf("snapshotFinished (%d): %s", p, C.GoString(err))
	}
	if wv := getView(uint64(p)); wv != nil {

	}
}
