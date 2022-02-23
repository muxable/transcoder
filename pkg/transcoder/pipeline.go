package transcoder

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#cgo LDFLAGS: -lgstvideo-1.0

#include <glib.h>
#include <gst/app/gstappsink.h>
#include <gst/app/gstappsrc.h>
#include <gst/gst.h>
#include <gst/video/video-event.h>

#include "pipeline.h"
*/
import "C"
import (
	"errors"
	"io"
	"log"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

var (
	csrc            = C.CString("src")
	csink           = C.CString("sink")
	cinternalsource = C.CString("internal-source")
	cinternalsink   = C.CString("internal-sink")
)

type unsafePipeline struct {
	element    *C.GstElement
	transcoder *Transcoder

	source *C.GstAppSrc
	sink   *C.GstAppSink

	clockRate uint32
	t0        uint64
	absts     uint64
	prevts    uint32

	capsNegotiated       sync.WaitGroup
	forceKeyUnitCallback func()
}

var sinkmap = map[*C.GstElement]*unsafePipeline{}
var sourcemap = map[*C.GstElement]*unsafePipeline{}

func (t *Transcoder) newUnsafePipeline(pipeline string, clockRate uint32) (*unsafePipeline, error) {
	if clockRate == 0 {
		return nil, errors.New("clock rate must be non-zero")
	}
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

	source := C.gst_bin_get_by_name(bin, cinternalsource)
	sink := C.gst_bin_get_by_name(bin, cinternalsink)

	p := &unsafePipeline{
		element:        element,
		source:         (*C.GstAppSrc)(unsafe.Pointer(source)),
		sink:           (*C.GstAppSink)(unsafe.Pointer(sink)),
		transcoder:     t,
		clockRate:      clockRate,
		capsNegotiated: sync.WaitGroup{},
	}
	p.capsNegotiated.Add(1)

	if source != nil {
		sourcepad := C.gst_element_get_static_pad(source, csrc)
		C.gst_pad_set_event_function_full(sourcepad, C.GSourceFunc(C.cgoSourcePadEventFunc), nil, nil)
		sourcemap[source] = p
	}
	if sink != nil {
		sinkpad := C.gst_element_get_static_pad(sink, csink)
		C.gst_pad_set_event_function_full(sinkpad, C.GSourceFunc(C.cgoSinkPadEventFunc), nil, nil)
		sinkmap[sink] = p
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

//export goSourcePadEventFunc
func goSourcePadEventFunc(pad *C.GstPad, obj *C.GstObject, event *C.GstEvent) C.gboolean {
	if C.gst_video_event_is_force_key_unit(event) == C.gboolean(1) {
		if p := sourcemap[(*C.GstElement)(unsafe.Pointer(obj))]; p != nil && p.forceKeyUnitCallback != nil {
			p.forceKeyUnitCallback()
		} else {
			zap.L().Warn("force key unit callback not set or pipeline not found")
		}
	}
	return C.gboolean(1)
}

func (p *unsafePipeline) OnUpstreamForceKeyUnit(callback func()) {
	p.forceKeyUnitCallback = callback
}

//export goSinkPadEventFunc
func goSinkPadEventFunc(pad *C.GstPad, obj *C.GstObject, event *C.GstEvent) C.gboolean {
	if event._type == C.GST_EVENT_CAPS {
		if obj == nil {
			zap.L().Warn("got caps event with nil object")
			return C.gboolean(1)
		}

		sink := (*C.GstElement)(unsafe.Pointer(obj))
		if p := sinkmap[sink]; p != nil {
			delete(sinkmap, sink)
			time.Sleep(10 * time.Millisecond) // gstreamer sends two caps payloads. we typically want the second, but who knows...
			p.capsNegotiated.Done()
		} else {
			zap.L().Warn("got caps event with nil pipeline")
		}
	}
	return C.gboolean(1)
}

func (p *unsafePipeline) sinkCaps() (*Caps, error) {
	log.Printf("waiting for caps negotiation")
	p.capsNegotiated.Wait()
	log.Printf("done")
	pad := C.gst_element_get_static_pad((*C.GstElement)(unsafe.Pointer(p.sink)), csink)
	if pad == nil {
		return nil, errors.New("failed to get src pad")
	}
	defer C.gst_object_unref(C.gpointer(pad))

	c := C.gst_pad_get_current_caps(pad)
	if c == nil {
		return nil, errors.New("failed to get caps")
	}
	return &Caps{caps: c}, nil
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

func (p *unsafePipeline) WriteRTP(pkt *rtp.Packet) error {
	buf, err := pkt.Marshal()
	if err != nil {
		return err
	}
	cbuf := C.CBytes(buf)
	defer C.free(cbuf)

	ptr := C.g_memdup_compat(C.gconstpointer(cbuf), C.ulong(len(buf)))
	data := C.gst_buffer_new_wrapped(ptr, C.ulong(len(buf)))

	if p.absts == 0 {
		// first packet
		p.absts = uint64(pkt.Timestamp)
		p.t0 = p.absts
	} else {
		tsm := uint64(p.prevts)
		tsn := uint64(pkt.Timestamp)
		if tsm == tsn {
			// do nothing.
		} else if tsm < tsn && tsn-tsm < (1<<31) {
			p.absts += tsn - tsm
		} else if tsm > tsn && tsm-tsn >= (1<<31) {
			p.absts += (1 << 32) - tsm + tsn
		} else if tsm < tsn && tsn-tsm >= (1<<31) {
			p.absts -= tsm + (1 << 32) - tsn
		} else if tsm > tsn && tsm-tsn < (1<<31) {
			p.absts -= tsm - tsn
		} else {
			return errors.New("illegal state")
		}
	}
	p.prevts = pkt.Timestamp

	pts := uint64(p.absts-p.t0) * uint64(time.Second) / uint64(p.clockRate)
	data.pts = C.GstClockTime(pts)

	gstReturn := C.gst_app_src_push_buffer(p.source, data)

	if gstReturn != C.GST_FLOW_OK {
		return errors.New("could not push buffer on appsrc element")
	}
	return nil
}

func (p *unsafePipeline) Close() error {
	if p.source != nil {
		delete(sourcemap, (*C.GstElement)(unsafe.Pointer(p.source)))
		if C.gst_element_send_event((*C.GstElement)(unsafe.Pointer(p.source)), C.gst_event_new_eos()) != C.int(1) {
			return errors.New("failed to end stream")
		}
	}
	if p.sink != nil {
		delete(sinkmap, (*C.GstElement)(unsafe.Pointer(p.sink)))
	}
	return nil
}
