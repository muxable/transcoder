package av

/*
#cgo pkg-config: libavcodec

#include <libavcodec/avcodec.h>
*/
import "C"
import (
	"math/rand"
	"unsafe"

	"github.com/muxable/rtptools/pkg/h265"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
)

type MuxContext struct {
	codec   webrtc.RTPCodecCapability
	encoder *EncodeContext
	outBuf  []*rtp.Packet

	payloader rtp.Payloader
	sequencer rtp.Sequencer
	ssrc      webrtc.SSRC
}

func payloader(mimeType string) rtp.Payloader {
	switch mimeType {
	case webrtc.MimeTypeH264:
		return &codecs.H264Payloader{}
	case webrtc.MimeTypeH265:
		return &h265.H265Payloader{}
	case webrtc.MimeTypeVP8:
		return &codecs.VP8Payloader{}
	case webrtc.MimeTypeVP9:
		return &codecs.VP9Payloader{}
	case webrtc.MimeTypeOpus:
		return &codecs.OpusPayloader{}
	case webrtc.MimeTypeG722:
		return &codecs.G722Payloader{}
	}
	return nil
}

func NewMuxer(codec webrtc.RTPCodecCapability, encoder *EncodeContext) *MuxContext {
	// we could use the ffmpeg payloaders here but pion provides payloaders for the types that
	// we are most likely to output and ffmpeg's API isn't great for muxing.
	return &MuxContext{
		codec:     codec,
		encoder:   encoder,
		payloader: payloader(codec.MimeType),
		sequencer: rtp.NewRandomSequencer(),
		ssrc:      webrtc.SSRC(rand.Uint32()),
	}
}

func (c *MuxContext) ReadRTP() (*rtp.Packet, error) {
	if len(c.outBuf) > 0 {
		packet := c.outBuf[0]
		c.outBuf = c.outBuf[1:]
		return packet, nil
	}

	p, err := c.encoder.ReadAVPacket()
	if err != nil {
		return nil, err
	}
	defer C.av_packet_free(&p)

	data := C.GoBytes(unsafe.Pointer(p.data), C.int(p.size))

	payloads := c.payloader.Payload(1200-12, data)
	for i, pp := range payloads {
		q := &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				Padding:        false,
				Extension:      false,
				Marker:         i == len(payloads)-1,
				SequenceNumber: c.sequencer.NextSequenceNumber(),
				Timestamp:      uint32(p.dts), // Figure out how to do timestamps
				SSRC:           uint32(c.ssrc),
			},
			Payload: pp,
		}
		c.outBuf = append(c.outBuf, q)
	}
	return c.ReadRTP()
}
