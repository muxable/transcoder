# transcode-file
transcode-file is a simple application that shows how to transcode an ivf file on disk.

## Instructions
### Run

```bash
go run main.go -addr localhost:50051 -i input.ivf -o output.ivf
```

You should see an `output.ivf` file produced. The output file does not contain the last ~1s
of video. This is a known issue due to the transcoder expecting live video sources, not
sources that terminate.

If you intend to transcode mostly non-live video sources, it would be easier to use ffmpeg :)