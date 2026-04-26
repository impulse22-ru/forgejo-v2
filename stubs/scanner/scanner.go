package scanner

import (
	"encoding/json"
	"time"
)

type Vulnerability struct {
	ID          string
	Package     string
	Severity    Severity
	CVE        string
	Description string
	FixedVersion string
}

type Severity string

const (
	SeverityLow     Severity = "LOW"
	SeverityMedium Severity = "MEDIUM"
	SeverityHigh   Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

type Advisory struct {
	ID          string         `json:"id"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Published    time.Time     `json:"published"`
	Affected     []string     `json:"affected"`
}

type ScanResult struct {
	Package    string          `json:"package"`
	Version    string          `json:"version"`
	Ecosystem  string          `json:"ecosystem"`
	Vulns     []Vulnerability `json:"vulnerabilities"`
	ScannedAt time.Time      `json:"scanned_at"`
}

type VulnerabilityScanner struct {
	advisoryDB map[string][]Advisory
}

func NewVulnerabilityScanner() *VulnerabilityScanner {
	return &VulnerabilityScanner{
		advisoryDB: make(map[string][]Advisory),
	}
}

func (v *VulnerabilityScanner) RegisterAdvisory(advisory Advisory) {
	v.advisoryDB[advisory.ID] = append(v.advisoryDB[advisory.ID], advisory)
}

func (v *VulnerabilityScanner) ScanPackage(pkg, version, ecosystem string) ScanResult {
	result := ScanResult{
		Package:   pkg,
		Version:   version,
		Ecosystem: ecosystem,
		ScannedAt: time.Now(),
	}

	result.Vulns = v.findVulnerabilities(pkg, version)
	return result
}

func (v *VulnerabilityScanner) findVulnerabilities(pkg, version string) []Vulnerability {
	return nil
}

type DependabotConfig struct {
	Version            int           `json:"version"`
	Updates            []UpdateConfig `json:"updates"`
	OpenPullRequestsLimit int         `json:"open-pull-requests-limit"`
}

type UpdateConfig struct {
	PackageEcosystem string   `json:"package-ecosystem"`
	Directory       string   `json:"directory"`
	Schedule        Schedule `json:"schedule"`
	Allow           []Allow  `json:"allow,omitempty"`
}

type Schedule struct {
	Interval string `json:"interval"`
	Day      string `json:"day,omitempty"`
	Time     string `json:"time,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type Allow struct {
	Type    string   `json:"type"`
	Updates []string `json:"updates"`
}

func ParseDependabotConfig(data []byte) (DependabotConfig, error) {
	var config DependabotConfig
	err := json.Unmarshal(data, &config)
	return config, err
}

type DependencyUpdate struct {
	Package   string
	From      string
	To        string
	Ecosystem string
	Type      string
}

func (d *DependencyUpdate) String() string {
	return d.Package + ": " + d.From + " -> " + d.To
}