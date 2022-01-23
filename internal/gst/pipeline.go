package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <glib.h>
#include <gst/gst.h>
*/
import "C"
import (
	"errors"
	"runtime"
	"unsafe"
)

type Pipeline struct {
	Bin
}

func ParseLaunch(s string) (e *Pipeline, err error) {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	gstElt := C.gst_parse_launch(cs, nil)
	if gstElt == nil {
		err = errors.New("could not create a Gstreamer pipeline")
		return
	}

	e = &Pipeline{}

	e.GstElement = gstElt

	runtime.SetFinalizer(e, func(e *Pipeline) {
		C.gst_object_unref(C.gpointer(unsafe.Pointer(e.GstElement)))
	})

	return
}

func PipelineNew() (e *Pipeline, err error) {
	gstElt := C.gst_pipeline_new(nil)
	if gstElt == nil {
		err = errors.New("could not create a Gstreamer pipeline")
		return
	}

	e = &Pipeline{}

	e.GstElement = gstElt

	runtime.SetFinalizer(e, func(e *Pipeline) {
		C.gst_object_unref(C.gpointer(unsafe.Pointer(e.GstElement)))
	})

	return
}
