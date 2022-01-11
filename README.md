# trancoder

This package implements a transcoding server. It accepts WebRTC signalled over gRPC and will respond with a transcoded track for each received track.

## Client SDK

For convenience if you don't want to deal with signalling, a lightweight client SDK is provided.

### Example usage

```go
import (
	transcoder "github.com/muxable/transcoder/pkg"
)

...

conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
    return err
}

client, err := transcoder.NewTranscoderAPIClient(conn)
if err != nil {
    return err
}

transcoded, err := client.Transcode(track)
if err != nil {
    return err
}
```

Here, `client.Transcode()` accepts a `TrackLocal` and returns a `TrackRemote`.