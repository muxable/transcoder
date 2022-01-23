#include <gst/gst.h>
#include <glib.h>

int
main (int   argc,
      char *argv[])
{
  GMainLoop *loop;

  GstElement *pipeline;

  /* Initialisation */
  gst_init (&argc, &argv);

  loop = g_main_loop_new (NULL, FALSE);

  /* Create gstreamer elements */
  pipeline = gst_parse_launch ("filesrc location=../../test/input.ivf ! decodebin ! autovideosink", NULL);


  gst_element_set_state (pipeline, GST_STATE_PLAYING);

  g_print ("Running...\n");
  g_main_loop_run (loop);

  /* Out of the main loop, clean up nicely */
  g_print ("Returned, stopping playback\n");
  gst_element_set_state (pipeline, GST_STATE_NULL);

  g_print ("Deleting pipeline\n");
  gst_object_unref (GST_OBJECT (pipeline));
  g_main_loop_unref (loop);

  return 0;
}