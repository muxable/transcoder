package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <glib.h>
#include <gst/gst.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Bin struct {
	*Element
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

	return
}

func (b *Bin) Close() error {
	C.gst_object_unref(C.gpointer(unsafe.Pointer(b.GstElement)))
	return nil
}

func (b *Bin) Add(elem ...*Element) {
	for _, e := range elem {
		C.gst_bin_add((*C.GstBin)(unsafe.Pointer(b.GstElement)), e.GstElement)
	}
}

func (b *Bin) GetByName(name string) (*Element) {
	n := (*C.gchar)(unsafe.Pointer(C.CString(name)))
	defer C.g_free(C.gpointer(unsafe.Pointer(n)))
	CElement := C.gst_bin_get_by_name((*C.GstBin)(unsafe.Pointer(b.GstElement)), n)

	if CElement == nil {
		return nil
	}
	return &Element{
		GstElement: CElement,
	}
}

func (b *Bin) Remove(child *Element) {
	C.gst_bin_remove((*C.GstBin)(unsafe.Pointer(b.GstElement)), child.GstElement)
}
