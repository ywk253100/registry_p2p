package utils

import (
	"io"
	"net/http"
)

type FlushWriter struct {
	W io.Writer
}

func NewFlushWriter(w io.Writer) *FlushWriter {
	f := &FlushWriter{
		W: w,
	}
	return f
}

func (f *FlushWriter) Write(str string) (n int, err error) {
	n, err = f.W.Write([]byte(str))
	if err != nil {
		return
	}

	if flush, ok := f.W.(http.Flusher); ok {
		flush.Flush()
	}
	return
}
