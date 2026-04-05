package detectors

import (
	"context"
	"regexp"
	"strings"
)

type RegexRule struct {
	ID            string
	Description   string
	Regex         *regexp.Regexp
	ReplaceEngine string
}

type RegexDetector struct {
	rules          []RegexRule
	pseudonymizers map[string]*ReplaceEnginePseudonymizer
}

func NewRegexDetector(rules []RegexRule) *RegexDetector {
	pseudonymizers := make(map[string]*ReplaceEnginePseudonymizer)
	for _, rule := range rules {
		if rule.ReplaceEngine != "" {
			key := rule.ID + ":" + rule.ReplaceEngine
			if _, exists := pseudonymizers[key]; !exists {
				pseudonymizers[key] = NewReplaceEnginePseudonymizer(rule.ReplaceEngine)
			}
		}
	}
	return &RegexDetector{rules: rules, pseudonymizers: pseudonymizers}
}

func (d *RegexDetector) Type() string {
	return "regex"
}

func (d *RegexDetector) Redact(ctx context.Context, content string, callback RedactionCallback) string {
	for _, rule := range d.rules {
		rule := rule // capture for closure
		if rule.ReplaceEngine != "" {
			key := rule.ID + ":" + rule.ReplaceEngine
			ps := d.pseudonymizers[key]
			content = rule.Regex.ReplaceAllStringFunc(content, func(match string) string {
				if len(match) == 0 {
					return match
				}
				fake := ps.GetOrCreate(match)
				callback(match, rule.ID, rule.Description)
				return fake
			})
		} else {
			content = rule.Regex.ReplaceAllStringFunc(content, func(match string) string {
				if len(match) == 0 {
					return match
				}
				return callback(match, rule.ID, rule.Description)
			})
		}
	}
	return content
}

// Unredact restores pseudonymized values produced by replace_engine rules.
func (d *RegexDetector) Unredact(content string) string {
	for _, rule := range d.rules {
		if rule.ReplaceEngine == "" {
			continue
		}
		key := rule.ID + ":" + rule.ReplaceEngine
		ps := d.pseudonymizers[key]
		ps.mu.RLock()
		for fake, real := range ps.fakeToReal {
			content = strings.ReplaceAll(content, fake, real)
		}
		ps.mu.RUnlock()
	}
	return content
}
