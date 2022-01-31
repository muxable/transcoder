#include "synchronizer.h"

#include <glib.h>
#include <gst/gst.h>

gboolean cgoBusFunc(GstBus *bus, GstMessage *msg, gpointer user_data)
{
    return goBusFunc(bus, msg, user_data);
}