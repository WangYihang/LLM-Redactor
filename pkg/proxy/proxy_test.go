package proxy

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/wangyihang/llm-prism/pkg/config"
	"github.com/wangyihang/llm-prism/pkg/utils/logging"
)

func TestRequestID(t *testing.T) {
	ctx := context.Background()
	id := GetRequestID(ctx)
	if id != "" {
		t.Errorf("Expected empty request ID, got %s", id)
	}

	ctx = WithRequestID(ctx, "test-id")
	id = GetRequestID(ctx)
	if id != "test-id" {
		t.Errorf("Expected test-id, got %s", id)
	}
}

func TestSetupInvalidURL(t *testing.T) {
	cli := &config.CLI{}
	cli.Run.ApiURL = ":invalid-url"

	// Using empty logger for testing, just passing nil to avoid file creation if we can,
	// but logging.New creates files. Let's just create a dummy Loggers struct instead of calling logging.New()
	logs := &logging.Loggers{}
	rp, err := Setup(cli, nil, logs)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	if rp != nil {
		t.Error("Expected nil reverse proxy")
	}
}

func TestSetupValidURL(t *testing.T) {
	cli := &config.CLI{}
	cli.Run.ApiURL = "http://localhost:8080"
	cli.Run.Provider = "base"

	logs := &logging.Loggers{}
	rp, err := Setup(cli, nil, logs)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if rp == nil {
		t.Fatal("Expected reverse proxy, got nil")
	}

	req := httptest.NewRequest("GET", "http://localhost:8080/path", nil)
	rp.Director(req)
	if req.URL.Scheme != "http" || req.URL.Host != "localhost:8080" {
		t.Errorf("Director failed to set URL: %v", req.URL)
	}
}
