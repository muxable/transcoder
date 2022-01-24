package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <glib.h>
#include <gst/gst.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

type Pipeline struct {
	*Bin
}

func ParseLaunch(s string) (e *Pipeline, err error) {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	gstElt := C.gst_parse_launch(cs, nil)
	if gstElt == nil {
		return nil, errors.New("could not create a Gstreamer pipeline")
	}
	return &Pipeline{Bin: &Bin{Element: &Element{GstElement: gstElt}}}, nil
}

func NewPipeline() (e *Pipeline, err error) {
	gstElt := C.gst_pipeline_new(nil)
	if gstElt == nil {
		return nil, errors.New("could not create a Gstreamer pipeline")
	}
	return &Pipeline{Bin: &Bin{Element: &Element{GstElement: gstElt}}}, nil
}

// Close closes the child element.
func (p *Pipeline) Close() error {
	C.gst_object_unref(C.gpointer(unsafe.Pointer(p.GstElement)))
	return nil
}