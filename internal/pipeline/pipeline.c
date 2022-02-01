#include "pipeline.h"

#include <glib.h>
#include <gst/gst.h>

gpointer g_memdup_compat(gconstpointer ptr, gsize size)
{
#if GLIB_CHECK_VERSION(2, 68, 0)
    return g_memdup2(ptr, size);
#else
    return g_memdup(ptr, size);
#endif
}