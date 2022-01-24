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

// strange that cgo doesn't like these inlined...
void X_gst_g_object_set_string(GstElement *e, const gchar* p_name, gchar* p_value) {
  g_object_set(G_OBJECT(e), p_name, p_value, NULL);
}

void X_gst_g_object_set_int(GstElement *e, const gchar* p_name, gint p_value) {
  g_object_set(G_OBJECT(e), p_name, p_value, NULL);
}

void X_gst_g_object_set_uint(GstElement *e, const gchar* p_name, guint p_value) {
  g_object_set(G_OBJECT(e), p_name, p_value, NULL);
}

void X_gst_g_object_set_bool(GstElement *e, const gchar* p_name, gboolean p_value) {
  g_object_set(G_OBJECT(e), p_name, p_value, NULL);
}

void X_gst_g_object_set_gdouble(GstElement *e, const gchar* p_name, gdouble p_value) {
  g_object_set(G_OBJECT(e), p_name, p_value, NULL);
}

void X_gst_g_object_set_caps(GstElement *e, const gchar* p_name, const GstCaps *p_value) {
  g_object_set(G_OBJECT(e), p_name, p_value, NULL);
}

void X_gst_g_object_set_structure(GstElement *e, const gchar* p_name, const GstStructure *p_value) {
  g_object_set(G_OBJECT(e), p_name, p_value, NULL);
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

func NewElement(s string, properties ...Property) (*Element, error) {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	CGstElement := C.gst_element_factory_make(cs, nil)
	if CGstElement == nil {
		return nil, errors.New("could not create element")
	}
	e := &Element{GstElement: CGstElement}
	e.Set(properties...)
	return e, nil
}

func Link(elements ...*Element) error {
	for i := 0; i < len(elements)-1; i++ {
		if C.gst_element_link(elements[i].GstElement, elements[i+1].GstElement) != C.gboolean(1) {
			return errors.New("could not link elements")
		}
	}
	return nil
}

func (e *Element) Set(properties ...Property) {
	for _, p := range properties {
		cname := (*C.gchar)(unsafe.Pointer(C.CString(p.Name)))
		defer C.g_free(C.gpointer(unsafe.Pointer(cname)))
		switch value := p.Value.(type) {
		case string:
			str := (*C.gchar)(unsafe.Pointer(C.CString(value)))
			defer C.g_free(C.gpointer(unsafe.Pointer(str)))
			C.X_gst_g_object_set_string(e.GstElement, cname, str)
		case int:
			C.X_gst_g_object_set_int(e.GstElement, cname, C.gint(value))
		case uint32:
			C.X_gst_g_object_set_uint(e.GstElement, cname, C.guint(value))
		case bool:
			var cvalue int
			if value {
				cvalue = 1
			}
			C.X_gst_g_object_set_bool(e.GstElement, cname, C.gboolean(cvalue))
		case float64:
			C.X_gst_g_object_set_gdouble(e.GstElement, cname, C.gdouble(value))
		case *Caps:
			C.X_gst_g_object_set_caps(e.GstElement, cname, value.caps)
		// case *Structure:
		// 	structure := value.(*Structure)
		// 	C.X_gst_g_object_set_structure(e.GstElement, cname, structure.C)
		default:
			panic(fmt.Errorf("SetObject: don't know how to set value for type %T", value))
		}
	}
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
