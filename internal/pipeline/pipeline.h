#ifndef PIPELINE_H
#define PIPELINE_H

#include <glib.h>
#include <gst/gst.h>

extern void goEOSFunc(GstElement *, gpointer);

void cgoEOSFunc(GstElement *, gpointer);

#endif