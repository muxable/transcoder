#ifndef ELEMENT_H
#define ELEMENT_H

#include <glib.h>

static void _g_object_set_addr(gpointer object, const gchar *property_name, void *value)
{
    g_object_set(object, property_name, *(gpointer **)value, NULL);
}

static void _g_object_get_string(gpointer object, const gchar *property_name, gchar **value)
{
    g_object_get(object, property_name, value, NULL);
}

static void _g_object_get_gint64(gpointer object, const gchar *property_name, gint64 *value)
{
    g_object_get(object, property_name, value, NULL);
}

#endif