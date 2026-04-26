package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

type Action struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Inputs      map[string]Input   `yaml:"inputs"`
	Outputs    map[string]Output  `yaml:"outputs"`
	Runs        Using            `yaml:"runs"`

	raw   string
	Path string
}

type Input struct {
	Description string `yaml:"description"`
	Required   bool   `yaml:"required"`
	Default   any    `yaml:"default"`
}

type Output struct {
	Description string `yaml:"description"`
}

type Using struct {
	Using    string `yaml:"using"`
	Shell    string `yaml:"shell"`
	Cwd     string `yaml:"working-directory"`
	Entrypoint string `yaml:"entrypoint"`

	Type ActionType

	Sections []Step
}

type ActionType string

const (
	ActionTypeDocker    ActionType = "docker"
	ActionTypeNode   ActionType = "node"
	ActionTypeComposite ActionType = "composite"
	ActionTypeScript ActionType = "script"
)

func (a *Action) Parse(data []byte) (*Action, error) {
	if err := yaml.Unmarshal(data, a); err != nil {
		return nil, fmt.Errorf("failed to parse action: %w", err)
	}

	a.raw = string(data)
	a.Type = detectActionType(a)

	return a, nil
}

func (a *Action) TypeString() string {
	return string(a.Type)
}

func detectActionType(a *Action) ActionType {
	if a.Runs.Docker != "" {
		return ActionTypeDocker
	}
	if a.Runs.Node != "" {
		return ActionTypeNode
	}
	if len(a.Runs.Sections) > 0 {
		return ActionTypeComposite
	}
	return ActionTypeScript
}

func (a *Action) Validate() error {
	if a.Runs.Using == "" {
		return fmt.Errorf("runs.using is required")
	}
	return nil
}

func (a *Action) GetInput(name string) (any, bool) {
	input, ok := a.Inputs[name]
	return input.Default, ok
}

func (a *Action) GetOutput(name string) (*Output, bool) {
	output, ok := a.Outputs[name]
	return &output, ok
}

type CompositeRunner struct {
	action  *Action
	context *Context
	stderr  *log.Logger
	stdout  *log.Logger
}

func NewCompositeRunner(action *Action) *CompositeRunner {
	return &CompositeRunner{
		action: action,
		context: NewContext(),
	}
}

func (r *CompositeRunner) Run(ctx context.Context) error {
	if len(r.action.Runs.Sections) == 0 {
		return fmt.Errorf("no steps defined")
	}

	for _, step := range r.action.Runs.Sections {
		if err := r.runStep(ctx, step); err != nil {
			return err
		}
	}

	return nil
}

func (r *CompositeRunner) runStep(ctx context.Context, step Step) error {
	output := map[string]string{}

	for name, value := range step.Outputs {
		output[name] = value
		r.context.SetOutput(name, value)
	}

	return nil
}

func (r *CompositeRunner) SetOutput(name, value string) {
	r.context.SetOutput(name, value)
}

func (r *CompositeRunner) GetOutput(name string) string {
	return r.context.GetOutput(name)
}

func (r *CompositeRunner) Done() {
	r.context.Done()
}

type Context struct {
	Inputs   map[string]any
	Outputs  map[string]string
	Env     map[string]string
	Matrix  map[string]string
	Needs   map[string]string
	StartedAt time.Time
}

func NewContext() *Context {
	return &Context{
		Inputs:  make(map[string]any),
		Outputs: make(map[string]string),
		Env:    make(map[string]string),
		Matrix: make(map[string]string),
		Needs:  make(map[string]string),
	}
}

func (c *Context) GetInput(name string) (any, bool) {
	val, ok := c.Inputs[name]
	return val, ok
}

func (c *Context) SetInput(name string, value any) {
	c.Inputs[name] = value
}

func (c *Context) GetOutput(name string) string {
	return c.Outputs[name]
}

func (c *Context) SetOutput(name string, value string) {
	c.Outputs[name] = value
}

func (c *Context) GetEnv(name string) (string, bool) {
	val, ok := c.Env[name]
	return val, ok
}

func (c *Context) SetEnv(name, value string) {
	c.Env[name] = value
}

func (c *Context) Done() {}

