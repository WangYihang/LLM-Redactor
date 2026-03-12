package providers

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetProvider(t *testing.T) {
	baseURL, _ := url.Parse("https://api.example.com")
	apiKey := "test-key"

	// Test default/base provider
	p := GetProvider("unknown", baseURL, apiKey)
	req := httptest.NewRequest("GET", "http://localhost", nil)
	p.Director(req)
	if req.Host != "api.example.com" {
		t.Errorf("Expected host api.example.com, got %s", req.Host)
	}
	if req.Header.Get("Authorization") != "Bearer test-key" {
		t.Errorf("Expected Bearer test-key, got %s", req.Header.Get("Authorization"))
	}

	// Test deepseek
	pDS := GetProvider("deepseek", baseURL, apiKey)
	if _, ok := pDS.(*DeepseekProvider); !ok {
		t.Errorf("Expected DeepseekProvider")
	}

	// Test kimi
	pKimi := GetProvider("kimi", baseURL, apiKey)
	if _, ok := pKimi.(*KimiProvider); !ok {
		t.Errorf("Expected KimiProvider")
	}
}
