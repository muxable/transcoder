package pipeline

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include <glib.h>
#include <gst/app/gstappsink.h>
#include <gst/app/gstappsrc.h>
#include <gst/gst.h>

#include "synchronizer.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"
	"unsafe"

	"github.com/pion/rtpio/pkg/rtpio"
	"go.uber.org/zap"
)

func init() {
	C.gst_init(nil, nil)
}

type Synchronizer struct {
	bin  *C.GstBin
	ctx  *C.GMainContext
	loop *C.GMainLoop
}

func NewSynchronizer() (*Synchronizer, error) {
	pipeline := C.gst_pipeline_new(nil)

	if C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING) == C.GST_STATE_CHANGE_FAILURE {
		return nil, errors.New("failed to set pipeline to playing")
	}
	ctx := C.g_main_context_new()
	loop := C.g_main_loop_new(ctx, C.int(0))
	watch := C.gst_bus_create_watch(C.gst_pipeline_get_bus((*C.GstPipeline)(unsafe.Pointer(pipeline))))

	C.g_source_set_callback(watch, C.GSourceFunc(C.cgoBusFunc), nil, nil)

	if C.g_source_attach(watch, ctx) == 0 {
		return nil, errors.New("failed to add bus watch")
	}
	defer C.g_source_unref(watch)

	s := &Synchronizer{
		bin:  (*C.GstBin)(unsafe.Pointer(pipeline)),
		ctx:  ctx,
		loop: loop,
	}

	go C.g_main_loop_run(loop)

	runtime.SetFinalizer(s, func(synchronizer *Synchronizer) {
		C.gst_object_unref(C.gpointer(unsafe.Pointer(synchronizer.bin)))
		C.g_main_loop_unref(synchronizer.loop)
		C.g_main_context_unref(synchronizer.ctx)
	})

	return s, nil
}

func (s *Synchronizer) Close() error {
	C.gst_element_set_state((*C.GstElement)(unsafe.Pointer(s.bin)), C.GST_STATE_NULL)
	C.g_main_loop_quit(s.loop)
	return nil
}

//export goBusFunc
func goBusFunc(bus *C.GstBus, msg *C.GstMessage) C.gboolean {
	switch msg._type {
	case C.GST_MESSAGE_ERROR:
		var gerr *C.GError
		var debugInfo *C.gchar
		defer func() {
			if gerr != nil {
				C.g_error_free(gerr)
			}
		}()
		defer C.g_free(C.gpointer(unsafe.Pointer(debugInfo)))
		C.gst_message_parse_error(msg, (**C.GError)(unsafe.Pointer(&gerr)), (**C.gchar)(unsafe.Pointer(&debugInfo)))
		zap.L().Error(C.GoString(gerr.message), zap.String("debug", C.GoString(debugInfo)))
	case C.GST_MESSAGE_WARNING:
		var gerr *C.GError
		var debugInfo *C.gchar
		defer func() {
			if gerr != nil {
				C.g_error_free(gerr)
			}
		}()
		defer C.g_free(C.gpointer(unsafe.Pointer(debugInfo)))
		C.gst_message_parse_warning(msg, (**C.GError)(unsafe.Pointer(&gerr)), (**C.gchar)(unsafe.Pointer(&debugInfo)))
		zap.L().Warn(C.GoString(gerr.message), zap.String("debug", C.GoString(debugInfo)))
	case C.GST_MESSAGE_INFO:
		var gerr *C.GError
		var debugInfo *C.gchar
		defer func() {
			if gerr != nil {
				C.g_error_free(gerr)
			}
		}()
		defer C.g_free(C.gpointer(unsafe.Pointer(debugInfo)))
		C.gst_message_parse_info(msg, (**C.GError)(unsafe.Pointer(&gerr)), (**C.gchar)(unsafe.Pointer(&debugInfo)))
		zap.L().Info(C.GoString(gerr.message), zap.String("debug", C.GoString(debugInfo)))
	case C.GST_MESSAGE_QOS:
		var live C.gboolean
		var runningTime, streamTime, timestamp, duration C.guint64
		C.gst_message_parse_qos(msg, &live, &runningTime, &streamTime, &timestamp, &duration)
		zap.L().Info("QOS",
			zap.Bool("live", live != 0),
			zap.Duration("runningTime", time.Duration(runningTime)),
			zap.Duration("streamTime", time.Duration(streamTime)),
			zap.Duration("timestamp", time.Duration(timestamp)),
			zap.Duration("duration", time.Duration(duration)))
	default:
		zap.L().Debug("pipeline message", zap.String("type", C.GoString(C.gst_message_type_get_name(msg._type))))
	}
	return C.gboolean(1)
}

type ReadOnlyPipeline interface {
	io.ReadCloser
	rtpio.RTPReadCloser
	ReadSample() (*Sample, error)
	GetElement(name string) (*Element, error)
	WaitForEndOfStream()
}

type WriteOnlyPipeline interface {
	io.WriteCloser
	rtpio.RTPWriteCloser
	WriteSample(*Sample) error
	GetElement(name string) (*Element, error)
	SendEndOfStream() error
}

type ReadWritePipeline interface {
	ReadOnlyPipeline
	WriteOnlyPipeline
}

func (s *Synchronizer) NewReadOnlyPipeline(str string) (ReadOnlyPipeline, error) {
	return newUnsafePipeline(s, fmt.Sprintf("%s ! appsink name=internal-sink sync=false async=false", str))
}

func (s *Synchronizer) NewWriteOnlyPipeline(str string) (WriteOnlyPipeline, error) {
	return newUnsafePipeline(s, fmt.Sprintf("appsrc format=time name=internal-source ! %s", str))
}

func (s *Synchronizer) NewReadWritePipeline(str string) (ReadWritePipeline, error) {
	return newUnsafePipeline(s, fmt.Sprintf("appsrc format=time name=internal-source ! %s ! appsink name=internal-sink sync=false async=false", str))
}
