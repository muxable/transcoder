package av

/*
#cgo pkg-config: libavcodec libavformat
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
*/
import "C"
import (
	"errors"
	"io"
	"strings"
	"unsafe"

	"github.com/pion/webrtc/v3"
)

type DecodeContext struct {
	codec      webrtc.RTPCodecParameters
	decoderctx *C.AVCodecContext
	pkt        *AVPacket
	demuxer    *DemuxContext
}

func NewDecoder(codec webrtc.RTPCodecParameters, demuxer *DemuxContext) *DecodeContext {
	return &DecodeContext{
		codec:   codec,
		pkt:     NewAVPacket(),
		demuxer: demuxer,
	}
}

func (c *DecodeContext) init() error {
	if err := c.demuxer.init(); err != nil {
		return err
	}

	var decodercodec *C.AVCodec
	var kind int32
	if strings.HasPrefix(c.codec.MimeType, "video") {
		kind = C.AVMEDIA_TYPE_VIDEO
	} else if strings.HasPrefix(c.codec.MimeType, "audio") {
		kind = C.AVMEDIA_TYPE_AUDIO
	} else {
		kind = C.AVMEDIA_TYPE_UNKNOWN
	}

	ret := C.av_find_best_stream(c.demuxer.avformatctx, kind, -1, -1, &decodercodec, 0)
	if ret < 0 {
		return av_err("av_find_best_stream", ret)
	}

	decoderctx := C.avcodec_alloc_context3(decodercodec)
	if decoderctx == nil {
		return errors.New("failed to create decoder context")
	}

	if averr := C.avcodec_parameters_to_context(decoderctx, ((*[1 << 30]*C.AVStream)(unsafe.Pointer(c.demuxer.avformatctx.streams)))[ret].codecpar); averr < 0 {
		return av_err("avcodec_parameters_to_context", averr)
	}

	if averr := C.avcodec_open2(decoderctx, decodercodec, nil); averr < 0 {
		return av_err("avcodec_open2", averr)
	}

	c.decoderctx = decoderctx

	return nil
}

func (c *DecodeContext) ReadAVFrame(f *AVFrame) error {
	if res := C.avcodec_receive_frame(c.decoderctx, f.frame); res < 0 {
		if res == -11 { // eagain
			err := c.demuxer.ReadAVPacket(c.pkt)
			if err != nil && err != io.EOF {
				return err
			}

			if averr := C.avcodec_send_packet(c.decoderctx, c.pkt.packet); averr < 0 {
				return av_err("avcodec_send_packet", averr)
			}

			// try again.
			return c.ReadAVFrame(f)
		}
		C.avcodec_free_context(&c.decoderctx)
		if err := c.pkt.Close(); err != nil {
			return err
		}
		return av_err("failed to receive frame", res)
	}
	return nil
}
