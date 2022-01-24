package gst

/*
#include <glib.h>
*/
import "C"

type Loop struct {
	loop *C.GMainLoop
}

func MainLoopNew() *Loop{
	return &Loop{
		loop: C.g_main_loop_new(nil, C.int(0)),
	}
}

func (l *Loop) Wait() {
	C.g_main_loop_run(l.loop)
}

func (l *Loop) Close() {
	C.g_main_loop_unref(l.loop)
}
