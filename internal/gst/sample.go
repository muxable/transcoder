package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <gst/gst.h>
*/
import "C"
type Sample struct {
	GstSample *C.GstSample
}
