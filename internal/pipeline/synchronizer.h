#ifndef SYNCHRONIZER_H
#define SYNCHRONIZER_H

#include <glib.h>
#include <gst/gst.h>

extern gboolean goBusFunc(GstBus *, GstMessage *);

gboolean cgoBusFunc(GstBus *, GstMessage *, gpointer);

#endif