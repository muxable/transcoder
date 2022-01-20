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

func IsValidCapsString(capsStr string) (bool) {
	pCapsStr := (*C.gchar)(unsafe.Pointer(C.CString(capsStr)))
	defer C.g_free(C.gpointer(unsafe.Pointer(pCapsStr)))

	gstCaps := C.gst_caps_from_string(pCapsStr)
	defer C.gst_caps_unref(gstCaps)

	return gstCaps != nil
}