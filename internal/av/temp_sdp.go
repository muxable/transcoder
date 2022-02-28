package av

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/pion/sdp"
	"github.com/pion/webrtc/v3"
)

func kind(codec webrtc.RTPCodecParameters) string {
	if strings.HasPrefix(codec.MimeType, "video") {
		return "video"
	} else if strings.HasPrefix(codec.MimeType, "audio") {
		return "audio"
	}
	return ""
}

func NewTempSDP(codec webrtc.RTPCodecParameters) (*os.File, error) {
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

	pt := strconv.FormatUint(uint64(codec.PayloadType), 10)
	rtpmap := fmt.Sprintf("%s/%d", strings.Split(codec.MimeType, "/")[1], codec.ClockRate)
	if codec.Channels > 0 {
		rtpmap += fmt.Sprintf("/%d", codec.Channels)
	}
	fmtp := codec.SDPFmtpLine
	s.WithMedia(&sdp.MediaDescription{
		ConnectionInformation: &sdp.ConnectionInformation{
			NetworkType: "IN",
			AddressType: "IP4",
			Address:     &sdp.Address{IP: net.IPv4(127, 0, 0, 1)},
		},
		MediaName: sdp.MediaName{
			Media:   kind(codec),
			Port:    sdp.RangedPort{Value: 6000, Range: nil}, // this value doesn't actually matter.
			Protos:  []string{"RTP", "AVP"},
			Formats: []string{pt},
		},
		Attributes: []sdp.Attribute{
			{
				Key:   "rtpmap",
				Value: fmt.Sprintf("%s %s", pt, rtpmap),
			},
			{
				Key:   "fmtp",
				Value: fmt.Sprintf("%s %s", pt, fmtp),
			},
		},
	})
	file, err := ioutil.TempFile("", "pion-webrtc-sdp")
	if err != nil {
		return nil, err
	}
	if _, err := file.WriteString(s.Marshal()); err != nil {
		return nil, err
	}
	return file, nil
}