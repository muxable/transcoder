package main

import (
	"fmt"
	"io"
	"os"

	"github.com/muxable/transcoder/internal/ffmpeg"
)

/**
Run with:

ffmpeg -f lavfi -i testsrc -c:v libx264 -f rtp rtp://localhost:4444 | go run main.go | ffplay -protocol_whitelist pipe,udp,rtp -

TODO: the above doesn't work, strangely.

Split into two processes

ffmpeg -f lavfi -i testsrc -c:v libx264 -f rtp rtp://localhost:4444 | go run main.go > video.sdp
ffplay -protocol_whitelist pipe,udp,rtp video.sdp
*/
func main() {
	s, err := ffmpeg.ParseSDP(os.Stdin)
	if err != nil {
		panic(err)
	}
	// receive tracks from ffmpeg
	tracks, err := ffmpeg.Receive(s)
	if err != nil {
		panic(err)
	}
	// for each received track, create a local track
	tls := make([]*ffmpeg.TrackLocal, len(tracks))
	for i, tr := range tracks {
		tl, err := ffmpeg.NewTrackLocal(tr.Codec(), tr.SessionName())
		if err != nil {
			panic(err)
		}
		tls[i] = tl
		go io.Copy(tl, tr)
	}
	// send tracks to ffmpeg
	t, err := ffmpeg.Send(tls)
	if err != nil {
		panic(err)
	}
	buf, err := t.Marshal()
	if err != nil {
		panic(err)
	}
	fmt.Print(string(buf))

	select {}
}
