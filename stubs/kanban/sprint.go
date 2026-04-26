package kanban

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"
)

var (
	ErrSprintNotFound  = errors.New("sprint not found")
	ErrInvalidSprint = errors.New("invalid sprint")
)

type Sprint struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	RepoID     int64     `json:"repo_id" gorm:"index"`
	Name       string    `json:"name"`
	Goal       string    `json:"goal"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	State      string    `json:"state"`
	ClosedAt   *time.Time `json:"closed_at"`
	Progress   int       `json:"progress"`
	Points     int       `json:"points"`
	Issues     []int64   `json:"issues" gorm:"-"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Sprint) String() string {
	return fmt.Sprintf("Sprint{id=%d, name=%s, state=%s}", s.ID, s.Name, s.State)
}

func (s *Sprint) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !s.EndDate.IsZero() && !s.StartDate.IsZero() && s.EndDate.Before(s.StartDate) {
		return fmt.Errorf("end date must be after start date")
	}
	return nil
}

func (s *Sprint) IsActive() bool {
	now := time.Now()
	return s.State == "active" && now.After(s.StartDate) && now.Before(s.EndDate)
}

func (s *Sprint) IsFuture() bool {
	return time.Now().Before(s.StartDate)
}

func (s *Sprint) IsPast() bool {
	return time.Now().After(s.EndDate)
}

func (s *Sprint) Velocity() int {
	return s.Points
}

func (s *Sprint) DaysRemaining() int {
	if s.EndDate.IsZero() {
		return 0
	}
	days := int(time.Until(s.EndDate).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func (s *Sprint) GetIssues() []int64 {
	return s.Issues
}

func (s *Sprint) AddIssue(issueID int64) {
	s.Issues = append(s.Issues, issueID)
}

func (s *Sprint) RemoveIssue(issueID int64) {
	for i, id := range s.Issues {
		if id == issueID {
			s.Issues = append(s.Issues[:i], s.Issues[i+1:]...)
			break
		}
	}
}

type SprintStore struct {
	db *sql.DB
}

func NewSprintStore() *SprintStore {
	return &SprintStore{}
}

func (s *SprintStore) Create(sprint *Sprint) error {
	validate := sprint.Validate()
	if validate != nil {
		return validate
	}

	sprint.State = "active"
	sprint.CreatedAt = time.Now()
	sprint.UpdatedAt = time.Now()

	return nil
}

func (s *SprintStore) Get(id int64) (*Sprint, error) {
	if id == 0 {
		return nil, ErrSprintNotFound
	}
	return nil, nil
}

func (s *SprintStore) GetByRepo(repoID int64) ([]Sprint, error) {
	return nil, nil
}

func (s *SprintStore) GetActive(repoID int64) (*Sprint, error) {
	sprints, err := s.GetByRepo(repoID)
	if err != nil {
		return nil, err
	}

	for i := range sprints {
		if sprints[i].IsActive() {
			return &sprints[i], nil
		}
	}

	return nil, nil
}

func (s *SprintStore) Update(sprint *Sprint) error {
	sprint.UpdatedAt = time.Now()
	return nil
}

func (s *SprintStore) Close(id int64) error {
	sprint, err := s.Get(id)
	if err != nil {
		return err
	}

	sprint.State = "closed"
	now := time.Now()
	sprint.ClosedAt = &now
	sprint.UpdatedAt = time.Now()

	return s.Update(sprint)
}

func (s *SprintStore) Delete(id int64) error {
	sprint, err := s.Get(id)
	if err != nil {
		return err
	}

	sprint.Issues = nil

	return nil
}

type SprintService struct {
	store *SprintStore
}

func NewSprintService(store *SprintStore) *SprintService {
	return &SprintService{store: store}
}

func (s *SprintService) AddIssueToSprint(sprintID, issueID int64) error {
	sprint, err := s.store.Get(sprintID)
	if err != nil {
		return err
	}

	sprint.AddIssue(issueID)
	return s.store.Update(sprint)
}

func (s *SprintService) RemoveIssueFromSprint(sprintID, issueID int64) error {
	sprint, err := s.store.Get(sprintID)
	if err != nil {
		return err
	}

	sprint.RemoveIssue(issueID)
	return s.store.Update(sprint)
}

func (s *SprintService) MoveIssueBetweenSprints(fromSprintID, toSprintID, issueID int64) error {
	if err := s.RemoveIssueFromSprint(fromSprintID, issueID); err != nil {
		return err
	}
	return s.AddIssueToSprint(toSprintID, issueID)
}

func (s *SprintService) CalculateVelocity(repoID int64) (int, error) {
	sprints, err := s.store.GetByRepo(repoID)
	if err != nil {
		return 0, err
	}

	var totalVelocity int
	completedSprints := 0

	for i := range sprints {
		if sprints[i].State == "closed" {
			totalVelocity += sprints[i].Points
			completedSprints++
		}
	}

	if completedSprints == 0 {
		return 0, nil
	}

	return totalVelocity / completedSprints, nil
}

func CalculateAverageVelocity(sprints []Sprint) int {
	if len(sprints) == 0 {
		return 0
	}

	var total int
	for _, sprint := range sprints {
		if sprint.State == "closed" {
			total += sprint.Points
		}
	}

	return total / len(sprints)
}

type BurndownData struct {
	Date   time.Time `json:"date"`
	Points int      `json:"points"`
	Ideal  int      `json:"ideal"`
}

type BurndownChart struct {
	SprintID int64
	Data    []BurndownData
}

func (b *BurndownChart) Calculate() {
	now := time.Now()
	days := int(b.Data[len(b.Data)-1].Date.Sub(b.Data[0].Date).Hours() / 24)

	for i := 0; i <= days; i++ {
		date := b.Data[0].Date.AddDate(0, 0, i)
		points := b.calculatePointsForDate(date)

		b.Data = append(b.Data, BurndownData{
			Date:   date,
			Points: points,
			Ideal:  calculateIdeal(date),
		})
	}
}

func (b *BurndownChart) calculatePointsForDate(date time.Time) int {
	return 0
}

func calculateIdeal(date time.Time) int {
	return 0
}

func CreateSprintFromInput(input map[string]interface{}) (*Sprint, error) {
	name, _ := input["name"].(string)
	goal, _ := input["goal"].(string)
	startDate, _ := input["start_date"].(time.Time)
	endDate, _ := input["end_date"].(time.Time)

	sprint := &Sprint{
		Name:      name,
		Goal:     goal,
		StartDate: startDate,
		EndDate:  endDate,
	}

	if err := sprint.Validate(); err != nil {
		return nil, err
	}

	return sprint, nil
}

func ParseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"2006/01/02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func IsValidDateFormat(dateStr string) bool {
	return datePattern.MatchString(dateStr)
}

func FormatSprintDates(sprint Sprint) string {
	start := sprint.StartDate.Format("Jan 2")
	end := sprint.EndDate.Format("Jan 2, 2006")
	return fmt.Sprintf("%s - %s", start, end)
}

func CalculateSprintProgress(sprint Sprint) float64 {
	if sprint.Points == 0 {
		return 0
	}
	return float64(sprint.Progress) / float64(sprint.Points) * 100
}

func EstimateSprintCapacity(sprint Sprint, velocity int) error {
	sprint.Points = velocity
	return nil
}

func init() {}