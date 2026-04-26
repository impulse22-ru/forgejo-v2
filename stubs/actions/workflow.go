package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/yaml"
)

var (
	ErrInvalidWorkflow = errors.New("invalid workflow")
	ErrInvalidMatrix  = errors.New("invalid matrix configuration")
	ErrJobNotFound    = errors.New("job not found")
)

type Workflow struct {
	Name  string `yaml:"name"`
	On    string `yaml:"on"`

	Env              map[string]string `yaml:"env"`
	Defaults         *Defaults        `yaml:"defaults"`
	Concurrency     *Concurrency    `yaml:"concurrency"`
	Permissions     string          `yaml:"permissions"`
	Jobs            map[string]Job  `yaml:"jobs"`
	WorkflowDispatch *WorkflowDispatch `yaml:"workflow_dispatch"`

	raw    string
	Path  string
	Repo  string
	Event string
}

type Defaults struct {
	Run *RunDefaults `yaml:"run"`
}

type RunDefaults struct {
	Shell string `yaml:"shell"`
	Cwd   string `yaml:"working-directory"`
}

type Concurrency struct {
	Group           string            `yaml:"group"`
	CancelInProgress bool            `yaml:"cancel-in-progress"`
}

type WorkflowDispatch struct {
	Inputs map[string]WorkflowInput `yaml:"inputs"`
}

type WorkflowInput struct {
	Description string `yaml:"description"`
	Type       string `yaml:"type"`
	Required   bool   `yaml:"required"`
	Default   any    `yaml:"default"`
}

func (w *Workflow) Parse(data []byte) (*Workflow, error) {
	if err := yaml.Unmarshal(data, w); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidWorkflow, err)
	}
	w.raw = string(data)
	return w, nil
}

func (w *Workflow) ToJSON() ([]byte, error) {
	return json.Marshal(w)
}

func (w *Workflow) Validate() error {
	if len(w.Jobs) == 0 {
		return fmt.Errorf("%w: no jobs defined", ErrInvalidWorkflow)
	}

	for name, job := range w.Jobs {
		if err := job.Validate(); err != nil {
			return fmt.Errorf("job %s: %w", name, err)
		}
	}

	return nil
}

type Job struct {
	Name            string            `yaml:"name"`
	RunsOn          string            `yaml:"runs-on"`
	If             string            `yaml:"if"`
	Needs          []string          `yaml:"needs"`
	Outputs        map[string]string `yaml:"outputs"`
	Env           map[string]string `yaml:"env"`
	Variables     map[string]string `yaml:"variables"`
	Defaults      *JobDefaults    `yaml:"defaults"`
	Matrix        *Matrix         `yaml:"matrix"`
	Strategy      *Strategy       `yaml:"strategy"`
	TimeoutMinutes int           `yaml:"timeout-minutes"`
	ContinueOnError bool        `yaml:"continue-on-error"`
	If           string         `yaml:"if"`

	RawPermissions string `yaml:"permissions"`
	Permissions *Permissions `yaml:"-"`

	Steps         []Step    `yaml:"steps"`
	RawContainer string   `yaml:"-"`
	Container   *Container `yaml:"container"`
	Uses        string    `yaml:"uses"`
	With        map[string]any `yaml:"with"`
}

type JobDefaults struct {
	Run *RunDefaults `yaml:"run"`
}

type Container struct {
	Image string `yaml:"image"`
	Options string `yaml:"options"`
	Credentials *Credentials `yaml:"credentials"`
}

type Credentials struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (j *Job) Validate() error {
	if len(j.Steps) == 0 && j.Uses == "" {
		return fmt.Errorf("no steps defined")
	}

	if j.RunsOn == "" {
		j.RunsOn = "ubuntu-latest"
	}

	return nil
}

