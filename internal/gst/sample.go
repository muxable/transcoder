package gst

type Sample struct {
	Data     []byte
	Duration uint64
	Pts      uint64
	Dts      uint64
	Offset   uint64
}
