#include <gst/gst.h>
#include <glib.h>

int
main (int   argc,
      char *argv[])
{
  GMainLoop *loop;

  GstElement *pipeline1, *pipeline2;

  /* Initialisation */
  gst_init (&argc, &argv);

  loop = g_main_loop_new (NULL, FALSE);

  /* Create gstreamer elements */
  pipeline1 = gst_parse_launch ("videotestsrc ! queue ! vp8enc deadline=1 ! queue ! rtpvp8pay ! udpsink host=127.0.0.1", NULL);
  gst_element_set_state (pipeline1, GST_STATE_PLAYING);
  
  pipeline2 = gst_parse_launch ("udpsrc address=0.0.0.0 caps=application/x-rtp,encoding-name=VP8 ! rtpvp8depay ! queue ! decodebin ! queue ! autovideosink", NULL);
  gst_element_set_state (pipeline2, GST_STATE_PLAYING);

  g_print ("Running...\n");
  g_main_loop_run (loop);

  /* Out of the main loop, clean up nicely */
  g_print ("Returned, stopping playback\n");
  gst_element_set_state (pipeline1, GST_STATE_NULL);
  gst_element_set_state (pipeline2, GST_STATE_NULL);

  g_print ("Deleting pipeline\n");
  gst_object_unref (GST_OBJECT (pipeline1));
  gst_object_unref (GST_OBJECT (pipeline2));
  g_main_loop_unref (loop);

  return 0;
}