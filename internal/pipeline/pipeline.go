package pipeline

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include <glib.h>
#include <gst/app/gstappsink.h>
#include <gst/app/gstappsrc.h>
#include <gst/gst.h>

#include "pipeline.h"
*/
import "C"
import (
	"errors"
	"io"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/pion/rtp"
	"go.uber.org/zap"
)

type unsafePipeline struct {
	element      *C.GstElement
	synchronizer *Synchronizer

	source *C.GstAppSrc
	sink   *C.GstAppSink

	eos sync.Mutex

	closed bool
}

func newUnsafePipeline(synchronizer *Synchronizer, pipeline string) (*unsafePipeline, error) {
	cstr := C.CString(pipeline)
	defer C.free(unsafe.Pointer(cstr))

	var gerr *C.GError
	element := C.gst_parse_bin_from_description(cstr, C.int(0), (**C.GError)(&gerr))

	if gerr != nil {
		defer C.g_error_free((*C.GError)(gerr))
		errMsg := C.GoString(gerr.message)
		return nil, errors.New(errMsg)
	}

	if C.gst_bin_add(synchronizer.bin, element) == 0 {
		return nil, errors.New("failed to add bin to pipeline")
	}

	if C.gst_element_sync_state_with_parent(element) == 0 {
		return nil, errors.New("failed to sync bin with parent")
	}

	bin := (*C.GstBin)(unsafe.Pointer(element))

	csource := C.CString("internal-source")
	defer C.free(unsafe.Pointer(csource))

	source := C.gst_bin_get_by_name(bin, csource)

	csink := C.CString("internal-sink")
	defer C.free(unsafe.Pointer(csink))

	sink := C.gst_bin_get_by_name(bin, csink)

	p := &unsafePipeline{
		element:      element,
		source:       (*C.GstAppSrc)(unsafe.Pointer(source)),
		sink:         (*C.GstAppSink)(unsafe.Pointer(sink)),
		synchronizer: synchronizer,
	}

	p.eos.Lock()

	if sink != nil {
		// add an eos handler as well.
		ceos := C.CString("eos")
		defer C.free(unsafe.Pointer(ceos))

		C.g_signal_connect_data(C.gpointer(unsafe.Pointer(sink)), ceos, C.GCallback(C.cgoEOSFunc), C.gpointer(pointer.Save(p)), nil, 0)
	}

	runtime.SetFinalizer(p, func(pipeline *unsafePipeline) {
		if err := pipeline.Close(); err != nil {
			zap.L().Error("failed to close pipeline", zap.Error(err))
		}
		if pipeline.source != nil {
			C.gst_object_unref(C.gpointer(unsafe.Pointer(pipeline.source)))
		}
		if pipeline.sink != nil {
			C.gst_object_unref(C.gpointer(unsafe.Pointer(pipeline.sink)))
		}
	})

	return p, nil
}

//export goEOSFunc
func goEOSFunc(object *C.GstElement, data unsafe.Pointer) {
	pointer.Restore(data).(*unsafePipeline).eos.Unlock()
}

func (p *unsafePipeline) GetElement(name string) (*Element, error) {
	cstr := C.CString(name)
	defer C.free(unsafe.Pointer(cstr))
	element := C.gst_bin_get_by_name((*C.GstBin)(unsafe.Pointer(p.element)), cstr)
	if element == nil {
		return nil, errors.New("failed to get element")
	}
	return &Element{gst: element}, nil
}

type Sample struct {
	Data     []byte
	PTS      *time.Duration
	DTS      *time.Duration
	Duration *time.Duration
	Offset   int
}

func (p *unsafePipeline) Read(buf []byte) (int, error) {
	buffer, err := p.ReadSample()
	if err != nil {
		return 0, err
	}
	if len(buffer.Data) > len(buf) {
		return 0, io.ErrShortBuffer
	}
	return copy(buf, buffer.Data), nil
}

