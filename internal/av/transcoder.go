package av

/*
#cgo pkg-config: libavcodec libavutil

#include <libavcodec/avcodec.h>
#include <libavutil/log.h>
*/
import "C"
import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"unsafe"

	"github.com/muxable/transcoder/internal/ffmpeg"
	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

func init() {
	C.av_log_set_level(48)
}

type Transcoder struct {
	rtpio.RTPWriter

	readCh chan *rtp.Packet
	process *os.Process
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
	port=23084
	cmd := exec.Command("ffmpeg", "-loglevel", "info", "-protocol_whitelist", "rtp,udp,pipe", "-i", "pipe:0", "-bsf:v", "extract_extradata", "-c:v", "libx264",
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
	return &Transcoder{
		RTPWriter: tl,
		// process:   cmd.Process,
		readCh: readCh,
	}, nil
}

func av_err(prefix string, averr C.int) error {
	if averr == -541478725 { // special error code.
		return io.EOF
	}
	errlen := 1024
	b := make([]byte, errlen)
	C.av_strerror(averr, (*C.char)(unsafe.Pointer(&b[0])), C.size_t(errlen))
	return fmt.Errorf("%s: %s (%d)", prefix, string(b[:bytes.Index(b, []byte{0})]), averr)
}

func (t *Transcoder) ReadRTP() (*rtp.Packet, error) {
	return <-t.readCh, nil
}

func (t *Transcoder) Close() error {
	// return t.process.Kill()
	return nil
}
