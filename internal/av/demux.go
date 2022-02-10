package av

/*
#cgo pkg-config: libavcodec libavformat

#include <libavcodec/avcodec.h>
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/muxable/rtptools/pkg/h265"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
)

type DemuxContext struct {
	codec   webrtc.RTPCodecCapability
	builder *samplebuilder.SampleBuilder
	in      rtpio.RTPReader
	outBuf  []*C.AVPacket
}

func depacketizer(mimeType string) rtp.Depacketizer {
	switch mimeType {
	case webrtc.MimeTypeH264:
		return &codecs.H264Packet{}
	case webrtc.MimeTypeH265:
		return &h265.H265Packet{}
	case webrtc.MimeTypeVP8:
		return &codecs.VP8Packet{}
	case webrtc.MimeTypeVP9:
		return &codecs.VP9Packet{}
	case webrtc.MimeTypeOpus:
		return &codecs.OpusPacket{}
	}
	return nil
}

func NewDemuxer(codec webrtc.RTPCodecCapability, in rtpio.RTPReader) *DemuxContext {
	builder := samplebuilder.New(10, depacketizer(codec.MimeType), codec.ClockRate)
	return &DemuxContext{
		codec:   codec,
		builder: builder,
		in:      in,
	}
}

func (c *DemuxContext) ReadAVPacket() (*C.AVPacket, error) {
	if len(c.outBuf) > 0 {
		pkt := c.outBuf[0]
		c.outBuf = c.outBuf[1:]
		return pkt, nil
	}
	p, err := c.in.ReadRTP()
	if err != nil {
		return nil, err
	}
	c.builder.Push(p)
	for {
		sample := c.builder.Pop()
		if sample == nil {
			break
		}
		avpacket := C.av_packet_alloc()
		if avpacket == nil {
			return nil, errors.New("failed to allocate packet")
		}
		avpacket.pts = C.AV_NOPTS_VALUE
		avpacket.dts = C.long(sample.PacketTimestamp)
		avpacket.duration = C.long(sample.Duration.Seconds() * float64(c.codec.ClockRate))
		avpacket.data = (*C.uint8_t)(unsafe.Pointer(&sample.Data[0]))
		avpacket.size = C.int(len(sample.Data))
		c.outBuf = append(c.outBuf, avpacket)
	}
	return c.ReadAVPacket()
}
