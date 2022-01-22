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
	"fmt"
	"io"
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

func (e *Element) PushBuffer(data []byte) (err error) {
	b := C.CBytes(data)
	defer C.free(b)

	p := C.compat_memdup(C.gconstpointer(b), C.ulong(len(data)))
	cdata := C.gst_buffer_new_wrapped(p, C.ulong(len(data)))

	gstReturn := C.gst_app_src_push_buffer((*C.GstAppSrc)(unsafe.Pointer(e.GstElement)), cdata)

	if gstReturn != C.GST_FLOW_OK {
		err = errors.New("could not push buffer on appsrc element")
		return
	}

	return
}

func (e *Element) PullSample() (sample *Sample, err error) {
	CGstSample := C.gst_app_sink_pull_sample((*C.GstAppSink)(unsafe.Pointer(e.GstElement)))
	if CGstSample == nil {
		err = errors.New("could not pull a sample from appsink")
		return
	}

	gstBuffer := C.gst_sample_get_buffer(CGstSample)

	if gstBuffer == nil {
		err = errors.New("could not pull a sample from appsink")
		return
	}

	mapInfo := (*C.GstMapInfo)(unsafe.Pointer(C.malloc(C.sizeof_GstMapInfo)))
	defer C.free(unsafe.Pointer(mapInfo))

	if int(C.gst_buffer_map(gstBuffer, mapInfo, C.GST_MAP_READ)) == 0 {
		err = fmt.Errorf("could not map gstBuffer %#v", gstBuffer)
		return
	}

	CData := (*[1 << 30]byte)(unsafe.Pointer(mapInfo.data))
	data := make([]byte, int(mapInfo.size))
	copy(data, CData[:])

	duration := uint64((*C.GstBuffer)(unsafe.Pointer(gstBuffer)).duration)
	pts := uint64((*C.GstBuffer)(unsafe.Pointer(gstBuffer)).pts)
	dts := uint64((*C.GstBuffer)(unsafe.Pointer(gstBuffer)).dts)
	offset := uint64((*C.GstBuffer)(unsafe.Pointer(gstBuffer)).offset)

	sample = &Sample{
		Data:     data,
		Duration: duration,
		Pts:      pts,
		Dts:      dts,
		Offset:   offset,
	}

	C.gst_buffer_unmap(gstBuffer, mapInfo)
	C.gst_sample_unref(CGstSample)

	return
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
	return e.PushBuffer(buf)
}

func (e *Element) ReadRTP() (*rtp.Packet, error) {
	sample, err := e.PullSample()
	if err != nil {
		if e.IsEOS() {
			return nil, io.EOF
		}
		return nil, err
	}

	p := &rtp.Packet{}
	if err := p.Unmarshal(sample.Data); err != nil {
		return nil, err
	}
	return p, nil
}

func (e *Element) Close() error {
	return e.EndOfStream()
}