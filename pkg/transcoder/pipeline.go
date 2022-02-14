package transcoder

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
	"runtime"
	"time"
	"unsafe"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

var (
	csink = C.CString("sink")
)

type unsafePipeline struct {
	element    *C.GstElement
	transcoder *Transcoder

	source *C.GstAppSrc
	sink   *C.GstAppSink
}

func (t *Transcoder) newUnsafePipeline(pipeline string) (*unsafePipeline, error) {
	log.Printf("%v", pipeline)

	cstr := C.CString(pipeline)
	defer C.free(unsafe.Pointer(cstr))

	var gerr *C.GError
	element := C.gst_parse_bin_from_description(cstr, C.int(0), (**C.GError)(&gerr))

	if gerr != nil {
		defer C.g_error_free((*C.GError)(gerr))
		errMsg := C.GoString(gerr.message)
		return nil, errors.New(errMsg)
	}

	if C.gst_bin_add(t.bin, element) == 0 {
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
		element:    element,
		source:     (*C.GstAppSrc)(unsafe.Pointer(source)),
		sink:       (*C.GstAppSink)(unsafe.Pointer(sink)),
		transcoder: t,
	}

	runtime.SetFinalizer(p, func(pipeline *unsafePipeline) {
		if C.gst_element_set_state(pipeline.element, C.GST_STATE_NULL) == C.GST_STATE_CHANGE_FAILURE {
			zap.L().Error("failed to set pipeline to null")
		}
		if C.gst_bin_remove(pipeline.transcoder.bin, pipeline.element) == 0 {
			zap.L().Error("failed to remove bin from pipeline")
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

func (p *unsafePipeline) sinkCaps() (*Caps, error) {
	pad := C.gst_element_get_static_pad((*C.GstElement)(unsafe.Pointer(p.sink)), csink)
	if pad == nil {
		return nil, errors.New("failed to get src pad")
	}
	defer C.gst_object_unref(C.gpointer(pad))
	
	for {
		c := C.gst_pad_get_current_caps(pad)
		if c == nil {
			time.Sleep(1000 * time.Millisecond) // it would be nice to not poll for this.
			log.Printf("waiting for caps")
			continue
		}
		return &Caps{caps: c}, nil
	}
}

func (p *unsafePipeline) Codec() (*webrtc.RTPCodecParameters, error) {
	caps, err := p.sinkCaps()
	if err != nil {
		return nil, err
	}
	return caps.RTPCodecParameters()
}

func (p *unsafePipeline) SSRC() (webrtc.SSRC, error) {
	caps, err := p.sinkCaps()
	if err != nil {
		return 0, err
	}
	return caps.SSRC()
}

func (p *unsafePipeline) Read(buf []byte) (int, error) {
	sample := C.gst_app_sink_pull_sample(p.sink)
	if sample == nil {
		return 0, io.EOF
	}
	defer C.gst_sample_unref(sample)

	cbuf := C.gst_sample_get_buffer(sample)
	if cbuf == nil {
		return 0, io.ErrUnexpectedEOF
	}

	var c C.gpointer
	var size C.ulong

	C.gst_buffer_extract_dup(cbuf, C.ulong(0), C.gst_buffer_get_size(cbuf), &c, &size)
	defer C.free(unsafe.Pointer(c))

	data := C.GoBytes(unsafe.Pointer(c), C.int(size))
	if len(data) > len(buf) {
		return 0, io.ErrShortBuffer
	}
	return copy(buf, data), nil
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
	return pkt, nil
}

func (p *unsafePipeline) Write(buf []byte) (int, error) {
	b := C.CBytes(buf)
	defer C.free(b)

	ptr := C.g_memdup_compat(C.gconstpointer(b), C.ulong(len(buf)))
	data := C.gst_buffer_new_wrapped(ptr, C.ulong(len(buf)))

	gstReturn := C.gst_app_src_push_buffer(p.source, data)

	if gstReturn != C.GST_FLOW_OK {
		return 0, errors.New("could not push buffer on appsrc element")
	}
	return len(buf), nil
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
