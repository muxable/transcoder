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
	"net"
	"runtime"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/mattn/go-pointer"
	"github.com/pion/rtpio/pkg/rtpio"
	"go.uber.org/zap"
)

func init() {
	C.gst_init(nil, nil)
}

type Synchronizer struct {
	id   uuid.UUID
	bin  *C.GstBin
	ctx  *C.GMainContext
	loop *C.GMainLoop

	closed bool
}

func NewSynchronizer() (*Synchronizer, error) {
	id := uuid.New()

	cname := C.CString(id.String())
	defer C.free(unsafe.Pointer(cname))

	pipeline := C.gst_pipeline_new(cname)

	if C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING) == C.GST_STATE_CHANGE_FAILURE {
		return nil, errors.New("failed to set pipeline to playing")
	}
	ctx := C.g_main_context_new()
	loop := C.g_main_loop_new(ctx, C.int(0))
	watch := C.gst_bus_create_watch(C.gst_pipeline_get_bus((*C.GstPipeline)(unsafe.Pointer(pipeline))))

	s := &Synchronizer{
		id:   id,
		bin:  (*C.GstBin)(unsafe.Pointer(pipeline)),
		ctx:  ctx,
		loop: loop,
	}

	C.g_source_set_callback(watch, C.GSourceFunc(C.cgoBusFunc), C.gpointer(pointer.Save(s)), nil)

	if C.g_source_attach(watch, ctx) == 0 {
		return nil, errors.New("failed to add bus watch")
	}
	defer C.g_source_unref(watch)

	go C.g_main_loop_run(loop)

	runtime.SetFinalizer(s, func(synchronizer *Synchronizer) {
		if err := synchronizer.Close(); err != nil {
			zap.L().Error("failed to close synchronizer", zap.Error(err))
		}
		C.gst_object_unref(C.gpointer(unsafe.Pointer(synchronizer.bin)))
		C.g_main_loop_unref(synchronizer.loop)
		C.g_main_context_unref(synchronizer.ctx)
	})

	return s, nil
}

func (s *Synchronizer) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if C.gst_element_set_state((*C.GstElement)(unsafe.Pointer(s.bin)), C.GST_STATE_NULL) == C.GST_STATE_CHANGE_FAILURE {
		return errors.New("failed to set pipeline to null")
	}
	C.g_main_loop_quit(s.loop)
	return nil
}

//export goBusFunc
func goBusFunc(bus *C.GstBus, msg *C.GstMessage, ptr C.gpointer) C.gboolean {
	s := pointer.Restore(unsafe.Pointer(ptr)).(*Synchronizer)
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
		zap.L().Error(C.GoString(gerr.message), zap.String("id", s.id.String()), zap.String("debug", C.GoString(debugInfo)))
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
		zap.L().Warn(C.GoString(gerr.message), zap.String("id", s.id.String()), zap.String("debug", C.GoString(debugInfo)))
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
		zap.L().Info(C.GoString(gerr.message), zap.String("id", s.id.String()), zap.String("debug", C.GoString(debugInfo)))
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
		zap.L().Debug(C.GoString(C.gst_message_type_get_name(msg._type)), zap.String("id", s.id.String()), zap.Uint32("seqnum", uint32(msg.seqnum)))
	}
	return C.gboolean(1)
}

type Pipeline interface {
	GetElement(name string) (*Element, error)
}

type ReadOnlyPipeline interface {
	Pipeline
	io.Reader
	rtpio.RTPReader
	ReadBuffer() (*Buffer, error)
}

type WriteOnlyPipeline interface {
	Pipeline
	io.WriteCloser
	rtpio.RTPWriteCloser
	// WriteBuffer(*Buffer) error
}

type ReadWritePipeline interface {
	ReadOnlyPipeline
	WriteOnlyPipeline
}

func unusedPort() (int, error) {
	// get an unused udp port.
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port, nil
}

func (s *Synchronizer) NewPipeline(str string) (Pipeline, error) {
	return newUnsafePipeline(s, str, nil)
}

func (s *Synchronizer) NewReadOnlyPipeline(str string) (ReadOnlyPipeline, error) {
	return newUnsafePipeline(s, fmt.Sprintf("%s ! queue ! appsink name=internal-sink sync=false async=false", str), nil)
}

func (s *Synchronizer) NewWriteOnlyPipeline(str string) (WriteOnlyPipeline, error) {
	writePort, err := unusedPort()
	if err != nil {
		return nil, err
	}
	return newUnsafePipeline(s, fmt.Sprintf("udpsrc address=127.0.0.1 port=%d name=internal-source ! queue ! %s", writePort, str), &writePort)
}

func (s *Synchronizer) NewReadWritePipeline(str string) (ReadWritePipeline, error) {
	writePort, err := unusedPort()
	if err != nil {
		return nil, err
	}
	return newUnsafePipeline(s, fmt.Sprintf("udpsrc address=127.0.0.1 port=%d name=internal-source ! queue ! %s ! queue ! appsink name=internal-sink sync=false async=false", writePort, str), &writePort)
}