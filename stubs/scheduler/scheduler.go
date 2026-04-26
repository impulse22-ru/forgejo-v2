package scheduler

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

var (
	ErrInvalidCron     = fmt.Errorf("invalid cron expression")
	ErrInvalidSchedule = fmt.Errorf("invalid schedule")
	ErrJobNotFound   = fmt.Errorf("job not found")
)

type CronSpec struct {
	Minute     string `yaml:"minute"`
	Hour       string `yaml:"hour"`
	DayOfMonth string `yaml:"day-of-month"`
	Month      string `yaml:"month"`
	DayOfWeek  string `yaml:"day-of-week"`
	Year      string `yaml:"year"`
}

func (c *CronSpec) String() string {
	return fmt.Sprintf("%s %s %s %s %s", c.Minute, c.Hour, c.DayOfMonth, c.Month, c.DayOfWeek)
}

func (c *CronSpec) Parse() (string, error) {
	parts := strings.Fields(c.String())
	if len(parts) != 5 {
		return "", ErrInvalidCron
	}
	return strings.Join(parts, " "), nil
}

func ParseCron(expr string) (*CronSpec, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("%w: expected 5 fields, got %d", ErrInvalidCron, len(parts))
	}

	return &CronSpec{
		Minute:     parts[0],
		Hour:       parts[1],
		DayOfMonth: parts[2],
		Month:      parts[3],
		DayOfWeek:  parts[4],
	}, nil
}

func (c *CronSpec) Validate() error {
	if !isValidField(c.Minute, 0, 59) {
		return fmt.Errorf("invalid minute: %s", c.Minute)
	}
	if !isValidField(c.Hour, 0, 23) {
		return fmt.Errorf("invalid hour: %s", c.Hour)
	}
	if !isValidField(c.DayOfMonth, 1, 31) {
		return fmt.Errorf("invalid day of month: %s", c.DayOfMonth)
	}
	if !isValidField(c.Month, 1, 12) {
		return fmt.Errorf("invalid month: %s", c.Month)
	}
	if !isValidField(c.DayOfWeek, 0, 6) {
		return fmt.Errorf("invalid day of week: %s", c.DayOfWeek)
	}
	return nil
}

func isValidField(field string, min, max int) bool {
	if field == "*" {
		return true
	}
	if strings.Contains(field, ",") {
		for _, part := range strings.Split(field, ",") {
			if !isValidField(strings.TrimSpace(part), min, max) {
				return false
			}
		}
		return true
	}
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return false
		}
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			return false
		}
		end, err := strconv.Atoi(parts[1])
		if err != nil {
			return false
		}
		return start >= min && end <= max && start <= end
	}
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil {
			return false
		}
		return step > 0
	}
	num, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return num >= min && num <= max
}

func ValidateCronExpression(expr string) bool {
	_, err := ParseCron(expr)
	return err == nil
}

func NormalizeCronExpression(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	expr = strings.ReplaceAll(expr, "\t", " ")
	expr = strings.ReplaceAll(expr, "  ", " ")
	parts := strings.Fields(expr)
	if len(parts) == 5 {
		return strings.Join(parts, " "), nil
	}
	if len(parts) == 6 {
		parts = parts[:5]
		return strings.Join(parts, " "), nil
	}
	return "", ErrInvalidCron
}

type Schedule struct {
	Cron       string
	Timezone  *time.Location
	Enabled   bool
	Skip      bool
	Workflow  string
	RepoID    int64
	Input     map[string]any
}

func (s *Schedule) Validate() error {
	if s.Cron == "" {
		return fmt.Errorf("cron expression is required")
	}
	if !ValidateCronExpression(s.Cron) {
		return fmt.Errorf("invalid cron expression: %s", s.Cron)
	}
	return nil
}

func (s *Schedule) NextRun(after time.Time) time.Time {
	spec, err := ParseCron(s.Cron)
	if err != nil {
		return time.Time{}
	}

	location := s.Timezone
	if location == nil {
		location = time.UTC
	}

	next := time.Date(after.Year(), after.Month(), after.Day(), after.Hour(), after.Minute(), 0, 0, location)

	for i := 0; i < 525600; i++ {
		if matchesCron(spec, next) && next.After(after) {
			return next
		}
		next = next.Add(time.Minute)
	}

	return time.Time{}
}

func matchesCron(spec *CronSpec, t time.Time) bool {
	return matchesField(spec.Minute, t.Minute()) &&
		matchesField(spec.Hour, t.Hour()) &&
		matchesField(spec.DayOfMonth, t.Day()) &&
		matchesField(spec.Month, int(t.Month())) &&
		matchesField(spec.DayOfWeek, int(t.Weekday()))
}

func matchesField(field string, value int) bool {
	if field == "*" {
		return true
	}

	strValue := strconv.Itoa(value)

	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil {
			return false
		}
		return value%step == 0
	}

	if strings.Contains(field, ",") {
		for _, part := range strings.Split(field, ",") {
			if strings.TrimSpace(part) == strValue {
				return true
			}
		}
		return false
	}

	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return false
		}
		start, _ := strconv.Atoi(parts[0])
		end, _ := strconv.Atoi(parts[1])
		return value >= start && value <= end
	}

	return field == strValue
}

