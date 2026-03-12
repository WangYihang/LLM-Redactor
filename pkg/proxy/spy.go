package proxy

import (
	"bytes"
	"net/http"
)

type Spy struct {
	http.ResponseWriter
	Buf  *bytes.Buffer
	Code int
}

func (w *Spy) Write(b []byte) (int, error) {
	w.Buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *Spy) WriteHeader(c int) {
	w.Code = c
	w.ResponseWriter.WriteHeader(c)
}

func (w *Spy) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
