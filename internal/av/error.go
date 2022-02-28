package av

/*
#cgo pkg-config: libavutil
#include <libavutil/error.h>
*/
import "C"
import (
	"bytes"
	"fmt"
	"io"
	"unsafe"
)

func AVERROR(code C.int) C.int {
	return -code
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