type ReusableWorkflow struct {
	Name  string `yaml:"name"`
	On    string `yaml:"on"`
	Inputs map[string]Input `yaml:"inputs"`
	Secrets map[string]Input `yaml:"secrets"`
	Outputs map[string]Output `yaml:"outputs"`
	Jobs   map[string]Job `yaml:"jobs"`
}

func (r *ReusableWorkflow) Parse(data []byte) (*ReusableWorkflow, error) {
	if err := yaml.Unmarshal(data, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *ReusableWorkflow) Validate() error {
	if !r.IsReusable() {
		return fmt.Errorf("not a reusable workflow")
	}
	return nil
}

func (r *ReusableWorkflow) IsReusable() bool {
	_, hasWorkflowCall := r.On["workflow_call"]
	return hasWorkflowCall
}

func (r *ReusableWorkflow) GetOutput(name string) (*Output, bool) {
	output, ok := r.Outputs[name]
	return &output, ok
}

type WorkflowRef struct {
	Path        string `json:"path"`
	Ref         string `json:"ref"`
	Workflow    string `json:"workflow,omitempty"`
	Inputs     map[string]any `json:"inputs,omitempty"`
	Secrets    map[string]string `json:"secrets,omitempty"`
	SourceWorkflow string  `json:"source_workflow,omitempty"`
}

func (w *WorkflowRef) Parse(data []byte) (*WorkflowRef, error) {
	if err := yaml.Unmarshal(data, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *WorkflowRef) Validate() error {
	if w.Path == "" && w.Workflow == "" {
		return fmt.Errorf("path or workflow required")
	}
	return nil
}

func (w *WorkflowRef) Resolve() (string, error) {
	ref := w.Ref
	if ref == "" {
		ref = "HEAD"
	}
	return ref, nil
}

type ReusableRunner struct {
	workflow *ReusableWorkflow
	ref      *WorkflowRef
	context  *Context
}

func NewReusableRunner(workflow *ReusableWorkflow, ref *WorkflowRef) *ReusableRunner {
	return &ReusableRunner{
		workflow: workflow,
		ref:      ref,
		context:  NewContext(),
	}
}

func (r *ReusableRunner) Run(ctx context.Context) error {
	for name, input := range r.workflow.Inputs {
		if value, ok := r.ref.Inputs[name]; ok {
			r.context.SetInput(name, value)
		} else if input.Default != nil {
			r.context.SetInput(name, input.Default)
		}
	}

	jobName := r.findJob()
	job, ok := r.workflow.Jobs[jobName]
	if !ok {
		return fmt.Errorf("job not found: %s", jobName)
	}

	return r.runJob(ctx, job)
}

func (r *ReusableRunner) findJob() string {
	for name := range r.workflow.Jobs {
		return name
	}
	return ""
}

func (r *ReusableRunner) runJob(ctx context.Context, job Job) error {
	for _, step := range job.Steps {
		if err := r.runStep(ctx, step); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReusableRunner) runStep(ctx context.Context, step Step) error {
	return nil
}

func (r *ReusableRunner) SetOutputs(outputs map[string]string) {
	for name, value := range outputs {
		r.context.SetOutput(name, value)
	}
}

func (r *ReusableRunner) GetOutput(name string) string {
	return r.context.GetOutput(name)
}

type ActionRegistry struct {
	actions map[string]*Action
}

func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		actions: make(map[string]*Action),
	}
}

func (r *ActionRegistry) Register(path string, action *Action) {
	r.actions[path] = action
}

func (r *ActionRegistry) Get(path string) (*Action, bool) {
	action, ok := r.actions[path]
	return action, ok
}

func (r *ActionRegistry) List() []*Action {
	var list []*Action
	for _, action := range r.actions {
		list = append(list, action)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func IsValidActionPath(path string) bool {
	matched, _ := regexp.MatchString(`^[_a-zA-Z0-9][-_/a-zA-Z0-9]*$`, path)
	return matched
}

func NormalizeActionPath(path string) string {
	if path == "@" {
		return "actions/checkout"
	}
	if !contains(path, "/") {
		return "actions/" + path
	}
	return path
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || find(s, substr))
}

func find(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func init() {
	setting.Actions.Enabled = true
}