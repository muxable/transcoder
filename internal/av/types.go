package av

/*
#cgo pkg-config: libavcodec
#include <libavcodec/avcodec.h>
*/
import "C"

// These are useful to avoid leaking the cgo interface.

type AVPacket struct {
	packet *C.AVPacket
}

func NewAVPacket() *AVPacket {
	packet := C.av_packet_alloc()
	if packet == nil {
		return nil
	}
	return &AVPacket{packet: packet}
}

func (p *AVPacket) Close() error {
	C.av_packet_free(&p.packet)
	return nil
}

type AVFrame struct {
	frame *C.AVFrame
}

func NewAVFrame() *AVFrame {
	frame := C.av_frame_alloc()
	if frame == nil {
		return nil
	}
	return &AVFrame{frame: frame}
}

func (f *AVFrame) Close() error {
	C.av_frame_free(&f.frame)
	return nil
}