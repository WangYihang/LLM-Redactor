package redactor

import (
	"regexp"
)

// Rule matches the Gitleaks official TOML structure
type Rule struct {
	ID          string         `toml:"id" json:"id"`
	Description string         `toml:"description" json:"description"`
	Regex       *regexp.Regexp `toml:"-" json:"-"`
	RawRegex    string         `toml:"regex" json:"regex"`
}

type Config struct {
	Rules     []Rule   `toml:"rules" json:"rules"`
	AllowList []string `toml:"allow_list" json:"allow_list"`
}

func (r *Rule) Compile() error {
	re, err := regexp.Compile(r.RawRegex)
	if err != nil {
		return err
	}
	r.Regex = re
	return nil
}
