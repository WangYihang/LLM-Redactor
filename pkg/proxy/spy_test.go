package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSpy(t *testing.T) {
	rec := httptest.NewRecorder()
	spy := &Spy{
		ResponseWriter: rec,
		Buf:            &bytes.Buffer{},
	}

	spy.WriteHeader(http.StatusTeapot)
	if spy.Code != http.StatusTeapot {
		t.Errorf("Expected status %d, got %d", http.StatusTeapot, spy.Code)
	}

	_, err := spy.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Failed to write to spy: %v", err)
	}
	if spy.Buf.String() != "hello" {
		t.Errorf("Expected body hello, got %s", spy.Buf.String())
	}
	if rec.Body.String() != "hello" {
		t.Errorf("Expected underlying rec body hello, got %s", rec.Body.String())
	}

	// Test flush
	spy.Flush()
}
