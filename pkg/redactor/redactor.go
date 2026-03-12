package redactor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
)

const RedactedPlaceholder = "[REDACTED_SECRET]"

type Redactor struct {
	config *Config
	logs   zerolog.Logger
}

func New(configPath string, logs zerolog.Logger) (*Redactor, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	// Try TOML first (Gitleaks official format)
	if err := toml.Unmarshal(data, &config); err != nil {
		// Fallback to JSON
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config (tried TOML and JSON): %w", err)
		}
	}

	var compatibleRules []Rule
	for _, rule := range config.Rules {
		// Go's regexp engine doesn't support lookaround (?!, ?=, ?<)
		if strings.Contains(rule.RawRegex, "?<") || strings.Contains(rule.RawRegex, "?=") || strings.Contains(rule.RawRegex, "?!") {
			continue
		}
		if err := rule.Compile(); err != nil {
			// Skip invalid/unsupported regex
			continue
		}
		compatibleRules = append(compatibleRules, rule)
	}
	config.Rules = compatibleRules

	return &Redactor{config: &config, logs: logs}, nil
}

func mask(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// RedactContent redacts a single string content and logs detections
func (r *Redactor) RedactContent(content string, context map[string]string) string {
	for _, rule := range r.config.Rules {
		// Simple regex replacement
		content = rule.Regex.ReplaceAllStringFunc(content, func(match string) string {
			// Check global allow list
			for _, allow := range r.config.AllowList {
				if match == allow {
					return match
				}
			}

			// LOG DETECTION
			evt := r.logs.Info().
				Str("rule_id", rule.ID).
				Str("description", rule.Description).
				Str("masked_content", mask(match)).
				Int("match_length", len(match))

			for k, v := range context {
				evt.Str(k, v)
			}
			evt.Msg("secret detected")

			return RedactedPlaceholder
		})
	}
	return content
}

// RedactRequest redacts the content of a /v1/chat/completions request body
func (r *Redactor) RedactRequest(body []byte, context map[string]string) ([]byte, error) {
	if !json.Valid(body) {
		return body, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body, err
	}

	messages, ok := data["messages"].([]interface{})
	if !ok {
		return body, nil
	}

	for _, msg := range messages {
		m, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}
		if content, ok := m["content"].(string); ok {
			m["content"] = r.RedactContent(content, context)
		}
	}

	return json.Marshal(data)
}

// StreamRedactor implements a sliding window redactor for SSE streams
type StreamRedactor struct {
	r       *Redactor
	buffer  []byte
	maxLen  int
	context map[string]string
}

func NewStreamRedactor(r *Redactor, windowSize int, context map[string]string) *StreamRedactor {
	if windowSize <= 0 {
		windowSize = 100
	}
	return &StreamRedactor{
		r:       r,
		maxLen:  windowSize,
		context: context,
	}
}

func extractContent(data map[string]interface{}) (string, map[string]interface{}, bool) {
	choices, ok := data["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", nil, false
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", nil, false
	}

	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		delta, ok = choice["message"].(map[string]interface{})
	}
	if !ok {
		return "", nil, false
	}

	content, ok := delta["content"].(string)
	return content, delta, ok
}

// RedactSSELine processes a single "data: ..." line
func (sr *StreamRedactor) RedactSSELine(line []byte) []byte {
	if !bytes.HasPrefix(line, []byte("data: ")) {
		return line
	}

	rawData := bytes.TrimPrefix(line, []byte("data: "))
	if string(rawData) == "[DONE]" {
		return line
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rawData, &data); err != nil {
		return line
	}

	content, delta, ok := extractContent(data)
	if !ok {
		return line
	}

	sr.buffer = append(sr.buffer, []byte(content)...)

	if len(sr.buffer) < sr.maxLen {
		redacted := sr.r.RedactContent(string(sr.buffer), sr.context)
		sr.buffer = []byte(redacted)
		delta["content"] = ""
	} else {
		toFlush := len(sr.buffer) - sr.maxLen
		flushContent := string(sr.buffer[:toFlush])
		sr.buffer = sr.buffer[toFlush:]

		redactedTotal := sr.r.RedactContent(flushContent+string(sr.buffer), sr.context)
		delta["content"] = redactedTotal[:len(redactedTotal)-len(sr.buffer)]
		sr.buffer = []byte(redactedTotal[len(redactedTotal)-len(sr.buffer):])
	}

	newRawData, _ := json.Marshal(data)
	return append([]byte("data: "), newRawData...)
}

func (sr *StreamRedactor) Flush() []byte {
	if len(sr.buffer) == 0 {
		return nil
	}
	redacted := sr.r.RedactContent(string(sr.buffer), sr.context)
	sr.buffer = nil
	return []byte(redacted)
}
