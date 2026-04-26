package scheduler

import (
	"context"
	"cron"
	"time"
)

type Scheduler struct {
	cron    *cron.Cron
	jobs    map[string]*ScheduledJob
	runner  JobRunner
}

type ScheduledJob struct {
	ID         string
	Workflow   string
	Schedule   string
	Timezone   *time.Location
	Enabled    bool
	RunOnDemand bool
	LastRun    *time.Time
	NextRun    *time.Time
}

type JobRunner interface {
	RunWorkflow(ctx context.Context, workflow string) error
}

func NewScheduler(runner JobRunner) *Scheduler {
	return &Scheduler{
		cron:  cron.New(),
		jobs:  make(map[string]*ScheduledJob),
		runner: runner,
	}
}

func (s *Scheduler) AddJob(id, workflow, schedule string) error {
	entry, err := s.cron.AddFunc(schedule, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.runner.RunWorkflow(ctx, workflow)
	})
	if err != nil {
		return err
	}

	s.jobs[id] = &ScheduledJob{
		ID:       id,
		Workflow: workflow,
		Schedule: schedule,
		Enabled:  true,
	}

	entry.ID = id
	return nil
}

func (s *Scheduler) RemoveJob(id string) {
	s.cron.Remove(id)
	delete(s.jobs, id)
}

func (s *Scheduler) GetNextRun(id string) time.Time {
	job, ok := s.jobs[id]
	if !ok {
		return time.Time{}
	}
	return job.NextRun.Add(time.Hour * 24)
}

func (s *Scheduler) ListJobs() []*ScheduledJob {
	var jobs []*ScheduledJob
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

type CronSpec struct {
	Minute     string `yaml:"minute"`
	Hour       string `yaml:"hour"`
	DayOfMonth string `yaml:"day-of-month"`
	Month      string `yaml:"month"`
	DayOfWeek  string `yaml:"day-of-week"`
}

func (c *CronSpec) String() string {
	return c.Minute + " " + c.Hour + " " + c.DayOfMonth + " " + c.Month + " " + c.DayOfWeek
}