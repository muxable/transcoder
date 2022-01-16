# transcode-mediadevice
transcode-mediadevice is an example application showing a webcam stream transcoded to different formats.

## Instructions
### Open transcode-mediadevice example page
[jsfiddle.net](https://jsfiddle.net/nbpfwry3/4/) you should see your Webcam, two text-areas and a 'Start Session' button

### Run transcode-mediadevice with your browsers SessionDescription as stdin
In the jsfiddle the top textarea is your browser, copy that and:
#### Linux/macOS
Run `echo $BROWSER_SDP | go run main.go`
#### Windows
1. Paste the SessionDescription into a file.
1. Run `go run main.go < my_file`

### Input transcode-mediadevice's SessionDescription into your browser
Copy the text that `transcode-mediadevice` just emitted and copy into second text area

### Hit 'Start Session' in jsfiddle, view the transcoded stream!
The output stream will be transcoded to H264, regardless of what input codec is used.