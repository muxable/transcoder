package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <glib.h>
#include <gst/gst.h>
*/
import "C"

func init() {
  C.gst_init(nil, nil);
}