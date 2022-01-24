#include <gst/gst.h>
#include <glib.h>

static gboolean bus_call(GstBus *bus, GstMessage *msg, gpointer data)
{
  GMainLoop *loop = (GMainLoop *)data;

  switch (GST_MESSAGE_TYPE(msg))
  {

  case GST_MESSAGE_EOS:
    g_print("End of stream\n");
    g_main_loop_quit(loop);
    break;

  case GST_MESSAGE_ERROR:
  {
    gchar *debug;
    GError *error;

    gst_message_parse_error(msg, &error, &debug);
    g_free(debug);

    g_printerr("Error: %s\n", error->message);
    g_error_free(error);

    g_main_loop_quit(loop);
    break;
  }
  default:
    break;
  }

  return TRUE;
}

int main(int argc, char *argv[])
{
  GstElement *pipeline, *source, *sink;
  GstBus *bus;
  GstMessage *msg;
  GstStateChangeReturn ret;

  GMainLoop *loop;

  /* Initialize GStreamer */
  gst_init(&argc, &argv);

  /* Create the elements */
  source = gst_element_factory_make("videotestsrc", "source");
  sink = gst_element_factory_make("autovideosink", "sink");

  /* Create the empty pipeline */
  pipeline = gst_pipeline_new("test-pipeline");

  if (!pipeline || !source || !sink)
  {
    g_printerr("Not all elements could be created.\n");
    return -1;
  }

  /* Build the pipeline */
  gst_bin_add_many(GST_BIN(pipeline), source, sink, NULL);
  if (gst_element_link(source, sink) != TRUE)
  {
    g_printerr("Elements could not be linked.\n");
    gst_object_unref(pipeline);
    return -1;
  }

  /* Modify the source's properties */
  g_object_set(source, "pattern", 0, NULL);

  /* Start playing */
  ret = gst_element_set_state(pipeline, GST_STATE_PLAYING);
  if (ret == GST_STATE_CHANGE_FAILURE)
  {
    g_printerr("Unable to set the pipeline to the playing state.\n");
    gst_object_unref(pipeline);
    return -1;
  }

  loop = g_main_loop_new(NULL, FALSE);

  /* Wait until error or EOS */
  bus = gst_element_get_bus(pipeline);

  gst_bus_add_watch(bus, bus_call, loop);
  gst_object_unref(bus); //Liberamos bus

  g_print("Running...\n");
  g_main_loop_run(loop); //Iteramos

  g_print("Returned, stopping playback\n");
  gst_element_set_state(pipeline, GST_STATE_NULL);

  g_print("Deleting pipeline\n");
  gst_object_unref(pipeline);
  g_main_loop_unref(loop);

  return 0;
}
