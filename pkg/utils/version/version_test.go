package version

import (
	"strings"
	"testing"
)

func TestVersionInfo(t *testing.T) {
	info := GetVersionInfo()
	if info.Version != "0.0.0" {
		t.Errorf("Expected 0.0.0, got %s", info.Version)
	}

	jsonStr := info.JSON()
	if !strings.Contains(jsonStr, `"version":"0.0.0"`) {
		t.Errorf("Expected JSON to contain version, got %s", jsonStr)
	}
}
