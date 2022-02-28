package av

/*
#cgo pkg-config: libavcodec libavformat
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include "mux.h"
*/
import "C"
import (
	"errors"
	"io"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

var (
	crtp = C.CString("rtp")
)

type result struct {
	p *rtp.Packet
	e error
}

type MuxContext struct {
	codec       webrtc.RTPCodecCapability
	avformatctx *C.AVFormatContext
	packet      *AVPacket
	encoder     *EncodeContext
	pch         chan *result
}

func NewMuxer(codec webrtc.RTPCodecCapability, encoder *EncodeContext) *MuxContext {
	c := &MuxContext{
		codec:   codec,
		packet:  NewAVPacket(),
		encoder: encoder,
		pch:     make(chan *result),
	}
	go func() {
		defer close(c.pch)
		if err := c.init(); err != nil {
			c.pch <- &result{e: err}
			return
		}
		for {
			if err := c.encoder.ReadAVPacket(c.packet); err != nil {
			c.pch <- &result{e: err}
			close(c.pch)
				return
			}
			if averr := C.av_write_frame(c.avformatctx, c.packet.packet); averr < 0 {
				c.pch <- &result{e: av_err("av_write_frame", averr)}
				return
			}
		}
	}()
	return c
}

//export goWritePacketFunc
func goWritePacketFunc(opaque unsafe.Pointer, buf *C.uint8_t, bufsize C.int) C.int {
	m := pointer.Restore(opaque).(*MuxContext)
	b := C.GoBytes(unsafe.Pointer(buf), bufsize)
	p := &rtp.Packet{}
	if err := p.Unmarshal(b); err != nil {
		zap.L().Error("failed to unmarshal rtp packet", zap.Error(err))
		return C.int(-1)
	}
	m.pch <- &result{p: p}
	return bufsize
}

func (c *MuxContext) init() error {
	if err := c.encoder.init(); err != nil {
		return err
	}

	outputformat := C.av_guess_format(crtp, nil, nil)
	if outputformat == nil {
		return errors.New("failed to find rtp output format")
	}

	buf := C.av_malloc(1500)
	if buf == nil {
		return errors.New("failed to allocate buffer")
	}

	var avformatctx *C.AVFormatContext

	if averr := C.avformat_alloc_output_context2(&avformatctx, outputformat, nil, nil); averr < 0 {
		return av_err("avformat_alloc_output_context2", averr)
	}

	avioctx := C.avio_alloc_context((*C.uchar)(buf), 1500, 1, pointer.Save(c), nil, (*[0]byte)(C.cgoWritePacketFunc), nil)
	if avioctx == nil {
		return errors.New("failed to create avio context")
	}

	avioctx.max_packet_size = 1200

	avformatctx.pb = avioctx

	avformatstream := C.avformat_new_stream(avformatctx, c.encoder.encoderctx.codec)
	if avformatstream == nil {
		return errors.New("failed to create rtp stream")
	}

	if averr := C.avcodec_parameters_from_context(avformatstream.codecpar, c.encoder.encoderctx); averr < 0 {
		return av_err("avcodec_parameters_from_context", averr)
	}

	if averr := C.avformat_write_header(avformatctx, nil); averr < 0 {
		return av_err("avformat_write_header", averr)
	}

	c.avformatctx = avformatctx

	return nil
}

func (c *MuxContext) ReadRTP() (*rtp.Packet, error) {
	r, ok := <-c.pch
	if !ok {
		return nil, io.EOF
	}
	return r.p, r.e
}
