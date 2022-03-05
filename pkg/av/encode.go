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

	"github.com/pion/webrtc/v3"
)

type EncodeContext struct {
	codec        webrtc.RTPCodecCapability
	encoderctx   *C.AVCodecContext
	frame        *AVFrame
	decoder      *DecodeContext
}

func NewEncoder(codec webrtc.RTPCodecCapability, decoder *DecodeContext) *EncodeContext {
	return &EncodeContext{
		codec:   codec,
		frame:   NewAVFrame(),
		decoder: decoder,
	}
}

func (c *EncodeContext) init() error {
	if err := c.decoder.init(); err != nil {
		return err
	}

	decoderctx := c.decoder.decoderctx

	encodercodec := C.avcodec_find_encoder(AvCodec[c.codec.MimeType])
	if encodercodec == nil {
		return errors.New("failed to start encoder")
	}

	encoderctx := C.avcodec_alloc_context3(encodercodec)
	if encoderctx == nil {
		return errors.New("failed to create encoder context")
	}

	encoderctx.channels = decoderctx.channels
	encoderctx.channel_layout = decoderctx.channel_layout
	encoderctx.sample_rate = C.int(c.codec.ClockRate)
	encoderctx.sample_fmt = decoderctx.sample_fmt
	encoderctx.width = decoderctx.width
	encoderctx.height = decoderctx.height
	encoderctx.pix_fmt = C.AV_PIX_FMT_YUV420P
	encoderctx.time_base = C.av_make_q(C.int(1), C.int(c.codec.ClockRate))

	var opts *C.AVDictionary
	defer C.av_dict_free(&opts)

	if c.codec.MimeType == webrtc.MimeTypeH264 {
		if averr := C.av_dict_set(&opts, C.CString("preset"), C.CString("ultrafast"), 0); averr < 0 {
			return av_err("av_dict_set", averr)
		}
		if averr := C.av_dict_set(&opts, C.CString("tune"), C.CString("zerolatency"), 0); averr < 0 {
			return av_err("av_dict_set", averr)
		}
		if averr := C.av_dict_set(&opts, C.CString("profile"), C.CString("baseline"), 0); averr < 0 {
			return av_err("av_dict_set", averr)
		}
	}

	if averr := C.avcodec_open2(encoderctx, encodercodec, &opts); averr < 0 {
		return av_err("avcodec_open2", averr)
	}

	encoderctx.rc_buffer_size = 4 * 1000 * 1000
	encoderctx.rc_max_rate = 20 * 1000 * 1000
	encoderctx.rc_min_rate = 1 * 1000 * 1000

	c.encoderctx = encoderctx

	return nil
}

func (c *EncodeContext) ReadAVPacket(p *AVPacket) error {
	if res := C.avcodec_receive_packet(c.encoderctx, p.packet); res < 0 {
		if res == AVERROR(C.EAGAIN) {
			err := c.decoder.ReadAVFrame(c.frame)
			if err != nil && err != io.EOF {
				return err
			}

			if c.frame.frame.pts != C.AV_NOPTS_VALUE {
				if res := C.avcodec_send_frame(c.encoderctx, c.frame.frame); res < 0 {
					return av_err("avcodec_send_frame", res)
				}
			}

			// try again.
			return c.ReadAVPacket(p)
		}
		C.avcodec_free_context(&c.encoderctx)
		if err := c.frame.Close(); err != nil {
			return err
		}
		return av_err("avcodec_receive_packet", res)
	}
	return nil
}
