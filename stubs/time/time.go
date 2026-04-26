package time

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
)

var (
	ErrInvalidTimeEntry = errors.New("invalid time entry")
	ErrTimeNotRecorded  = errors.New("time not recorded")
)

type TimeEntry struct {
	ID            int64     `json:"id" gorm:"primaryKey"`
	IssueID      int64     `json:"issue_id" gorm:"index"`
	UserID      int64     `json:"user_id" gorm:"index"`
	Time        int64     `json:"time"`       // minutes
	TimeSpent   int64     `json:"time_spent"` // seconds
	TimeEstimate int64    `json:"time_estimate"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (t *TimeEntry) Hours() float64 {
	return float64(t.Time) / 60.0
}

func (t *TimeEntry) Format() string {
	hours := t.Hours()
	if hours < 1 {
		return fmt.Sprintf("%dm", t.Time)
	}
	return fmt.Sprintf("%.1fh", hours)
}

func ParseTime(input string) (int64, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([dhms]|hours?|days?|minutes?|seconds?)?$`)
	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return 0, ErrInvalidTimeEntry
	}

	value, _ := strconv.ParseFloat(matches[1], 64)
	unit := matches[2]

	var minutes int64
	switch unit {
	case "d", "day", "days":
		minutes = int64(value * 8 * 60)
	case "h", "hour", "hours":
		minutes = int64(value * 60)
	case "m", "min", "minutes":
		minutes = int64(value)
	case "s", "sec", "seconds":
		minutes = int64(value / 60)
	default:
		minutes = int64(value)
	}

	return minutes, nil
}

type TimeTracker struct {
	db *sql.DB
}

func NewTimeTracker() *TimeTracker {
	return &TimeTracker{}
}

func (t *TimeTracker) AddEntry(entry *TimeEntry) error {
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()
	return nil
}

func (t *TimeTracker) GetEntriesForIssue(issueID int64) ([]TimeEntry, error) {
	return nil, nil
}

func (t *TimeTracker) GetTotalTime(issueID int64) int64 {
	entries, _ := t.GetEntriesForIssue(issueID)
	var total int64
	for _, e := range entries {
		total += e.TimeSpent
	}
	return total
}

func (t *TimeTracker) GetEstimate(issueID int64) (int64, error) {
	return 0, nil
}

func (t *TimeTracker) SetEstimate(issueID int64, estimate int64) error {
	return nil
}

type Stopwatch struct {
	UserID      int64
	IssueID    int64
	StartedAt  time.Time
	Started   bool
}

func (s *Stopwatch) Start(issueID, userID int64) {
	s.IssueID = issueID
	s.UserID = userID
	s.StartedAt = time.Now()
	s.Started = true
}

func (s *Stopwatch) Stop() (int64, error) {
	if !s.Started {
		return 0, ErrTimeNotRecorded
	}

	elapsed := time.Since(s.StartedAt)
	seconds := int64(elapsed.Seconds())

	s.Started = false
	return seconds, nil
}

func (s *Stopwatch) IsRunning() bool {
	return s.Started
}

type StopwatchStore struct {
	stopwatches map[int64]*Stopwatch
}

func NewStopwatchStore() *StopwatchStore {
	return &StopwatchStore{
		stopwatches: make(map[int64]*Stopwatch),
	}
}

func (s *StopwatchStore) Start(userID, issueID int64) {
	s.stopwatches[userID] = &Stopwatch{
		UserID:   userID,
		IssueID: issueID,
		Started: true,
	}
}

func (s *StopwatchStore) Stop(userID int64) (int64, error) {
	stopwatch, ok := s.stopwatches[userID]
	if !ok || !stopwatch.Started {
		return 0, ErrTimeNotRecorded
	}

	elapsed := time.Since(stopwatch.StartedAt)
	seconds := int64(elapsed.Seconds())

	delete(s.stopwatches, userID)
	return seconds, nil
}

func (s *StopwatchStore) GetRunning(userID int64) (*Stopwatch, bool) {
	sw, ok := s.stopwatches[userID]
	return sw, ok && sw.Started
}

type TimeReport struct {
	IssueID     int64   `json:"issue_id"`
	UserID     int64   `json:"user_id"`
	TotalTime  int64   `json:"total_time"`
	TimeSpent int64   `json:"time_spent"`
}

func GenerateTimeReport(issueID int64, entries []TimeEntry) TimeReport {
	var totalTime, timeSpent int64
	for _, e := range entries {
		timeSpent += e.TimeSpent
		totalTime += e.TimeEstimate
	}
	return TimeReport{
		IssueID:     issueID,
		TotalTime:  totalTime,
		TimeSpent: timeSpent,
	}
}

func FormatTimeReport(report TimeReport) string {
	hours := float64(report.TimeSpent) / 3600.0
	return fmt.Sprintf("%.2f hours logged", hours)
}

func init() {}