package actions

import (
	"context"
	"fmt"
)

type MatrixConfig struct {
	Include []map[string]string `yaml:"include"`
	Exclude []map[string]string `yaml:"exclude"`
}

type Job struct {
	Name    string
	ID      string
	Matrix  map[string][]string
	Steps   []Step
	Env     map[string]string
	RunsOn  string
}

type Step struct {
	Name    string
	Uses    string
	Run     string
	If      string
	Env     map[string]string
	With    map[string]string
}

type Workflow struct {
	Name  string
	On    string
	Jobs  map[string]Job
	Env   map[string]string
}

func NewWorkflow() *Workflow {
	return &Workflow{
		Jobs: make(map[string]Job),
		Env:  make(map[string]string),
	}
}

func (w *Workflow) ExpandMatrix() []Job {
	var jobs []Job

	for jobID, job := range w.Jobs {
		if len(job.Matrix) == 0 {
			jobs = append(jobs, job)
			continue
		}

		matrixCombinations := generateCombinations(job.Matrix)
		for _, combo := range matrixCombinations {
			newJob := job
			newJob.ID = jobID
			newJob.Name = job.Name
			jobs = append(jobs, newJob)
		}
	}

	return jobs
}

func generateCombinations(matrix map[string][]string) []map[string]string {
	if len(matrix) == 0 {
		return []map[string]string{{}}
	}

	var result []map[string]string
	for key, values := range matrix {
		var combos []map[string]string
		for _, val := range values {
			for _, existing := range result {
				combo := make(map[string]string)
				for k, v := range existing {
					combo[k] = v
				}
				combo[key] = val
				combos = append(combos, combo)
			}
		}
		result = combos
	}

	return result
}

type ExecutionResult struct {
	JobID    string
	Status   string
	Artifact int64
}

func (e *ExecutionResult) String() string {
	return fmt.Sprintf("Job: %s, Status: %s, Artifacts: %d", e.JobID, e.Status, e.Artifact)
}