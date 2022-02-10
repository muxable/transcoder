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

type DecodeContext struct {
	decoderctx *C.AVCodecContext
	codec      webrtc.RTPCodecCapability
	demuxer    *DemuxContext
	outBuf     []*C.AVFrame
	draining    bool
}

func NewDecoder(codec webrtc.RTPCodecCapability, demuxer *DemuxContext) *DecodeContext {
	return &DecodeContext{codec: codec, demuxer: demuxer}
}

func (c *DecodeContext) initDecoder() error {
	decodercodec := C.avcodec_find_decoder(AvCodec[c.codec.MimeType])
	if decodercodec == nil {
		return errors.New("failed to start decoder")
	}

	decoderctx := C.avcodec_alloc_context3(decodercodec)
	if decoderctx == nil {
		return errors.New("failed to create decoder context")
	}
	decoderctx.time_base = C.av_make_q(C.int(1), C.int(c.codec.ClockRate))

	if C.avcodec_open2(decoderctx, decodercodec, nil) < 0 {
		return errors.New("failed to open decoder")
	}

	c.decoderctx = decoderctx

	return nil
}

func (c *DecodeContext) ReadAVFrame() (*C.AVFrame, error) {
	if len(c.outBuf) > 0 {
		frame := c.outBuf[0]
		c.outBuf = c.outBuf[1:]
		return frame, nil
	} else if c.draining {
		return nil, io.EOF
	}
	p, err := c.demuxer.ReadAVPacket()
	if err != nil && err != io.EOF {
		return nil, err
	}

	if p != nil {
		defer C.av_packet_free(&p)
	} else {
		c.draining = true
	}

	if c.decoderctx == nil {
		if err := c.initDecoder(); err != nil {
			return nil, err
		}
	}

	if res := C.avcodec_send_packet(c.decoderctx, p); res < 0 {
		return nil, av_err("failed to send packet", res)
	}

	// receive the frame from the decoder and send it to the encoder
	for {
		frame := C.av_frame_alloc()
		if res := C.avcodec_receive_frame(c.decoderctx, frame); res < 0 {
			if res == -11 { // eagain
				break
			}
			err := av_err("failed to receive frame", res)
			C.avcodec_free_context(&c.decoderctx)
			if err == io.EOF {
				break
			}
			return nil, err
		}
		frame.pts = frame.best_effort_timestamp
		c.outBuf = append(c.outBuf, frame)
	}

	return c.ReadAVFrame()
}
