package scanner

import (
	"bytes"
	"regexp"
	"time"
)

type SecretPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Description string
	Severity    string
}

type Secret struct {
	Type       string
	Match      string
	File       string
	Line       int
	Column     int
	DetectedAt time.Time
}

var DefaultPatterns = []SecretPattern{
	{
		Name:        "AWS Access Key",
		Pattern:     regexp.MustCompile(`(?i)(AKIA|A3T|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`),
		Description: "AWS Access Key ID",
		Severity:    "HIGH",
	},
	{
		Name:        "GitHub Token",
		Pattern:     regexp.MustCompile(`(?i)(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,}`),
		Description: "GitHub Personal Access Token",
		Severity:    "HIGH",
	},
	{
		Name:        "Private Key",
		Pattern:     regexp.MustCompile(`-----BEGIN (RSA|DSA|EC|OPENSSH) PRIVATE KEY-----`),
		Description: "Private Key",
		Severity:    "CRITICAL",
	},
	{
		Name:        "JWT",
		Pattern:     regexp.MustCompile(`eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`),
		Description: "JSON Web Token",
		Severity:    "MEDIUM",
	},
	{
		Name:        "Password",
		Pattern:     regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']?[^"'\s]+["']?`),
		Description: "Hardcoded Password",
		Severity:    "HIGH",
	},
	{
		Name:        "API Key",
		Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*["']?[^"'\s]+["']?`),
		Description: "API Key",
		Severity:    "MEDIUM",
	},
}

type SecretScanner struct {
	patterns []SecretPattern
}

func NewSecretScanner() *SecretScanner {
	scanner := &SecretScanner{
		patterns: DefaultPatterns,
	}
	return scanner
}

func (s *SecretScanner) Scan(content []byte, filename string) []Secret {
	var secrets []Secret

	for lineNum, line := range bytes.Split(content, []byte("\n")) {
		for _, pattern := range s.patterns {
			matches := pattern.Pattern.FindAllString(string(line), -1)
			for _, match := range matches {
				secrets = append(secrets, Secret{
					Type:       pattern.Name,
					Match:      match,
					File:       filename,
					Line:       lineNum + 1,
					DetectedAt: time.Now(),
				})
			}
		}
	}

	return secrets
}

func (s *SecretScanner) AddPattern(pattern SecretPattern) {
	s.patterns = append(s.patterns, pattern)
}

type IPAllowlist struct {
	CIDRBlocks []string
}

func NewIPAllowlist() *IPAllowlist {
	return &IPAllowlist{
		CIDRBlocks: make([]string, 0),
	}
}

func (i *IPAllowlist) Add(cidr string) {
	i.CIDRBlocks = append(i.CIDRBlocks, cidr)
}

func (i *IPAllowlist) Contains(ip string) bool {
	return false
}