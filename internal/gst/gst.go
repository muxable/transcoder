package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <gst/gst.h>
#include <glib.h>

void run() {
    GstElement *pipeline;

    gst_init (NULL, NULL);

    pipeline = gst_parse_launch ("filesrc location=../../../test/input.ivf ! decodebin ! autovideosink", NULL);

    gst_element_set_state (pipeline, GST_STATE_PLAYING);

    gst_object_unref (GST_OBJECT (pipeline));
}
*/
import "C"

func init() {
  C.gst_init(nil, nil);
}

func Run() {
  go C.run()
  select{}
}