package redactor

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestRuleFiltering(t *testing.T) {
	config := `
[[rules]]
id = "go-compatible"
description = "Should be kept"
regex = "sk-[a-zA-Z0-9]{32}"

[[rules]]
id = "incompatible-lookaround"
description = "Should be skipped"
regex = "(?<=secret:)[a-z]+"
`
	tmpFile := "test_rules.toml"
	if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	r, err := New(tmpFile, zerolog.Nop())
	if err != nil {
		t.Fatalf("Failed to create redactor: %v", err)
	}

	if len(r.config.Rules) != 1 {
		t.Errorf("Expected 1 compatible rule, got %d", len(r.config.Rules))
	}
}

func TestRedactRequest(t *testing.T) {
	r := &Redactor{
		config: &Config{
			Rules: []Rule{
				{ID: "test-secret", RawRegex: "SECRET_KEY_[0-9]{5}"},
			},
		},
		logs: zerolog.Nop(),
	}
	if err := r.config.Rules[0].Compile(); err != nil {
		t.Fatalf("Failed to compile rule: %v", err)
	}

	reqBody := `{"messages": [{"role": "user", "content": "The key is SECRET_KEY_12345"}]}`
	redacted, _ := r.RedactRequest([]byte(reqBody), nil)

	if strings.Contains(string(redacted), "SECRET_KEY_12345") {
		t.Error("Secret not redacted in request")
	}
}

func TestStreamRedactorSlidingWindow(t *testing.T) {
	r := &Redactor{
		config: &Config{
			Rules: []Rule{
				{ID: "split-secret", RawRegex: "DANGER_ZONE"},
			},
		},
		logs: zerolog.Nop(),
	}
	if err := r.config.Rules[0].Compile(); err != nil {
		t.Fatalf("Failed to compile rule: %v", err)
	}

	// 使用较大窗口以容纳完整占位符
	sr := NewStreamRedactor(r, 30, nil)

	// 模拟敏感词被切分： "DAN" + "GER_ZONE"
	line1 := `data: {"choices":[{"delta":{"content":"DAN"}}]} `
	line2 := `data: {"choices":[{"delta":{"content":"GER_ZONE suffix"}}]} `

	res1 := sr.RedactSSELine([]byte(line1))
	res2 := sr.RedactSSELine([]byte(line2))
	res3 := sr.Flush()

	fullResult := string(res1) + string(res2) + string(res3)

	if strings.Contains(fullResult, "DANGER_ZONE") {
		t.Errorf("Secret leaked: %s", fullResult)
	}
	if !strings.Contains(fullResult, RedactedPlaceholder) {
		t.Errorf("Placeholder missing: %s", fullResult)
	}
}

func TestDetectionLogging(t *testing.T) {
	var buf bytes.Buffer
	r := &Redactor{
		config: &Config{
			Rules: []Rule{
				{ID: "log-rule", Description: "Test Desc", RawRegex: "HIT_ME"},
			},
		},
		logs: zerolog.New(&buf),
	}
	if err := r.config.Rules[0].Compile(); err != nil {
		t.Fatalf("Failed to compile rule: %v", err)
	}

	r.RedactContent("Text HIT_ME text", map[string]string{"ctx_key": "ctx_val"})

	output := buf.String()
	if !strings.Contains(output, "log-rule") || !strings.Contains(output, "ctx_val") {
		t.Errorf("Audit log incomplete: %s", output)
	}
}
