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
			arr.Dict(zerolog.Dict().Str("name", k).Str("value", redactHeaderValue(k, v)))
		}
	}
	e.Array("headers", arr)

	if json.Valid(b) {
		var compact bytes.Buffer
		if err := json.Compact(&compact, b); err == nil {
			e.RawJSON("body", compact.Bytes())
		} else {
			e.Str("body", string(b))
		}
	} else {
		e.Str("body", string(b))
	}
}

const redactedHeaderValue = "[REDACTED]"

func redactHeaderValue(name, value string) string {
	if isSensitiveHeader(name) {
		return redactedHeaderValue
	}
	return value
}

func isSensitiveHeader(name string) bool {
	switch strings.ToLower(name) {
	case "authorization",
		"proxy-authorization",
		"x-api-key",
		"x-goog-api-key",
		"x-auth-token",
		"x-access-token",
		"x-amz-security-token",
		"api-key",
		"apikey",
		"token",
		"cookie",
		"set-cookie":
		return true
	default:
		return false
	}
}
