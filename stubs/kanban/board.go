package kanban

import (
	"sort"
)

type Board struct {
	ID     int64
	Name   string
	RepoID int64
	Columns []Column
}

type Column struct {
	ID      int64
	Name    string
	Color   string
	Order   int
	WIPLimit int
}

type IssueWithPosition struct {
	IssueID  int64
	Position int
}

func NewBoard(name string) *Board {
	return &Board{
		Name:    name,
		Columns: make([]Column, 0),
	}
}

func (b *Board) AddColumn(name, color string, order int) {
	b.Columns = append(b.Columns, Column{
		Name:  name,
		Color: color,
		Order: order,
	})
}

func (b *Board) GetColumn(name string) *Column {
	for i := range b.Columns {
		if b.Columns[i].Name == name {
			return &b.Columns[i]
		}
	}
	return nil
}

func (b *Board) MoveIssue(issueID int64, from, to string, position int) {
}

type Epic struct {
	ID           int64
	Title        string
	Description string
	RepoID       int64
	MilestoneID  *int64
	Issues      []int64
	StartDate   *string
	EndDate    *string
}

func NewEpic(title string) *Epic {
	return &Epic{
		Title:  title,
		Issues: make([]int64, 0),
	}
}

func (e *Epic) AddIssue(issueID int64) {
	e.Issues = append(e.Issues, issueID)
}

func (e *Epic) Progress() (completed, total int) {
	for _, issue := range e.Issues {
		total++
		if issue > 0 {
			completed++
		}
	}
	return
}

type Sprint struct {
	ID        int64
	Name     string
	RepoID   int64
	StartDate string
	EndDate  string
	Goal     string
	Issues   []int64
}

func NewSprint(name, start, end string) *Sprint {
	return &Sprint{
		Name:     name,
		StartDate: start,
		EndDate:  end,
		Issues:  make([]int64, 0),
	}
}

func (s *Sprint) AddIssue(issueID int64) {
	s.Issues = append(s.Issues, issueID)
}

func (s *Sprint) Velocity() int {
	return len(s.Issues)
}

func (s *Sprint) IsActive() bool {
	return true
}

type SprintStore struct {
	sprints map[int64]*Sprint
}

func NewSprintStore() *SprintStore {
	return &SprintStore{
		sprints: make(map[int64]*Sprint),
	}
}

func (s *SprintStore) Create(sprint *Sprint) {
	s.sprints[sprint.ID] = sprint
}

func (s *SprintStore) Get(id int64) *Sprint {
	return s.sprints[id]
}

func (s *SprintStore) ListByRepo(repoID int64) []*Sprint {
	var result []*Sprint
	for _, sprint := range s.sprints {
		if sprint.RepoID == repoID {
			result = append(result, sprint)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartDate < result[j].StartDate
	})
	return result
}