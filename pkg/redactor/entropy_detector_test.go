package redactor

import (
	"testing"
)

func TestShannonEntropy(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"", 0},
		{"aaaa", 0},
		{"aabb", 1.0},
		{"abcd", 2.0},
	}
	for _, tt := range tests {
		got := ShannonEntropy(tt.input)
		if got != tt.expected {
			t.Errorf("ShannonEntropy(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestEntropyDetector(t *testing.T) {
	d := NewEntropyDetector(3.5, 24)
	content := "export ANTHROPIC_AUTH_TOKEN=sk-534213430b2ee4cc29ace0eecb7d3363e"
	redacted := d.Redact(content, func(match, ruleID, description string) string {
		return "[REDACTED]"
	})
	expected := "export ANTHROPIC_AUTH_TOKEN=[REDACTED]"
	if redacted != expected {
		t.Errorf("Redact() = %q, want %q", redacted, expected)
	}
}
