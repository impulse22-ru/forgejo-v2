package pr

import (
	"time"
)

type PullRequest struct {
	ID          int64
	Index       int64
	Title       string
	Body        string
	State       string
	HeadBranch  string
	BaseBranch  string
	IsDraft     bool
	IsAutoMerge bool
	Author     int64
	Created    time.Time
	Updated    time.Time
	Merged     *time.Time
}

type Comment struct {
	ID            int64
	PullRequestID  int64
	CommitSHA     string
	Path          string
	LineStart     int
	LineEnd       int
	ParentID      int64
	Body          string
	AuthorID      int64
	Resolved      bool
	ResolvedBy    int64
	ResolvedAt    *time.Time
}

func NewPullRequest() *PullRequest {
	return &PullRequest{
		State: "open",
	}
}

func (p *PullRequest) SetDraft(draft bool) {
	p.IsDraft = draft
}

func (p *PullRequest) MarkReady() {
	p.IsDraft = false
}

func (p *PullRequest) CanMerge() bool {
	if p.IsDraft {
		return false
	}
	if p.State != "open" {
		return false
	}
	return true
}

func (c *Comment) IsInline() bool {
	return c.Path != "" && c.LineStart > 0
}

func (c *Comment) Resolve() {
	resolvedAt := time.Now()
	c.Resolved = true
	c.ResolvedAt = &resolvedAt
}