func (p *unsafePipeline) ReadSample() (*Sample, error) {
	sample := C.gst_app_sink_pull_sample(p.sink)
	if sample == nil {
		return nil, io.EOF
	}
	defer C.gst_sample_unref(sample)

	cbuf := C.gst_sample_get_buffer(sample)
	if cbuf == nil {
		return nil, io.ErrUnexpectedEOF
	}

	var copy C.gpointer
	var size C.ulong

	C.gst_buffer_extract_dup(cbuf, C.ulong(0), C.gst_buffer_get_size(cbuf), &copy, &size)

	defer C.free(unsafe.Pointer(copy))

	var duration, pts, dts *time.Duration
	if time.Duration(cbuf.duration) != C.GST_CLOCK_TIME_NONE {
		d := (time.Duration(cbuf.duration) * time.Nanosecond)
		duration = &d
	}
	if time.Duration(cbuf.pts) != C.GST_CLOCK_TIME_NONE {
		p := (time.Duration(cbuf.pts) * time.Nanosecond)
		pts = &p
	}
	if time.Duration(cbuf.dts) != C.GST_CLOCK_TIME_NONE {
		d := (time.Duration(cbuf.dts) * time.Nanosecond)
		dts = &d
	}

	return &Sample{
		Data:     C.GoBytes(unsafe.Pointer(copy), C.int(size)),
		Duration: duration,
		PTS:      pts,
		DTS:      dts,
		Offset:   int(cbuf.offset),
	}, nil
}

func (p *unsafePipeline) ReadRTP() (*rtp.Packet, error) {
	pkt := &rtp.Packet{}
	buf := make([]byte, 1500)
	if _, err := p.Read(buf); err != nil {
		return nil, err
	}
	if err := pkt.Unmarshal(buf); err != nil {
		return nil, err
	}
	return pkt, nil
}

func (p *unsafePipeline) Write(buf []byte) (int, error) {
	if err := p.WriteSample(&Sample{Data: buf}); err != nil {
		return 0, err
	}
	return len(buf), nil
}

func (p *unsafePipeline) WriteSample(b *Sample) error {
	cbuf := C.CBytes(b.Data)
	defer C.free(cbuf)
	cpbuf := C.g_memdup_compat(C.gconstpointer(cbuf), C.ulong(len(b.Data)))
	gstbuf := C.gst_buffer_new_wrapped(cpbuf, C.ulong(len(b.Data)))

	if b.PTS != nil {
		gstbuf.pts = C.GstClockTime(b.PTS.Nanoseconds())
	}
	if b.DTS != nil {
		gstbuf.dts = C.GstClockTime(b.DTS.Nanoseconds())
	}
	if b.Duration != nil {
		gstbuf.duration = C.GstClockTime(b.Duration.Nanoseconds())
	}

	if C.gst_app_src_push_buffer(p.source, gstbuf) != C.GST_FLOW_OK {
		return errors.New("failed to push buffer to source")
	}
	return nil
}

func (p *unsafePipeline) WriteRTP(pkt *rtp.Packet) error {
	buf, err := pkt.Marshal()
	if err != nil {
		return err
	}
	// TODO: does the PTS need to be set here? not sure, most depayloaders seem ok.
	return p.WriteSample(&Sample{
		Data: buf,
	})
}

func (p *unsafePipeline) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	if C.gst_element_set_state(p.element, C.GST_STATE_NULL) == C.GST_STATE_CHANGE_FAILURE {
		return errors.New("failed to set element to NULL")
	}
	if C.gst_bin_remove(p.synchronizer.bin, p.element) == 0 {
		return errors.New("failed to remove element from bin")
	}
	return nil
}

func (p *unsafePipeline) SendEndOfStream() error {
	if p.source == nil {
		return errors.New("no source")
	}
	if C.gst_app_src_end_of_stream(p.source) != C.GST_FLOW_OK {
		return errors.New("failed to end stream")
	}
	return nil
}

func (p *unsafePipeline) WaitForEndOfStream() {
	p.eos.Lock()
	defer p.eos.Unlock()
}
