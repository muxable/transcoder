package gst

/*
#cgo pkg-config: gstreamer-1.0
#include <gst/gst.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"io"
	"unsafe"
)

type Sample struct {
	GstSample *C.GstSample
}

func (s *Sample) MarshalTo(buf []byte) error {
	gstBuffer := C.gst_sample_get_buffer(s.GstSample)

	if gstBuffer == nil {
		return errors.New("could not pull a sample from appsink")
	}

	mapInfo := (*C.GstMapInfo)(unsafe.Pointer(C.malloc(C.sizeof_GstMapInfo)))
	defer C.free(unsafe.Pointer(mapInfo))

	if int(C.gst_buffer_map(gstBuffer, mapInfo, C.GST_MAP_READ)) == 0 {
		return fmt.Errorf("could not map gstBuffer %#v", gstBuffer)
	}
	defer C.gst_buffer_unmap(gstBuffer, mapInfo)

	CData := (*[1 << 30]byte)(unsafe.Pointer(mapInfo.data))
	data := CData[:int(mapInfo.size)]
	if len(buf) < len(data) {
		return io.ErrShortBuffer
	}
	if copy(buf, data) != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func (s *Sample) Close() error {
	C.gst_object_unref(C.gpointer(unsafe.Pointer(s.GstSample)))
	return nil
}
