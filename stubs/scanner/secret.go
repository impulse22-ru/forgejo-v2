package scanner

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
)

var (
	ErrSecretNotFound = fmt.Errorf("secret not found")
	ErrInvalidPattern = fmt.Errorf("invalid pattern")
)

type Pattern struct {
	Name        string            `json:"name"`
	Regex       string            `json:"regex"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Entropy    float32           `json:"entropy"`
	Hotfile    bool             `json:"hotfile"`
}

func (p *Pattern) Compile() (*regexp.Regexp, error) {
	if p.Regex == "" {
		return nil, ErrInvalidPattern
	}
	return regexp.Compile(p.Regex)
}

type Secret struct {
	Type        string  `json:"type"`
	Match       string  `json:"match"`
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Entropy    float32 `json:"entropy"`
	SHA        string  `json:"sha"`
	Severity    string  `json:"severity"`
	DetectedAt time.Time `json:"detected_at"`
}

func (s *Secret) String() string {
	return fmt.Sprintf("Secret{type=%s, file=%s, line=%d}", s.Type, s.File, s.Line)
}

func (s *Secret) Masked() string {
	if len(s.Match) <= 8 {
		return strings.Repeat("*", len(s.Match))
	}
	return s.Match[:4] + strings.Repeat("*", len(s.Match)-8) + s.Match[len(s.Match)-4:]
}

type ScanResult struct {
	RepoID      int64     `json:"repo_id"`
	CommitSHA  string    `json:"commit_sha"`
	Branch     string    `json:"branch"`
	Secrets    []Secret  `json:"secrets"`
	FilesScanned int     `json:"files_scanned"`
	AlertsCreated int    `json:"alerts_created"`
	ScannedAt  time.Time `json:"scanned_at"`
}

func (r *ScanResult) HasSecrets() bool {
	return len(r.Secrets) > 0
}

func (r *ScanResult) CriticalCount() int {
	count := 0
	for _, s := range r.Secrets {
		if s.Severity == "CRITICAL" {
			count++
		}
	}
	return count
}

type DefaultPatterns struct {
	patterns []Pattern
}

func NewDefaultPatterns() *DefaultPatterns {
	return &DefaultPatterns{
		patterns: []Pattern{
			{
				Name: "AWS Access Key",
				Regex: "(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}",
				Description: "AWS Access Key ID",
				Severity: "CRITICAL",
			},
			{
				Name: "AWS Secret Key",
				Regex: "(?i)aws(.{0,20})?(?-i)['\"][0-9a-zA-Z/+]{40}['\"]",
				Description: "AWS Secret Access Key",
				Severity: "CRITICAL",
			},
			{
				Name: "GitHub Token",
				Regex: "(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,}",
				Description: "GitHub Personal Access Token",
				Severity: "CRITICAL",
			},
			{
				Name: "GitLab Token",
				Regex: "glpat-[0-9a-zA-Z\\-]{20,}",
				Description: "GitLab Personal Access Token",
				Severity: "CRITICAL",
			},
			{
				Name: "Private Key",
				Regex: "-----BEGIN (RSA|EC|DSA|OPENSSH|PGP) PRIVATE KEY-----",
				Description: "Private Key",
				Severity: "CRITICAL",
			},
			{
				Name: "JWT Token",
				Regex: "eyJ[A-Za-z0-9_-]*\\.eyJ[A-Za-z0-9_-]*\\.[A-Za-z0-9_-]*",
				Description: "JSON Web Token",
				Severity: "HIGH",
			},
			{
				Name: "Slack Token",
				Regex: "xox[baprs]-([0-9a-zA-Z]{10,48})?",
				Description: "Slack Token",
				Severity: "HIGH",
			},
			{
				Name: "Password in Code",
				Regex: "(?i)(password|passwd|pwd|secret|token|api_key|apikey)\\s*[:=]\\s*['\"]?[\\w-]{8,}",
				Description: "Hardcoded Password",
				Severity: "MEDIUM",
			},
			{
				Name: "API Key",
				Regex: "(?i)(api[_-]?key|apikey)\\s*[:=]\\s*['\"]?[\\w-]{16,}",
				Description: "API Key",
				Severity: "MEDIUM",
			},
			{
				Name: "Database URL",
				Regex: "(?i)(mysql|postgres|sqlite|mongodb)://[^\\s]+",
				Description: "Database Connection String",
				Severity: "HIGH",
			},
		},
	}
}

func (p *DefaultPatterns) All() []Pattern {
	return p.patterns
}

func (p *DefaultPatterns) Get(name string) *Pattern {
	for _, pat := range p.patterns {
		if pat.Name == name {
			return &pat
		}
	}
	return nil
}

type Scanner struct {
	patterns *DefaultPatterns
	store    *SecretStore
}

func NewScanner() *Scanner {
	return &Scanner{
		patterns: NewDefaultPatterns(),
		store:    NewSecretStore(),
	}
}

func (s *Scanner) Scan(content []byte, filename string) []Secret {
	var secrets []Secret

	lines := bytes.Split(content, []byte("\n"))
	for lineNum, line := range lines {
		lineStr := string(line)

		for _, pattern := range s.patterns.All() {
			re, err := pattern.Compile()
			if err != nil {
				continue
			}

			matches := re.FindAllStringIndex(lineStr, -1)
			for _, match := range matches {
				secrets = append(secrets, Secret{
					Type:      pattern.Name,
					Match:    lineStr[match[0]:match[1]],
					File:     filename,
					Line:    lineNum + 1,
					Severity: pattern.Severity,
					SHA:     generateSHA256(lineStr),
				})
			}
		}
	}

	return secrets
}

func (s *Scanner) ScanRepository(ctx context.Context, repoID int64) (*ScanResult, error) {
	result := &ScanResult{
		RepoID:     repoID,
		ScannedAt: time.Now(),
	}

	files := []string{".env", "config.yaml", "secrets.json"}
	for _, file := range files {
		content := []byte("fake content for " + file)
		secrets := s.Scan(content, file)
		result.Secrets = append(result.Secrets, secrets...)
		result.FilesScanned++
	}

	if len(result.Secrets) > 0 {
		result.AlertsCreated = len(result.Secrets)
	}

	return result, nil
}

func (s *Scanner) EnablePushProtection(repoID int64, enabled bool) error {
	return nil
}

func (s *Scanner) HandleWebhook(w http.ResponseWriter, req *http.Request) {
	secrets := []Secret{
		{Type: "Test Secret", File: "test.txt", Line: 1, Severity: "HIGH"},
	}

	data, _ := json.Marshal(secrets)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

type SecretStore struct {
	secrets map[int64][]Secret
}

func NewSecretStore() *SecretStore {
	return &SecretStore{
		secrets: make(map[int64][]Secret),
	}
}

func (s *SecretStore) Create(repoID int64, secret *Secret) error {
	s.secrets[repoID] = append(s.secrets[repoID], *secret)
	return nil
}

func (s *SecretStore) GetByRepo(repoID int64) ([]Secret, bool) {
	secrets, ok := s.secrets[repoID]
	return secrets, ok
}

func (s *SecretStore) Delete(repoID int64) error {
	delete(s.secrets, repoID)
	return nil
}

func generateSHA256(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func CalculateEntropy(s string) float32 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]float32)
	for _, c := range s {
		freq[c]++
	}

	var entropy float32
	length := float32(len(s))
	for _, count := range freq {
		p := count / length
		entropy -= p * float32(float64(p)/float64(float32(0))) * float32(float64(p))/float64(float32(0)))
	}
}

type Config struct {
	Enabled       bool
	Patterns     []string `json:"patterns"`
	ScanOnPush   bool    `json:"scan_on_push"`
	ScanOnPR    bool    `json:"scan_on_pr"`
	AlertOwners bool    `json:"alert_owners"`
}

func (c *Config) Validate() error {
	return nil
}

func init() {}