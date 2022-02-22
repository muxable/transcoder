package av

/*
#cgo pkg-config: libavcodec

#include <libavcodec/avcodec.h>
*/
import "C"
import (
	"errors"
	"io"

	"github.com/pion/webrtc/v3"
)

type EncodeContext struct {
	encoderctx *C.AVCodecContext

	codec   webrtc.RTPCodecCapability
	decoder *DecodeContext
	outBuf  []*C.AVPacket
	draining bool
}

func NewEncoder(codec webrtc.RTPCodecCapability, decoder *DecodeContext) *EncodeContext {
	return &EncodeContext{
		codec:   codec,
		decoder: decoder,
	}
}

func (c *EncodeContext) initEncoder() error {
	encodercodec := C.avcodec_find_encoder(AvCodec[c.codec.MimeType])
	if encodercodec == nil {
		return errors.New("failed to start encoder")
	}

	encoderctx := C.avcodec_alloc_context3(encodercodec)
	if encoderctx == nil {
		return errors.New("failed to create encoder context")
	}

	encoderctx.channels = c.decoder.decoderctx.channels
	encoderctx.channel_layout = c.decoder.decoderctx.channel_layout
	encoderctx.sample_rate = C.int(c.codec.ClockRate)
	encoderctx.sample_fmt = c.decoder.decoderctx.sample_fmt
	encoderctx.width = c.decoder.decoderctx.width
	encoderctx.height = c.decoder.decoderctx.height
	encoderctx.pix_fmt = C.AV_PIX_FMT_YUV420P
	encoderctx.time_base = C.av_make_q(C.int(1), C.int(c.codec.ClockRate))

	if C.avcodec_open2(encoderctx, encodercodec, nil) < 0 {
		return errors.New("failed to open encoder")
	}

	encoderctx.rc_buffer_size = 4 * 1000 * 1000
	encoderctx.rc_max_rate = 20 * 1000 * 1000
	encoderctx.rc_min_rate = 1 * 1000 * 1000

	c.encoderctx = encoderctx

	return nil
}

func (c *EncodeContext) ReadAVPacket() (*C.AVPacket, error) {
	if len(c.outBuf) > 0 {
		packet := c.outBuf[0]
		c.outBuf = c.outBuf[1:]
		return packet, nil
	} else if c.draining {
		return nil, io.EOF
	}

	frame, err := c.decoder.ReadAVFrame()
	if err != nil && err != io.EOF {
		return nil, err
	}

	if frame != nil {
		defer C.av_frame_free(&frame)
	} else {
		c.draining = true
	}

	if c.encoderctx == nil {
		if err := c.initEncoder(); err != nil {
			return nil, err
		}
	}

	if res := C.avcodec_send_frame(c.encoderctx, frame); res < 0 {
		return nil, av_err("failed to send frame", res)
	}

	for {
		packet := C.av_packet_alloc()
		if res := C.avcodec_receive_packet(c.encoderctx, packet); res < 0 {
			if res == -11 { // eagain
				break
			}
			err := av_err("failed to receive packet", res)
			C.avcodec_free_context(&c.encoderctx)
			if err == io.EOF {
				break
			}
			return nil, err
		}
		c.outBuf = append(c.outBuf, packet)
	}

	return c.ReadAVPacket()
}
