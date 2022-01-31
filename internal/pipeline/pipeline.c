#include "pipeline.h"

#include <glib.h>
#include <gst/gst.h>

void cgoEOSFunc(GstElement *object, gpointer user_data)
{
    return goEOSFunc(object, user_data);
}

gpointer g_memdup_compat(gconstpointer ptr, gsize size)
{
#if GLIB_CHECK_VERSION(2, 16, 0)
    return g_memdup2(ptr, size);
#else
    return g_memdup(ptr, size);
#endif
}