package main

import (
	"fmt"
	"os"

	"github.com/muxable/transcoder/internal/gst"
	"github.com/tinyzimmer/go-glib/glib"
)

func main() {
    // Initialize GStreamer with the arguments passed to the program. Gstreamer
    // and the bindings will automatically pop off any handled arguments leaving
    // nothing but a pipeline string (unless other invalid args are present).
    gst.Init(&os.Args)

    // Create a main loop. This is only required when utilizing signals via the bindings.
    // In this example, the AddWatch on the pipeline bus requires iterating on the main loop.
    mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)

    pipelineString := "videotestsrc is-live=true ! queue ! decodebin ! queue ! videoconvert ! autovideosink"

    /// Let GStreamer create a pipeline from the parsed launch syntax on the cli.
    pipeline, err := gst.NewPipelineFromString(pipelineString)
    if err != nil {
        fmt.Println(err)
        os.Exit(2)
    }

    // Add a message handler to the pipeline bus, printing interesting information to the console.
    pipeline.GetPipelineBus().AddWatch(func(msg *gst.Message) bool {
        switch msg.Type() {
        case gst.MessageEOS: // When end-of-stream is received flush the pipeling and stop the main loop
            pipeline.BlockSetState(gst.StateNull)
            mainLoop.Quit()
        case gst.MessageError: // Error messages are always fatal
            err := msg.ParseError()
            fmt.Println("ERROR:", err.Error())
            if debug := err.DebugString(); debug != "" {
                fmt.Println("DEBUG:", debug)
            }
            mainLoop.Quit()
        default:
            // All messages implement a Stringer. However, this is
            // typically an expensive thing to do and should be avoided.
            fmt.Println(msg)
        }
        return true
    })

    // Start the pipeline
    pipeline.SetState(gst.StatePlaying)

    // Block and iterate on the main loop
    mainLoop.Run()
}
