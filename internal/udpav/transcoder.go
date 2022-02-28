package udpav

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/muxable/transcoder/internal/ffmpeg"
	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

type Transcoder struct {
	rtpio.RTPWriter

	readCh chan *rtp.Packet
	process *os.Process

	dial *net.UDPConn
}

func NewTranscoder(from webrtc.RTPCodecParameters, to webrtc.RTPCodecCapability) (*Transcoder, error) {
	tl, err := ffmpeg.NewTrackLocal(from, "session")
	if err != nil {
		return nil, err
	}
	port, err := ffmpeg.GetFreePort()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("ffmpeg", "-loglevel", "info", "-protocol_whitelist", "rtp,udp,pipe", "-i", "pipe:0",
	"-bsf:v", "extract_extradata", "-c:v", "libx264",
	"-tune", "zerolatency", "-preset", "ultrafast",
	"-profile:v", "baseline", "-r", "24", "-g", "60",
	"-max_delay", "0", "-bf", "0", "-bsf:v", "dump_extra=freq=keyframe",
	"-f", "rtp", "-pkt_size", "1200", fmt.Sprintf("rtp://localhost:%d", port))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	s, err := ffmpeg.Send([]*ffmpeg.TrackLocal{tl})
	if err != nil {
		return nil, err
	}
	b, err := s.Marshal()
	if err != nil {
		return nil, err
	}
	log.Printf("SDP: %s", string(b))
	if _, err := stdin.Write(b); err != nil {
		return nil, err
	}
	if err := stdin.Close(); err != nil {
		return nil, err
	}
	// ffmpeg won't print an sdp until it receives some data, so we have to do this lazily.
	readCh := make(chan *rtp.Packet)
	go func() {
		t, err := ffmpeg.ParseSDP(stdout)
		if err != nil {
			return
		}
		tr, err := ffmpeg.Receive(t)
		if err != nil {
			return
		}
		for {
			p, err := tr[0].ReadRTP()
			if err != nil {
				break
			}
			readCh <- p
		}
	}()
	dial, _ := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5030})
	return &Transcoder{
		RTPWriter: tl,
		// process:   cmd.Process,
		readCh: readCh,
		dial: dial,
	}, nil
}

func (t *Transcoder) ReadRTP() (*rtp.Packet, error) {
	p := <-t.readCh
	buf, err := p.Marshal()
	if err != nil {
		return nil, err
	}
	t.dial.Write(buf)
	return p, nil
}

func (t *Transcoder) Close() error {
	// return t.process.Kill()
	return nil
}