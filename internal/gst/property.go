package gst

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include <gst/gst.h>
*/
import "C"

type Property struct {
	Name  string
	Value interface{}
}

const (
	FormatUndefined = C.GST_FORMAT_UNDEFINED
	FormatDefault   = C.GST_FORMAT_DEFAULT
	FormatBytes     = C.GST_FORMAT_BYTES
	FormatTime      = C.GST_FORMAT_TIME
	FormatBuffers   = C.GST_FORMAT_BUFFERS
	FormatPercent   = C.GST_FORMAT_PERCENT
)
