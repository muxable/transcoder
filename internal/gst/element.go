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
	"io"
	"unsafe"

	"github.com/pion/rtp"
)

type Element struct {
	GstElement *C.GstElement
}

func FactoryElementMake(s string) (*Element, error) {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	CGstElement := C.gst_element_factory_make(cs, nil)
	if CGstElement == nil {
		return nil, errors.New("could not create element")
	}
	return &Element{GstElement: CGstElement}, nil
}

func Link(elements ...*Element) error {
	for i := 0; i < len(elements)-1; i++ {
		if C.gst_element_link(elements[i].GstElement, elements[i+1].GstElement) != C.int(1) {
			return errors.New("could not link elements")
		}
	}
	return nil
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

func (e *Element) WriteSample(s *Sample) error {
	gstReturn := C.gst_app_src_push_sample((*C.GstAppSrc)(unsafe.Pointer(e.GstElement)), s.GstSample)

	if gstReturn != C.GST_FLOW_OK {
		return errors.New("could not push buffer on appsrc element")
	}
	return nil
}

func (e *Element) ReadSample() (*Sample, error) {
	CGstSample := C.gst_app_sink_pull_sample((*C.GstAppSink)(unsafe.Pointer(e.GstElement)))
	if CGstSample == nil {
		return nil, errors.New("could not pull a sample from appsink")
	}

	return &Sample{GstSample: CGstSample}, nil
}

func (e *Element) Write(buf []byte) (int, error) {
	b := C.CBytes(buf)
	defer C.free(b)

	p := C.compat_memdup(C.gconstpointer(b), C.ulong(len(buf)))
	wrapped := C.gst_buffer_new_wrapped(p, C.ulong(len(buf)))

	if C.gst_app_src_push_buffer((*C.GstAppSrc)(unsafe.Pointer(e.GstElement)), wrapped) != C.GST_FLOW_OK {
		return 0, errors.New("could not push buffer on appsrc element")
	}

	return len(buf), nil
}

func (e *Element) Read(buf []byte) (int, error) {
	sample, err := e.ReadSample()
	if err != nil {
		if e.IsEOS() {
			return 0, io.EOF
		}
		return 0, err
	}
	if err := sample.MarshalTo(buf); err != nil {
		return 0, err
	}
	return len(buf), nil
}

func (e *Element) IsEOS() bool {
	Cbool := C.gst_app_sink_is_eos((*C.GstAppSink)(unsafe.Pointer(e.GstElement)))
	return Cbool == 1
}

func (e *Element) WriteRTP(p *rtp.Packet) error {
	buf, err := p.Marshal()
	if err != nil {
		return err
	}
	_, err = e.Write(buf)
	return err
}

func (e *Element) ReadRTP() (*rtp.Packet, error) {
	buf := make([]byte, 1500)
	n, err := e.Read(buf)
	if err != nil {
		return nil, err
	}
	p := &rtp.Packet{}
	if err := p.Unmarshal(buf[:n]); err != nil {
		return nil, err
	}
	return p, nil
}

func (e *Element) Close() error {
	return e.EndOfStream()
}