type Step struct {
	Name            string         `yaml:"name"`
	If             string         `yaml:"if"`
	Uses           string         `yaml:"uses"`
	Run            string         `yaml:"run"`
	Shell          string         `yaml:"shell"`
	WorkingDirectory string     `yaml:"working-directory"`
	Env           map[string]string `yaml:"env"`
	ContinueOnError bool        `yaml:"continue-on-error"`
	TimeoutMinutes int         `yaml:"timeout-minutes"`
	With          map[string]any `yaml:"with"`
	ID             string       `yaml:"id"`
	WorkingDir     string      `yaml:"-"`

	Credentials struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"credentials"`

	Outputs map[string]string `yaml:"outputs"`

	RawEnv map[string]any `yaml:"-"`
}

type Matrix struct {
	Include []map[string]string `yaml:"include"`
	Exclude []map[string]string `yaml:"exclude"`

	raw map[string][]string `yaml:"-"`
}

func (m *Matrix) Parse(raw map[string][]string) (*Matrix, error) {
	m.raw = raw
	m.Include = make([]map[string]string, 0)
	m.Exclude = make([]map[string]string, 0)

	return m, nil
}

func (m *Matrix) Expand() ([]map[string]string, error) {
	if len(m.raw) == 0 {
		return []map[string]string{{}}, nil
	}

	combos := generateCombinations(m.raw)

	for _, exclude := range m.Exclude {
		combos = filterExcludes(combos, exclude)
	}

	return combos, nil
}

func generateCombinations(matrix map[string][]string) []map[string]string {
	if len(matrix) == 0 {
		return []map[string]string{{}}
	}

	var result []map[string]string
	result = append(result, map[string]string{})

	for key, values := range matrix {
		var newResult []map[string]string
		for _, val := range values {
			for _, existing := range result {
				combo := make(map[string]string)
				for k, v := range existing {
					combo[k] = v
				}
				combo[key] = val
				newResult = append(newResult, combo)
			}
		}
		result = newResult
	}

	return result
}

func filterExcludes(combos []map[string]string, exclude map[string]string) []map[string]string {
	var filtered []map[string]string
	for _, combo := range combos {
		excludeCombo := true
		for key, val := range exclude {
			if combo[key] != val {
				excludeCombo = false
				break
			}
		}
		if !excludeCombo {
			filtered = append(filtered, combo)
		}
	}
	return filtered
}

func (m *Matrix) AddInclude(include map[string]string) {
	if m.Include == nil {
		m.Include = make([]map[string]string, 0)
	}
	m.Include = append(m.Include, include)
}

func (m *Matrix) AddExclude(exclude map[string]string) {
	if m.Exclude == nil {
		m.Exclude = make([]map[string]string, 0)
	}
	m.Exclude = append(m.Exclude, exclude)
}

type Strategy struct {
	FailFast     bool            `yaml:"fail-fast"`
	Matrix     map[string][]string `yaml:"matrix"`
	MaxParallel int             `yaml:"max-parallel"`
}

func (s *Strategy) GetMaxJobs() int {
	if s.MaxParallel <= 0 {
		return 0
	}
	return s.MaxParallel
}

type Permissions struct {
	Contents string `yaml:"contents"`
	Actions  string `yaml:"actions"`
	Packages string `yaml:"packages"`
	Pages   string `yaml:"pages"`
	PullRequests string `yaml:"pull-requests"`
	Statuses string `yaml:"statuses"`
}

type WorkflowFile struct {
	Path string
	Data []byte
}

func ParseWorkflowFile(data []byte) (*Workflow, error) {
	var workflow Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidWorkflow, err)
	}

	if err := workflow.Validate(); err != nil {
		return nil, err
	}

	return &workflow, nil
}

func ParseWorkflowFilePath(filePath string) (*Workflow, error) {
	data, err := log.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	workflow, err := ParseWorkflowFile(data)
	if err != nil {
		return nil, err
	}

	workflow.Path = filePath
	workflow.Repo = filepath.Dir(filepath.Dir(filePath))
	workflow.Event = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	return workflow, nil
}

type WorkflowRun struct {
	ID         int64     `json:"id"`
	WorkflowID string    `json:"workflow_id"`
	RepoID    int64     `json:"repo_id"`
	Event     string    `json:"event"`
	Status    string    `json:"status"`
	Conclusion string   `json:"conclusion"`
	RunNumber int      `json:"run_number"`
	RunAttempt int     `json:"run_attempt"`
	HeadSHA   string    `json:"head_sha"`
	Ref       string    `json:"ref"`
	Workflow  string    `json:"workflow"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Jobs     []JobRun `json:"jobs"`
}

type JobRun struct {
	ID        int64    `json:"id"`
	Name      string  `json:"name"`
	Status   string  `json:"status"`
	Conclusion string `json:"conclusion"`
	StartedAt time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Steps    []StepRun `json:"steps"`
}

