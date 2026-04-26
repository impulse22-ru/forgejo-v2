package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

type Advisory struct {
	ID          string   `json:"id"`
	GHSAID      string   `json:"ghsa_id,omitempty"`
	CVEID       string   `json:"cve_id,omitempty"`
	Package     string   `json:"package"`
	Identifier string   `json:"identifier"`
	Severity    string   `json:"severity"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	FixedVersion string  `json:"fixed_version"`
	Published  time.Time `json:"published"`
	Updated    time.Time `json:"updated"`
}

func (a *Advisory) String() string {
	if a.CVEID != "" {
		return a.CVEID
	}
	if a.GHSAID != "" {
		return a.GHSAID
	}
	return a.ID
}

func (a *Advisory) IsCritical() bool {
	return a.Severity == "CRITICAL"
}

func (a *Advisory) IsHigh() bool {
	return a.Severity == "HIGH"
}

type Vulnerability struct {
	Advisory   *Advisory
	Package    string
	Version    string
	Scope     string
}

func (v *Vulnerability) String() string {
	return fmt.Sprintf("%s@%s (%s)", v.Package, v.Version, v.Advisory)
}

type ScanResult struct {
	Repository  string          `json:"repository"`
	ScannedAt  time.Time       `json:"scanned_at"`
	Findings  []Vulnerability `json:"findings"`
	Summary   ScanSummary   `json:"summary"`
}

type ScanSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Total    int `json:"total"`
}

func (r *ScanResult) HasVulnerabilities() bool {
	return len(r.Findings) > 0
}

func (r *ScanResult) HighOrAbove() bool {
	return r.Summary.Critical > 0 || r.Summary.High > 0
}

type Scanner struct {
	advisories *AdvisoryDB
	client    *http.Client
}

func NewScanner() *Scanner {
	return &Scanner{
		advisories: NewAdvisoryDB(),
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Scanner) ScanRepository(ctx context.Context, repo string, lockfiles []string) (*ScanResult, error) {
	result := &ScanResult{
		Repository: repo,
		ScannedAt: time.Now(),
	}

	for _, lockfile := range lockfiles {
		packages, err := s.parseLockfile(lockfile)
		if err != nil {
			continue
		}

		for _, pkg := range packages {
			vulns, err := s.checkVulnerabilities(pkg.Name, pkg.Version)
			if err != nil {
				continue
			}
			result.Findings = append(result.Findings, vulns...)
		}
	}

	result.Summary = summarizeResults(result.Findings)
	return result, nil
}

func (s *Scanner) parseLockfile(path string) ([]Package, error) {
	data, err := io.ReadAll(strings.NewReader(path))
	if err != nil {
		return nil, err
	}

	var packages []Package
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, err
	}

	return packages, nil
}

func (s *Scanner) checkVulnerabilities(name, version string) ([]Vulnerability, error) {
	advisories := s.advisories.Get(name, version)
	if len(advisories) == 0 {
		return nil, nil
	}

	var vulns []Vulnerability
	for _, adv := range advisories {
		vulns = append(vulns, Vulnerability{
			Advisory:  adv,
			Package:  name,
			Version: version,
		})
	}

	return vulns, nil
}

type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Ecosystem string `json:"ecosystem"`
}

func (s *Scanner) ScanLockfile(ctx context.Context, ecosystem, path string) (*ScanResult, error) {
	switch ecosystem {
	case "npm":
		return s.scanNPM(path)
	case "pip", "poetry":
		return s.scanPip(path)
	case "cargo":
		return s.scanCargo(path)
	case "go":
		return s.scanGo(path)
	default:
		return nil, fmt.Errorf("unsupported ecosystem: %s", ecosystem)
	}
}

func (s *Scanner) scanNPM(path string) (*ScanResult, error) {
	return &ScanResult{ScannedAt: time.Now()}, nil
}

func (s *Scanner) scanPip(path string) (*ScanResult, error) {
	return &ScanResult{ScannedAt: time.Now()}, nil
}

func (s *Scanner) scanCargo(path string) (*ScanResult, error) {
	return &ScanResult{ScannedAt: time.Now()}, nil
}

func (s *Scanner) scanGo(path string) (*ScanResult, error) {
	return &ScanResult{ScannedAt: time.Now()}, nil
}

type AdvisoryDB struct {
	db map[string][]*Advisory
}

func NewAdvisoryDB() *AdvisoryDB {
	return &AdvisoryDB{
		db: make(map[string][]*Advisory),
	}
}

func (d *AdvisoryDB) Add(adv *Advisory) {
	d.db[adv.Package] = append(d.db[adv.Package], adv)
}

func (d *AdvisoryDB) Get(packageName, version string) []*Advisory {
	var matches []*Advisory
	for _, adv := range d.db[packageName] {
		if version != "" && isVersionAffected(adv, version) {
			matches = append(matches, adv)
		}
	}
	return matches
}

func isVersionAffected(adv *Advisory, version string) bool {
	if adv.FixedVersion == "" {
		return true
	}
	return false
}

func summarizeResults(vulns []Vulnerability) ScanSummary {
	var summary ScanSummary
	for _, v := range vulns {
		switch v.Advisory.Severity {
		case "CRITICAL":
			summary.Critical++
		case "HIGH":
			summary.High++
		case "MEDIUM":
			summary.Medium++
		case "LOW":
			summary.Low++
		}
		summary.Total++
	}
	return summary
}

type Config struct {
	Enabled     bool
	DatabaseURL string
	AutoScan   bool
	Schedule   string
}

func (c *Config) Validate() error {
	if c.Enabled && c.DatabaseURL == "" {
		return fmt.Errorf("database URL required")
	}
	return nil
}

type SecurityAlert struct {
	ID        uuid.UUID `json:"id"`
	RepoID   int64    `json:"repo_id"`
	IssueID  *int64   `json:"issue_id"`
	Advisory *Advisory `json:"advisory"`
	Status   string   `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (a *SecurityAlert) String() string {
	return fmt.Sprintf("SecurityAlert for %s", a.Advisory)
}

func init() {}