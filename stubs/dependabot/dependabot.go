package dependabot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"github.com/gofrs/uuid"
)

var (
	ErrConfigNotFound = fmt.Errorf("dependabot config not found")
	ErrUpdateFailed = fmt.Errorf("failed to create update")
)

type Config struct {
	Version            int           `yaml:"version" json:"version"`
	Updates           []UpdateConfig  `yaml:"updates" json:"updates"`
	VersioningConfig   string        `yaml:"versioning-options,omitempty" json:"versioning-options,omitempty"`
}

type UpdateConfig struct {
	PackageEcosystem PackageEcosystem `yaml:"package-ecosystem" json:"package-ecosystem"`
	Directory       string      `yaml:"directory" json:"directory"`
	Schedule       *Schedule   `yaml:"schedule" json:"schedule"`
	OpenPullRequestsLimit int      `yaml:"open-pull-requests-limit" json:"open-pull-requests-limit"`
	TargetBranch   string      `yaml:"target-branch,omitempty" json:"target-branch,omitempty"`
	Labels        []string    `yaml:"labels,omitempty" json:"labels,omitempty"`
	Assignees     []string    `yaml:"assignees,omitempty" json:"assignees,omitempty"`
	Reviewers     []string    `yaml:"reviewers,omitempty" json:"reviewers,omitempty"`
	CommitMessage string     `yaml:"commit-message,omitempty" json:"commit-message,omitempty"`
	Milestone    string     `yaml:"milestone,omitempty" json:"milestone,omitempty"`
	Ignore       []IgnoreConfig `yaml:"ignore,omitempty" json:"ignore,omitempty"`
	Allow        []AllowConfig `yaml:"allow,omitempty" json:"allow,omitempty"`
}

type PackageEcosystem string

const (
	EcosystemNPM     PackageEcosystem = "npm"
	EcosystemPip     PackageEcosystem = "pip"
	EcosystemGo      PackageEcosystem = "go"
	EcosystemCargo   PackageEcosystem = "cargo"
	EcosystemMaven  PackageEcosystem = "maven"
	EcosystemNuGet PackageEcosystem = "nuget"
	EcosystemDocker PackageEcosystem = "docker"
	EcosystemHex    PackageEcosystem = "hex"
	EcosystemPub    PackageEcosystem = "pub"
)

type Schedule struct {
	Interval string `yaml:"interval" json:"interval"`
	Day      string `yaml:"day,omitempty" json:"day,omitempty"`
	Time     string `yaml:"time,omitempty" json:"time,omitempty"`
	Timezone string `yaml:"timezone,omitempty" json:"timezone,omitempty"`
}

func (s *Schedule) Validate() error {
	valid := map[string]bool{
		"daily":   true,
		"weekly":  true,
		"monthly": true,
	}
	if !valid[s.Interval] {
		return fmt.Errorf("invalid interval: %s", s.Interval)
	}
	return nil
}

type IgnoreConfig struct {
	DependencyName string   `yaml:"dependency-name,omitempty" json:"dependency-name,omitempty"`
	Versions    []string `yaml:"versions,omitempty" json:"versions,omitempty"`
	UpdateTypes []string `yaml:"update-types,omitempty" json:"update-types,omitempty"`
}

type AllowConfig struct {
	Type    string   `yaml:"type" json:"type"`
	Updates []string `yaml:"updates,omitempty" json:"updates,omitempty"`
}

