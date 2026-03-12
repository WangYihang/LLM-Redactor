package proxy

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestEnrichLogEvent(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	evt := log.Info()

	sysLog := zerolog.Nop()

	h := http.Header{}
	h.Add("Content-Type", "application/json")
	h.Add("Authorization", "Bearer secret-token")
	h.Add("Cookie", "session=abc123")

	// Test regular JSON
	body := []byte(`{"hello": "world"}`)
	EnrichLogEvent(evt, body, h, sysLog)
	evt.Msg("test")

	out := buf.String()
	if !strings.Contains(out, `"body":{"hello":"world"}`) {
		t.Errorf("Expected body to be raw JSON, got %s", out)
	}
	if !strings.Contains(out, `"name":"Content-Type"`) || !strings.Contains(out, `"value":"application/json"`) {
		t.Errorf("Expected headers to be logged, got %s", out)
	}
	if !strings.Contains(out, `"name":"Authorization"`) || !strings.Contains(out, `"value":"[REDACTED]"`) {
		t.Errorf("Expected authorization header to be redacted, got %s", out)
	}
	if strings.Contains(out, "secret-token") {
		t.Errorf("Expected authorization value to be redacted, got %s", out)
	}
	if !strings.Contains(out, `"name":"Cookie"`) || !strings.Contains(out, `"value":"[REDACTED]"`) {
		t.Errorf("Expected cookie header to be redacted, got %s", out)
	}

	// Test GZIP
	buf.Reset()
	evt2 := log.Info()
	var gzBuf bytes.Buffer
	w := gzip.NewWriter(&gzBuf)
	_, _ = w.Write([]byte(`{"gzipped": true}`))
	_ = w.Close()

	h2 := http.Header{}
	h2.Add("Content-Encoding", "gzip")
	EnrichLogEvent(evt2, gzBuf.Bytes(), h2, sysLog)
	evt2.Msg("test gz")

	out2 := buf.String()
	if !strings.Contains(out2, `"body":{"gzipped":true}`) {
		t.Errorf("Expected decompressed json body, got %s", out2)
	}
}