type StepRun struct {
	Name      string `json:"name"`
	Status   string `json:"status"`
	Conclusion string `json:"conclusion"`
	Number   int    `json:"number"`
}

type MatrixConfig struct {
	OS         []string `json:"os"`
	Node       []string `json:"node"`
	Python     []string `json:"python"`
	Go        []string `json:"go"`
	Ruby       []string `json:"ruby"`
	Include    []map[string]string `json:"include"`
	Exclude    []map[string]string `json:"exclude"`
}

func ParseMatrix(config map[string]any) (map[string][]string, error) {
	result := make(map[string][]string)

	for key, value := range config {
		switch v := value.(type) {
		case []any:
			var values []string
			for _, item := range v {
				switch val := item.(type) {
				case string:
					values = append(values, val)
				case int:
					values = append(values, strconv.Itoa(val))
				}
			}
			result[key] = values
		case string:
			result[key] = []string{v}
		}
	}

	return result, nil
}

type WorkflowFilter struct {
	Event     string
	Ref      string
	Branch   string
	Path     string
	Tag     string
	Actor   string
}

func (f *WorkflowFilter) ShouldRun(w *Workflow) bool {
	if f.Event != "" && f.Event != w.Event {
		return false
	}

	return true
}

func ListWorkflows(repoPath string) ([]WorkflowFile, error) {
	workflowDir := path.Join(repoPath, ".forgejo", "workflows")
	if _, err := log.Stat(workflowDir); err != nil {
		return nil, err
	}

	files, err := log.ReadDir(workflowDir)
	if err != nil {
		return nil, err
	}

	var workflows []WorkflowFile
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yml" || filepath.Ext(file.Name()) == ".yaml" {
			data, err := log.ReadFile(path.Join(workflowDir, file.Name()))
			if err != nil {
				continue
			}
			workflows = append(workflows, WorkflowFile{
				Path: path.Join(workflowDir, file.Name()),
				Data: data,
			})
		}
	}

	return workflows, nil
}

type WorkflowScheduler struct {
	jobs map[string]*ScheduledJob
}

type ScheduledJob struct {
	ID         string
	Workflow   string
	RepoID    int64
	Schedule  string
	Timezone  *time.Location
	Enabled   bool
	LastRun   *time.Time
	NextRun   *time.Time
	Input    map[string]any
}

func NewWorkflowScheduler() *WorkflowScheduler {
	return &WorkflowScheduler{
		jobs: make(map[string]*ScheduledJob),
	}
}

func (s *WorkflowScheduler) AddJob(job *ScheduledJob) {
	s.jobs[job.ID] = job
}

func (s *WorkflowScheduler) RemoveJob(id string) {
	delete(s.jobs, id)
}

func (s *WorkflowScheduler) GetJob(id string) *ScheduledJob {
	return s.jobs[id]
}

func (s *WorkflowScheduler) ListJobs() []*ScheduledJob {
	var jobs []*ScheduledJob
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].ID < jobs[j].ID
	})
	return jobs
}

func (s *WorkflowScheduler) FindDueJobs() []*ScheduledJob {
	now := time.Now()
	var due []*ScheduledJob
	for _, job := range s.jobs {
		if !job.Enabled {
			continue
		}
		if job.NextRun == nil || !now.Before(*job.NextRun) {
			due = append(due, job)
		}
	}
	return due
}

func IsValidCronExpression(expr string) bool {
matched, _ := regexp.MatchString(`^[^\*]+/[^\ ]+ [^\*]+/[^\*]+ [^\*]+/[^\*]+ [^\*]+/[^\*]+ [^\*]+$`, expr)
return matched
}

func ParseCronExpression(expr string) (time.Time, error) {
	return time.Parse("2006 1 2 15 04", expr)
}

func (s *Workflow) GetJob(jobName string) (*Job, error) {
	job, ok := s.Jobs[jobName]
	if !ok {
		return nil, ErrJobNotFound
	}
	return &job, nil
}

func (s *Workflow) GetJobsNeeding(jobName string) ([]*Job, error) {
	job, err := s.GetJob(jobName)
	if err != nil {
		return nil, err
	}

	var jobs []*Job
	for name, j := range s.Jobs {
		for _, need := range j.Needs {
			if need == jobName {
				jobs = append(jobs, &j)
			}
		}
	}

	return jobs, nil
}

func init() {}