#ifndef PIPELINE_H
#define PIPELINE_H

#include <glib.h>
#include <gst/gst.h>

gpointer g_memdup_compat(gconstpointer, gsize);

extern gboolean goSourcePadEventFunc(GstPad *, GstObject *, GstEvent *);
extern gboolean goSinkPadEventFunc(GstPad *, GstObject *, GstEvent *);

gboolean cgoSourcePadEventFunc(GstPad *, GstObject *, GstEvent *);
gboolean cgoSinkPadEventFunc(GstPad *, GstObject *, GstEvent *);

#endif