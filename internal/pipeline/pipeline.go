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
	"log"
	"math"
	"net"
	"runtime"
	"time"
	"unsafe"

	"github.com/pion/rtp"
	"go.uber.org/zap"
)

type unsafePipeline struct {
	element      *C.GstElement
	synchronizer *Synchronizer

	source *C.GstAppSrc
	sink   *C.GstAppSink
	write  *net.UDPConn
}

func newUnsafePipeline(synchronizer *Synchronizer, pipeline string, write *int) (*unsafePipeline, error) {
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
	if write != nil {
		writeConn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: *write})
		if err != nil {
			return nil, err
		}
		p.write = writeConn
	}

	runtime.SetFinalizer(p, func(pipeline *unsafePipeline) {
		if C.gst_element_set_state(pipeline.element, C.GST_STATE_NULL) == C.GST_STATE_CHANGE_FAILURE {
			zap.L().Error("failed to set pipeline to null")
		}
		if C.gst_bin_remove(pipeline.synchronizer.bin, p.element) == 0 {
			zap.L().Error("failed to remove bin from pipeline")
		}
		if pipeline.source != nil {
			C.gst_object_unref(C.gpointer(unsafe.Pointer(pipeline.source)))
		}
		if pipeline.sink != nil {
			C.gst_object_unref(C.gpointer(unsafe.Pointer(pipeline.sink)))
		}
		if err := pipeline.write.Close(); err != nil {
			zap.L().Error("failed to close write connection", zap.Error(err))
		}
	})

	return p, nil
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

type Buffer struct {
	Data     []byte
	PTS      *time.Duration
	DTS      *time.Duration
	Duration *time.Duration
	Offset   int
}

func (p *unsafePipeline) Read(buf []byte) (int, error) {
	buffer, err := p.ReadBuffer()
	if err != nil {
		return 0, err
	}
	if len(buffer.Data) > len(buf) {
		return 0, io.ErrShortBuffer
	}
	return copy(buf, buffer.Data), nil
}

const GstClockTimeNone = math.MaxUint64

func (p *unsafePipeline) ReadBuffer() (*Buffer, error) {
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
	if cbuf.duration != GstClockTimeNone {
		d := (time.Duration(cbuf.duration) * time.Nanosecond)
		duration = &d
	}
	if cbuf.pts != GstClockTimeNone {
		p := (time.Duration(cbuf.pts) * time.Nanosecond)
		pts = &p
	}
	if cbuf.dts != GstClockTimeNone {
		d := (time.Duration(cbuf.dts) * time.Nanosecond)
		dts = &d
	}

	return &Buffer{
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
	n, err := p.Read(buf)
	if err != nil {
		return nil, err
	}
	if err := pkt.Unmarshal(buf[:n]); err != nil {
		return nil, err
	}
	log.Printf("%v %v", pkt.SequenceNumber, pkt.MarshalSize())
	return pkt, nil
}

func (p *unsafePipeline) Write(buf []byte) (int, error) {
	return p.write.Write(buf)
}

func (p *unsafePipeline) WriteRTP(pkt *rtp.Packet) error {
	buf, err := pkt.Marshal()
	if err != nil {
		return err
	}
	n, err := p.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return io.ErrShortWrite
	}
	return nil
}

func (p *unsafePipeline) Close() error {
	if p.source != nil {
		if C.gst_element_send_event((*C.GstElement)(unsafe.Pointer(p.source)), C.gst_event_new_eos()) != C.int(1) {
			return errors.New("failed to end stream")
		}
	}
	return nil
}
