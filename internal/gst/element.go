package gst

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include <glib.h>
#include <gst/gst.h>
#include <gst/app/gstappsrc.h>
#include <gst/app/gstappsink.h>

gpointer compat_memdup(gconstpointer mem, gsize byte_size) {
#if GLIB_CHECK_VERSION(2, 68, 0)
    return g_memdup2(mem, byte_size);
#else
    return g_memdup(mem, byte_size);
#endif
}
*/
import "C"
import (
	"errors"
	"log"
	"runtime"
	"unsafe"

	"github.com/pion/rtp"
)

type Element struct {
	GstElement *C.GstElement
}

func (e *Element) SetState(state StateOptions) StateChangeReturn {
	Cint := C.gst_element_set_state(e.GstElement, C.GstState(state))
	return StateChangeReturn(Cint)
}

func (e *Element) EndOfStream() (err error) {
	// EndOfStream signals that the appsrc will not receive any further
	// input via PushBuffer and permits the pipeline to finish properly.

	gstReturn := C.gst_app_src_end_of_stream((*C.GstAppSrc)(unsafe.Pointer(e.GstElement)))
	if gstReturn != C.GST_FLOW_OK {
		err = errors.New("could not send end_of_stream")
	}
	return
}

func (e *Element) PushSample(s *Sample) error {
	gstReturn := C.gst_app_src_push_sample((*C.GstAppSrc)(unsafe.Pointer(e.GstElement)), s.GstSample)

	log.Printf("%v", gstReturn)
	if gstReturn != C.GST_FLOW_OK {
		return errors.New("could not push buffer on appsrc element")
	}
	return nil
}

func (e *Element) PullSample() (*Sample, error) {
	CGstSample := C.gst_app_sink_pull_sample((*C.GstAppSink)(unsafe.Pointer(e.GstElement)))
	if CGstSample == nil {
		return nil, errors.New("could not pull a sample from appsink")
	}

	s := &Sample{}
	s.GstSample = CGstSample

	runtime.SetFinalizer(s, func(s *Sample) {
		C.gst_object_unref(C.gpointer(unsafe.Pointer(s.GstSample)))
	})

	return s, nil
}

func (e *Element) IsEOS() bool {
	Cbool := C.gst_app_sink_is_eos((*C.GstAppSink)(unsafe.Pointer(e.GstElement)))
	return Cbool == 1
}

func (e *Element) WriteRTP(p *rtp.Packet) error {
	// buf, err := p.Marshal()
	// if err != nil {
	// 	return err
	// }
	// return e.PushBuffer(buf)
	return nil
}

func (e *Element) ReadRTP() (*rtp.Packet, error) {
	return nil, nil
	// sample, err := e.PullSample()
	// if err != nil {
	// 	if e.IsEOS() {
	// 		return nil, io.EOF
	// 	}
	// 	return nil, err
	// }

	// p := &rtp.Packet{}
	// if err := p.Unmarshal(sample.Data); err != nil {
	// 	return nil, err
	// }
	// return p, nil
}

func (e *Element) Close() error {
	return e.EndOfStream()
}
