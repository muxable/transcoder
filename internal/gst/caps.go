package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <gst/gst.h>
*/
import (
	"C"
)
import (
	"unsafe"
)

func IsValidCapsString(capsStr string) bool {
	pCapsStr := (*C.gchar)(unsafe.Pointer(C.CString(capsStr)))
	defer C.g_free(C.gpointer(unsafe.Pointer(pCapsStr)))

	gstCaps := C.gst_caps_from_string(pCapsStr)
	defer C.gst_caps_unref(gstCaps)

	return gstCaps != nil
}

type Caps struct {
	caps *C.GstCaps
}

func CapsFromString(caps string) *Caps {
	c := (*C.gchar)(unsafe.Pointer(C.CString(caps)))
	defer C.g_free(C.gpointer(unsafe.Pointer(c)))
	CCaps := C.gst_caps_from_string(c)
	return &Caps{caps: CCaps}
}

func (c *Caps) ToString() string {
	CStr := C.gst_caps_to_string(c.caps)
	defer C.g_free(C.gpointer(unsafe.Pointer(CStr)))
	return C.GoString((*C.char)(unsafe.Pointer(CStr)))
}

func (c *Caps) String() string {
	CStr := C.gst_caps_to_string(c.caps)
	defer C.g_free(C.gpointer(unsafe.Pointer(CStr)))
	return C.GoString((*C.char)(unsafe.Pointer(CStr)))
}
