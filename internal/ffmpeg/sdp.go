package ffmpeg

import (
	"bufio"
	"io"
	"strings"

	"github.com/pion/sdp/v3"
)

func ParseSDP(r io.Reader) (*sdp.SessionDescription, error) {
	scanner := bufio.NewScanner(r)
	lines := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "SDP:" {
			// ignore this extra line that ffmpeg spits out
			continue
		}
		lines = append(lines, line)
		if line == "" {
			break
		}
	}
	s := &sdp.SessionDescription{}
	if err := s.Unmarshal([]byte(strings.Join(lines, "\n"))); err != nil {
		return nil, err
	}
	return s, nil
}