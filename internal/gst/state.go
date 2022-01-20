package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <glib.h>
#include <gst/gst.h>
*/
import "C"

type StateOptions int

const (
	StateVoidPending StateOptions = C.GST_STATE_VOID_PENDING
	StateNull        StateOptions = C.GST_STATE_NULL
	StateReady       StateOptions = C.GST_STATE_READY
	StatePaused      StateOptions = C.GST_STATE_PAUSED
	StatePlaying     StateOptions = C.GST_STATE_PLAYING
)

type StateChangeReturn int

const (
	StateChangeFailure StateChangeReturn = C.GST_STATE_CHANGE_FAILURE
	StateChangeSuccess StateChangeReturn = C.GST_STATE_CHANGE_SUCCESS
	StateChangeAsync   StateChangeReturn = C.GST_STATE_CHANGE_ASYNC
	StateChangePreroll StateChangeReturn = C.GST_STATE_CHANGE_NO_PREROLL
)
