#include "pipeline.h"

#include <glib.h>
#include <gst/gst.h>

void cgoEOSFunc(GstElement *object, gpointer user_data)
{
    return goEOSFunc(object, user_data);
}