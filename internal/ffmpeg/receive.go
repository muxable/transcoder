package ffmpeg

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v3"
)

// Receive starts a new receive stream based on an incoming SDP.
func Receive(desc *sdp.SessionDescription) ([]*TrackRemote, error) {
	tracks := make([]*TrackRemote, len(desc.MediaDescriptions))
	for i, media := range desc.MediaDescriptions {
		if media.MediaName.Media == "application" {
			continue
		}

		kind := webrtc.NewRTPCodecType(media.MediaName.Media)

		// validate some assumptions we're making.
		if media.MediaName.Port.Range != nil {
			return nil, errors.New("ranged ports not supported")
		}

		pt, err := strconv.ParseUint(media.MediaName.Formats[0], 10, 8)
		if err != nil {
			return nil, err
		}

		// listen on the connection information.
		readCh := make(chan []byte, 100)
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: media.MediaName.Port.Value})
		if err != nil {
			return nil, err
		}
		conn.SetReadBuffer(50 * 1024 * 1024)
		go func() {
			for {
				buf := make([]byte, 1500)
				n, _, err := conn.ReadFromUDP(buf)

				// then create a track for each media description.
				if err != nil {
					return
				}

				readCh <- buf[:n]
			}
		}()
		mimeType := ""
		clockRate := uint64(90000)
		fmtp := ""
		channels := uint64(1)
		for _, attribute := range media.Attributes {
			prefix := fmt.Sprintf("%d ", pt)
			if attribute.Key == "rtpmap" && strings.HasPrefix(attribute.Value, prefix) {
				parts := strings.Split(attribute.Value[len(prefix):], "/")
				mimeType = fmt.Sprintf("%s/%s", media.MediaName.Media, parts[0])
				clockRate, err = strconv.ParseUint(parts[1], 10, 32)
				if err != nil {
					return nil, err
				}
				if len(parts) > 2 {
					channels, err = strconv.ParseUint(parts[1], 10, 16)
					if err != nil {
						return nil, err
					}
				}
			}
			if attribute.Key == "fmtp" && strings.HasPrefix(attribute.Value, prefix) {
				fmtp = attribute.Value[len(prefix):]
			}
		}
		tracks[i] = &TrackRemote{
			readCh: readCh,
			payloadType: webrtc.PayloadType(pt),
			kind:        kind,
			codec: webrtc.RTPCodecParameters{
				PayloadType: webrtc.PayloadType(pt),
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:    mimeType,
					ClockRate:   uint32(clockRate),
					Channels:    uint16(channels),
					SDPFmtpLine: fmtp,
				},
			},
		}
	}
	return tracks, nil
}