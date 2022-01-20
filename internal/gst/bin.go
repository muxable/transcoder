package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <glib.h>
#include <gst/gst.h>
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

type Bin struct {
	Element
}

func ParseBinFromDescription(binStr string) (bin *Bin, err error) {
	var gError *C.GError

	pDesc := (*C.gchar)(unsafe.Pointer(C.CString(binStr)))
	defer C.g_free(C.gpointer(unsafe.Pointer(pDesc)))

	gstElt := C.gst_parse_bin_from_description(pDesc, C.int(0), &gError)

	if gError != nil {
		err = fmt.Errorf("create bin error for %s", binStr)
		return
	}

	bin = &Bin{}
	bin.GstElement = gstElt

	runtime.SetFinalizer(bin, func(bin *Bin) {
		C.gst_object_unref(C.gpointer(unsafe.Pointer(bin.GstElement)))
	})

	return
}

func (b *Bin) GetByName(name string) (element *Element) {
	n := (*C.gchar)(unsafe.Pointer(C.CString(name)))
	defer C.g_free(C.gpointer(unsafe.Pointer(n)))
	CElement := C.gst_bin_get_by_name((*C.GstBin)(unsafe.Pointer(b.GstElement)), n)

	if CElement == nil {
		return
	}
	element = &Element{
		GstElement: CElement,
	}
	return
}

func (b *Bin) Add(child *Element) {
	C.gst_bin_add((*C.GstBin)(unsafe.Pointer(b.GstElement)), child.GstElement)
}

func (b *Bin) Remove(child *Element) {
	C.gst_bin_remove((*C.GstBin)(unsafe.Pointer(b.GstElement)), child.GstElement)
}
