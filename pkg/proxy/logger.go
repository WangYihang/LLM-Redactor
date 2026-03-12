package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

func EnrichLogEvent(e *zerolog.Event, b []byte, h http.Header, sysLog zerolog.Logger) {
	if strings.Contains(h.Get("Content-Encoding"), "gzip") {
		if z, err := gzip.NewReader(bytes.NewReader(b)); err == nil {
			if d, _ := io.ReadAll(z); d != nil {
				b = d
			}
			if err := z.Close(); err != nil {
				sysLog.Debug().Err(err).Msg("failed to close gzip reader")
			}
		}
	}

	arr := zerolog.Arr()
	for k, vv := range h {
		for _, v := range vv {
			arr.Dict(zerolog.Dict().Str("name", k).Str("value", v))
		}
	}
	e.Array("headers", arr)

	if json.Valid(b) {
		e.RawJSON("body", b)
	} else {
		e.Str("body", string(b))
	}
}
