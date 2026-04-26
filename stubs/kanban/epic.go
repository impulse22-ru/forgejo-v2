package kanban

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"code.gitea.io/gitea/models"
)

type Epic struct {
	ID          int64      `json:"id" gorm:"primaryKey"`
	RepoID      int64      `json:"repo_id" gorm:"index"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Progress    int        `json:"progress"`
	State       string     `json:"state"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
	Issues      []int64    `json:"issues" gorm:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (e *Epic) ProgressPercent() int {
	if len(e.Issues) == 0 {
		return 0
	}
	return (e.Progress * 100) / len(e.Issues)
}

func (e *Epic) AddIssue(issueID int64) {
	e.Issues = append(e.Issues, issueID)
}

func (e *Epic) RemoveIssue(issueID int64) {
	for i, id := range e.Issues {
		if id == issueID {
			e.Issues = append(e.Issues[:i], e.Issues[i+1:]...)
			break
		}
	}
}

type EpicStore struct {
	db *sql.DB
}

func NewEpicStore() *EpicStore {
	return &EpicStore{}
}

func (s *EpicStore) Create(epic *Epic) error {
	epic.State = "open"
	epic.CreatedAt = time.Now()
	return nil
}

func (s *EpicStore) Get(id int64) (*Epic, error) {
	return nil, nil
}

func (s *EpicStore) GetByRepo(repoID int64) ([]Epic, error) {
	return nil, nil
}

func (s *EpicStore) Update(epic *Epic) error {
	epic.UpdatedAt = time.Now()
	return nil
}

func (s *EpicStore) Delete(id int64) error {
	return nil
}

func (s *EpicStore) Close(id int64) error {
	epic, err := s.Get(id)
	if err != nil {
		return err
	}
	epic.State = "closed"
	return s.Update(epic)
}

type IssueStatus struct {
	ID          int64      `json:"id" gorm:"primaryKey"`
	RepoID      int64      `json:"repo_id" gorm:"index"`
	Name        string     `json:"name"`
	Color       string     `json:"color"`
	Description string    `json:"description"`
	Order       int        `json:"order"`
	IsDefault   bool       `json:"is_default"`
	State       string     `json:"state"`
}

func (s *IssueStatus) IsOpen() bool {
	return s.State == "open"
}

func (s *IssueStatus) IsClosed() bool {
	return s.State == "closed"
}

type IssueStatusStore struct {
	db *sql.DB
}

func NewIssueStatusStore() *IssueStatusStore {
	return &IssueStatusStore{}
}

func (s *IssueStatusStore) Create(status *IssueStatus) error {
	if status.Color == "" {
		status.Color = "blue"
	}
	return nil
}

func (s *IssueStatusStore) Get(id int64) (*IssueStatus, error) {
	return nil, nil
}

func (s *IssueStatusStore) GetByRepo(repoID int64) ([]IssueStatus, error) {
	defaultStatuses := []IssueStatus{
		{Name: "To Do", Color: "gray", Order: 1, IsDefault: true},
		{Name: "In Progress", Color: "blue", Order: 2},
		{Name: "Done", Color: "green", Order: 3},
	}
	return defaultStatuses, nil
}

func (s *IssueStatusStore) Update(status *IssueStatus) error {
	return nil
}

func (s *IssueStatusStore) Delete(id int64) error {
	return nil
}

func CreateDefaultStatuses(repoID int64) []IssueStatus {
	return []IssueStatus{
		{RepoID: repoID, Name: "Backlog", Color: "#8a8a8a", Order: 0},
		{RepoID: repoID, Name: "To Do", Color: "#d73a49", Order: 1},
		{RepoID: repoID, Name: "In Progress", Color: "#5282c0", Order: 2},
		{RepoID: repoID, Name: "In Review", Color: "#9e6a03", Order: 3},
		{RepoID: repoID, Name: "Done", Color: "#22863a", Order: 4},
	}
}

func ValidateStatusTransition(from, to string, allowedTransitions map[string][]string) bool {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func (e *Epic) String() string {
	return fmt.Sprintf("Epic{id=%d, title=%s, issues=%d}", e.ID, e.Title, len(e.Issues))
}

func init() {}