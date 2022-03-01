package ffmpeg

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pion/sdp/v3"
)

// Receive starts a new receive stream based on an incoming SDP.
func Send(tracks []*TrackLocal) (*sdp.SessionDescription, error) {
	s := &sdp.SessionDescription{
		Version: 0,
		Origin: sdp.Origin{
			Username:       "-",
			NetworkType:    "IN",
			AddressType:    "IP4",
			UnicastAddress: "127.0.0.1",
		},
		SessionName: "Pion WebRTC",
	}

	for _, track := range tracks {
		pt := strconv.FormatUint(uint64(track.PayloadType()), 10)
		rtpmap := fmt.Sprintf("%s/%d", strings.Split(track.Codec().MimeType, "/")[1], track.Codec().ClockRate)
		port, err := GetFreePort()
		if err != nil {
			return nil, err
		}
		if track.Codec().Channels > 0 {
			rtpmap += fmt.Sprintf("/%d", track.Codec().Channels)
		}
		fmtp := track.Codec().RTPCodecCapability.SDPFmtpLine
		s.WithMedia(&sdp.MediaDescription{
			ConnectionInformation: &sdp.ConnectionInformation{
				NetworkType: "IN",
				AddressType: "IP4",
				Address:     &sdp.Address{Address: "127.0.0.1"},
			},
			MediaName: sdp.MediaName{
				Media:   track.Kind().String(),
				Port:    sdp.RangedPort{Value: port, Range: nil},
				Protos:  []string{"RTP", "AVP"},
				Formats: []string{pt},
			},
			Attributes: []sdp.Attribute{
				{
					Key:   "rtpmap",
					Value: pt + " " + rtpmap,
				},
				{
					Key:   "fmtp",
					Value: pt + " " + fmtp,
				},
			},
		})
		dial, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127,0,0,1), Port: port})
		if err != nil {
			return nil, err
		}
		dial.SetWriteBuffer(10 * 1024 * 1024)
		track.Writers = append(track.Writers, dial)
	}
	return s, nil
}