# muxable/transcoder

This package implements a transcoding server. It accepts WebRTC signalled over gRPC and will respond with a transcoded track for each received track.

**Are you interested in livestreaming video with WebRTC?** Join us at [Muxable](mailto:kevin@muxable.com)!

## Client SDK

For convenience if you don't want to deal with signalling and reconnection semantics, a lightweight client SDK is provided.

### Example usage

```go
import "github.com/muxable/transcoder/pkg/transcoder"

...

conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
    return err
}

client, err := transcoder.NewClient(context.Background(), conn)
if err != nil {
    return err
}

transcoded, err := client.Transcode(track)
if err != nil {
    return err
}
```

Here, `client.Transcode()` accepts a `TrackLocal` and returns a `TrackRemote`. The mime type of the output track can also be specified, for example:

```go
transcoded, err := client.Transcode(track, transcode.ToMimeType(webrtc.MimeTypeVP9))
```

## Cloud hosting

If you are interested in using this service pre-deployed, please email kevin@muxable.com. We are exploring offering endpoints as a paid service.

## Debugging

The transcoding server uses GStreamer under the hood. To view debug messages from GStreamer, set `GST_DEBUG=3`.