func (s *Schedule) ToCron() string {
	return s.Cron
}

type Scheduler struct {
	schedules map[string]*Schedule
	cron    *Cron
	runner  JobRunner
}

type Cron struct {
	entries map[string]*Entry
}

type Entry struct {
	ID        string
	Schedule *Schedule
	Job      func() error
	LastRun  *time.Time
	NextRun  *time.Time
}

func NewScheduler(runner JobRunner) *Scheduler {
	return &Scheduler{
		schedules: make(map[string]*Schedule),
		cron:     &Cron{entries: make(map[string]*Entry)},
		runner:  runner,
	}
}

func (s *Scheduler) AddSchedule(id string, schedule *Schedule) error {
	if err := schedule.Validate(); err != nil {
		return err
	}

	s.schedules[id] = schedule
	s.cron.entries[id] = &Entry{
		ID:        id,
		Schedule: schedule,
		Job: func() error {
			return s.runner.Run(schedule.Workflow, schedule.RepoID, schedule.Input)
		},
	}

	return nil
}

func (s *Scheduler) RemoveSchedule(id string) {
	delete(s.schedules, id)
	delete(s.cron.entries, id)
}

func (s *Scheduler) GetSchedule(id string) *Schedule {
	return s.schedules[id]
}

func (s *Scheduler) ListSchedules() []*Schedule {
	var schedules []*Schedule
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}
	return schedules
}

func (s *Scheduler) GetNextRun(id string) *time.Time {
	if entry, ok := s.cron.entries[id]; ok {
		entry.NextRun = entry.Schedule.NextRun(time.Now())
		return entry.NextRun
	}
	return nil
}

func (s *Scheduler) GetLastRun(id string) *time.Time {
	if entry, ok := s.cron.entries[id]; ok {
		return entry.LastRun
	}
	return nil
}

func (s *Scheduler) RunPending() error {
	for id, schedule := range s.schedules {
		if !schedule.Enabled {
			continue
		}

		nextRun := schedule.NextRun(time.Now())
		if nextRun.IsZero() {
			continue
		}

		if time.Now().After(nextRun) {
			if err := s.runner.Run(schedule.Workflow, schedule.RepoID, schedule.Input); err != nil {
				log.Error("Failed to run scheduled workflow %s: %v", id, err)
				continue
			}

			schedule.Skip = true
		}
	}

	return nil
}

func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.RunPending()
			}
		}
	}()
}

func (s *Scheduler) Stop() {}

type JobRunner interface {
	Run(workflow string, repoID int64, input map[string]any) error
}

type CronParser interface {
	Parse(expr string) (*CronSpec, error)
}

var CommonSchedules = map[string]string{
	"@hourly":      "0 * * * *",
	"@daily":      "0 0 * * *",
	"@weekly":     "0 0 * * 0",
	"@monthly":    "0 0 1 * *",
	"@yearly":     "0 0 1 1 *",
	"@midnight":   "0 0 * * *",
	"@weekdays":   "0 0 * * 1-5",
}

func GetCommonSchedule(name string) (string, bool) {
	cron, ok := CommonSchedules[name]
	return cron, ok
}

func IsValidScheduleName(name string) bool {
	_, ok := CommonSchedules[name]
	return ok
}

var weekdayNames = map[string]int{
	"sunday":    0,
	"monday":   1,
	"tuesday":  2,
	"wednesday": 3,
	"thursday": 4,
	"friday":   5,
	"saturday": 6,
}

var monthNames = map[string]int{
	"january":   1,
	"february": 2,
	"march":    3,
	"april":    4,
	"may":      5,
	"june":     6,
	"july":     7,
	"august":   8,
	"september": 9,
	"october":  10,
	"november": 11,
	"december": 12,
}

func ExpandScheduleNames(expr string) string {
	expr = strings.ToLower(expr)

	for name, cron := range CommonSchedules {
		expr = strings.ReplaceAll(expr, name, cron)
	}

	for name, day := range weekdayNames {
		expr = strings.ReplaceAll(expr, name, strconv.Itoa(day))
	}

	for name, month := range monthNames {
		expr = strings.ReplaceAll(expr, name, strconv.Itoa(month))
	}

	return expr
}

var activityNamePattern = regexp.MustCompile(`^@?(hourly|daily|weekly|monthly|yearly|midnight|weekdays)$`)

func IsActivitySchedule(expr string) bool {
	return activityNamePattern.MatchString(strings.ToLower(expr))
}

func CreateScheduleFromActivity(expr string) (*Schedule, error) {
	cron, ok := GetCommonSchedule(expr)
	if !ok {
		return nil, fmt.Errorf("unknown schedule: %s", expr)
	}

	return &Schedule{
		Cron:     cron,
		Enabled: true,
	}, nil
}

func init() {
	setting.Actions.Enabled = true
}