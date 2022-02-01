package codecs

import (
	"math/rand"
	"time"

	"github.com/pion/rtp"
)

type packetizer struct {
	MTU              uint16
	Payloader        rtp.Payloader
	Sequencer        rtp.Sequencer
	TimestampOffset  uint32
	extensionNumbers struct { // put extension numbers in here. If they're 0, the extension is disabled (0 is not a legal extension number)
		AbsSendTime int // http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
	}
	timegen func() time.Time
}

// NewTSPacketizer returns a new instance of a Packetizer for a specific payloader
func NewTSPacketizer(mtu uint16, payloader rtp.Payloader, sequencer rtp.Sequencer) rtp.Packetizer {
	src := rand.NewSource(time.Now().UnixNano())
	return &packetizer{
		MTU:             mtu,
		Payloader:       payloader,
		Sequencer:       sequencer,
		TimestampOffset: uint32(src.Int63()),
		timegen:         time.Now,
	}
}

func (p *packetizer) EnableAbsSendTime(value int) {
	p.extensionNumbers.AbsSendTime = value
}

// Packetize packetizes the payload of an RTP packet and returns one or more RTP packets
func (p *packetizer) Packetize(payload []byte, pts uint32) []*rtp.Packet {
	// Guard against an empty payload
	if len(payload) == 0 {
		return nil
	}

	payloads := p.Payloader.Payload(p.MTU-12, payload)
	packets := make([]*rtp.Packet, len(payloads))

	for i, pp := range payloads {
		packets[i] = &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				Padding:        false,
				Extension:      false,
				Marker:         i == len(payloads)-1,
				SequenceNumber: p.Sequencer.NextSequenceNumber(),
				Timestamp:      pts + p.TimestampOffset,
			},
			Payload: pp,
		}
	}

	if len(packets) != 0 && p.extensionNumbers.AbsSendTime != 0 {
		sendTime := rtp.NewAbsSendTimeExtension(p.timegen())
		// apply http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
		b, err := sendTime.Marshal()
		if err != nil {
			return nil // never happens
		}
		err = packets[len(packets)-1].SetExtension(uint8(p.extensionNumbers.AbsSendTime), b)
		if err != nil {
			return nil // never happens
		}
	}

	return packets
}

// SkipSamples causes a gap in sample count between Packetize requests so the
// RTP payloads produced have a gap in timestamps
func (p *packetizer) SkipSamples(skippedSamples uint32) {
}