func ParseConfig(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) Validate() error {
	for _, update := range c.Updates {
		if err := update.Schedule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type DependencyUpdate struct {
	Name          string           `yaml:"name" json:"name"`
	Version      string           `yaml:"version" json:"version"`
	Type         string           `yaml:"type" json:"type"`
	Ecosystem    PackageEcosystem `yaml:"ecosystem" json:"ecosystem"`
	Directory   string          `yaml:"directory" json:"directory"`
	Dependants  []string        `yaml:"dependants,omitempty" json:"dependants,omitempty"`
}

func (d *DependencyUpdate) String() string {
	return fmt.Sprintf("%s: %s", d.Name, d.Version)
}

type DependabotRunner struct {
	config *Config
	client *http.Client
}

func NewDependabotRunner(config *Config) *DependabotRunner {
	return &DependabotRunner{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *DependabotRunner) Run(ctx context.Context, repoID int64) ([]DependencyUpdate, error) {
	var updates []DependencyUpdate

	for _, update := range r.config.Updates {
		deps, err := r.checkUpdates(ctx, update)
		if err != nil {
			log.Error("Failed to check updates for %s: %v", update.PackageEcosystem, err)
			continue
		}
		updates = append(updates, deps...)
	}

	return updates, nil
}

func (r *DependabotRunner) checkUpdates(ctx context.Context, config UpdateConfig) ([]DependencyUpdate, error) {
	var updates []DependencyUpdate

	switch config.PackageEcosystem {
	case "npm":
		updates = r.checkNPM(ctx, config.Directory)
	case "pip":
		updates = r.checkPip(ctx, config.Directory)
	case "go":
		updates = r.checkGo(ctx, config.Directory)
	case "cargo":
		updates = r.checkCargo(ctx, config.Directory)
	}

	return updates, nil
}

func (r *DependabotRunner) checkNPM(ctx context.Context, dir string) []DependencyUpdate {
	update := DependencyUpdate{
		Name:      "lodash",
		Version:   "4.17.20",
		Type:      "version-update:semver:minor",
		Ecosystem: "npm",
		Directory: dir,
	}
	return []DependencyUpdate{update}
}

func (r *DependabotRunner) checkPip(ctx context.Context, dir string) []DependencyUpdate {
	update := DependencyUpdate{
		Name:      "requests",
		Version:   "2.28.0",
		Type:      "version-update:semver:minor",
		Ecosystem: "pip",
		Directory: dir,
	}
	return []DependencyUpdate{update}
}

func (r *DependabotRunner) checkGo(ctx context.Context, dir string) []DependencyUpdate {
	update := DependencyUpdate{
		Name:      "github.com/example/pkg",
		Version:   "v1.2.0",
		Type:      "version-update:semver:minor",
		Ecosystem: "go",
		Directory: dir,
	}
	return []DependencyUpdate{update}
}

func (r *DependabotRunner) checkCargo(ctx context.Context, dir string) []DependencyUpdate {
	update := DependencyUpdate{
		Name:      "tokio",
		Version:   "1.25.0",
		Type:      "version-update:semver:minor",
		Ecosystem: "cargo",
		Directory: dir,
	}
	return []DependencyUpdate{update}
}

func (r *DependabotRunner) CreatePR(ctx context.Context, update DependencyUpdate) (int64, error) {
	return 0, nil
}

type UpdateRequest struct {
	RepoID        int64
	Ecosystem     PackageEcosystem
	Directory   string
	Dependencies []DependencyUpdate
	Config      UpdateConfig
	ScheduledAt time.Time
}

type UpdateJob struct {
	ID          uuid.UUID      `json:"id"`
	RepoID      int64         `json:"repo_id"`
	Status      string       `json:"status"`
	Error       string       `json:"error,omitempty"`
	Dependencies []DependencyUpdate `json:"dependencies"`
	PRsCreated int           `json:"prs_created"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

func (j *UpdateJob) String() string {
	return fmt.Sprintf("Job{id=%s, status=%s, prs=%d}", j.ID, j.Status, j.PRsCreated)
}

type JobStore struct {
	jobs map[uuid.UUID]*UpdateJob
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[uuid.UUID]*UpdateJob),
	}
}

func (s *JobStore) Create(job *UpdateJob) {
	s.jobs[job.ID] = job
}

func (s *JobStore) Get(id uuid.UUID) (*UpdateJob, bool) {
	job, ok := s.jobs[id]
	return job, ok
}

func (s *JobStore) UpdateStatus(id uuid.UUID, status string, err error) {
	if job, ok := s.jobs[id]; ok {
		job.Status = status
		if err != nil {
			job.Error = err.Error()
		}
		job.UpdatedAt = time.Now()
	}
}

func (s *JobStore) ListByRepo(repoID int64) []*UpdateJob {
	var jobs []*UpdateJob
	for _, job := range s.jobs {
		if job.RepoID == repoID {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func FindDependabotFile(repoPath string) (string, error) {
	patterns := []string{
		repoPath + "/.github/dependabot.yml",
		repoPath + "/.github/dependabot.yaml",
		repoPath + "/.forgejo/dependabot.yml",
		repoPath + "/dependabot.yml",
	}

	for _, p := range patterns {
		if strings.HasPrefix(p, repoPath) {
			return p, nil
		}
	}

	return "", ErrConfigNotFound
}

func LoadDependabotConfig(repoPath string) (*Config, error) {
	path, err := FindDependabotFile(repoPath)
	if err != nil {
		return nil, err
	}

	data, err := http.Get(path)
	if err != nil {
		return nil, err
	}

	return ParseConfig(data)
}

func GetEcosystems() []string {
	return []string{
		"npm",
		"pip",
		"go",
		"cargo",
		"maven",
		"nuget",
		"docker",
		"hex",
		"pub",
	}
}

func init() {}