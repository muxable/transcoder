package transcoder

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include <glib.h>
#include <gst/gst.h>

#include "element.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

type Element struct {
	gst *C.GstElement
}

func (e *Element) Set(name string, value interface{}) {
	ptr := C.gpointer(unsafe.Pointer(e.gst))
	cname := (*C.gchar)(unsafe.Pointer(C.CString(name)))
	defer C.g_free(C.gpointer(unsafe.Pointer(cname)))
	switch value := value.(type) {
	case string:
		str := C.CString(value)
		defer C.free(unsafe.Pointer(str))
		C._g_object_set_addr(ptr, cname, unsafe.Pointer(&str))
	case int:
		iv := C.gint(value)
		C._g_object_set_addr(ptr, cname, unsafe.Pointer(&iv))
	case uint32:
		uiv := C.guint(value)
		C._g_object_set_addr(ptr, cname, unsafe.Pointer(&uiv))
	case bool:
		var ib C.gboolean
		if value {
			ib = C.gboolean(1)
		}
		C._g_object_set_addr(ptr, cname, unsafe.Pointer(&ib))
	case float64:
		fv := C.gdouble(value)
		C._g_object_set_addr(ptr, cname, unsafe.Pointer(&fv))
	// case *Caps:
	// 	caps := value.(*Caps)
	// 	C.X_gst_g_object_set_caps(e.gst, cname, caps.caps)
	// case *Structure:
	// 	structure := value.(*Structure)
	// 	C.X_gst_g_object_set_structure(e.gst, cname, structure.C)
	default:
		panic(fmt.Errorf("SetObject: don't know how to set value for type %T", value))
	}
}

var ErrPropertyNotFound = errors.New("property not found")

func (e *Element) GetString(name string) string {
	ptr := C.gpointer(unsafe.Pointer(e.gst))
	cname := (*C.gchar)(unsafe.Pointer(C.CString(name)))
	defer C.g_free(C.gpointer(unsafe.Pointer(cname)))

	var v *C.gchar
	C._g_object_get_string(ptr, cname, &v)
	defer C.g_free(C.gpointer(unsafe.Pointer(v)))

	return C.GoString(v)
}

func (e *Element) GetInt(name string) int {
	ptr := C.gpointer(unsafe.Pointer(e.gst))
	cname := (*C.gchar)(unsafe.Pointer(C.CString(name)))
	defer C.g_free(C.gpointer(unsafe.Pointer(cname)))

	var v C.gint64
	C._g_object_get_gint64(ptr, cname, &v)
	return int(v)
}
