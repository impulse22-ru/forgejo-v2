package pr

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"
)

var (
	ErrDraftNotFound  = errors.New("draft PR not found")
	ErrNotADraft      = errors.New("not a draft PR")
	ErrInvalidState   = errors.New("invalid state")
)

type PullRequest struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	RepoID     int64     `json:"repo_id" gorm:"index"`
	Index      int64     `json:"index" gorm:"unique(repo_id)"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Ref        string    `json:"ref"`
	BaseRef    string    `json:"base_ref"`
	HeadRepoID int64     `json:"head_repo_id" gorm:"index"`
	HeadBranch string    `json:"head_branch"`
	BaseBranch string    `json:"base_branch"`
	Draft      bool      `json:"draft" gorm:"index"`
	Merged     bool      `json:"merged"`
	MergedAt   *time.Time `json:"merged_at"`
	MergedBy  int64     `json:"merged_by"`
	State     string    `json:"state" gorm:"index"`
	IsDeleted  bool      `json:"is_deleted"`
	Updated   time.Time `json:"updated"`
	Created   time.Time `json:"created"`

	HasMerged    bool      `json:"-"`
	MergeChanged bool     `json:"-"`
}

func (p *PullRequest) String() string {
	return fmt.Sprintf("PullRequest{id=%d, title=%s, draft=%t}", p.ID, p.Title, p.Draft)
}

func (p *PullRequest) Marshal() map[string]interface{} {
	return map[string]interface{}{
		"id":          p.ID,
		"repo_id":     p.RepoID,
		"index":       p.Index,
		"title":       p.Title,
		"content":     p.Content,
		"head_branch": p.HeadBranch,
		"base_branch": p.BaseBranch,
		"draft":       p.Draft,
		"merged":      p.Merged,
		"state":       p.State,
	}
}

func (p *PullRequest) IsMergeable() bool {
	if p.Draft || p.Merged || p.State != models.PullRequestStateOpen {
		return false
	}
	return !p.MergeChanged
}

func (p *PullRequest) CanAutoMerge() bool {
	return p.State == models.PullRequestStateOpen && !p.Draft && p.IsMergeable()
}

func (p *PullRequest) SetDraft(draft bool) error {
	if p.Merged {
		return ErrInvalidState
	}
	p.Draft = draft
	p.Updated = time.Now()
	return nil
}

func (p *PullRequest) MarkReady() {
	p.Draft = false
	p.Updated = time.Now()
}

func (p *PullRequest) GetURL() string {
	return fmt.Sprintf("/%d/pulls/%d", p.RepoID, p.Index)
}

func (p *PullRequest) GetExternalURL() string {
	return fmt.Sprintf("/%s/pulls/%d", url.PathEscape(strconv.FormatInt(p.HeadRepoID, 10)), p.Index)
}

func (p *PullRequest) GetComparisonURL() string {
	return fmt.Sprintf("%s...%s", p.BaseBranch, p.HeadBranch)
}

func (p *PullRequest) GetCommitsURL() string {
	return fmt.Sprintf("/%d/pulls/%d/commits", p.RepoID, p.Index)
}

func (p *PullRequest) GetFilesURL() string {
	return fmt.Sprintf("/%d/pulls/%d/files", p.RepoID, p.Index)
}

type AutoMerge struct {
	ID            int64     `json:"id" gorm:"primaryKey"`
	PullRequestID int64     `json:"pull_request_id" gorm:"unique"`
	Doer         int64     `json:"doer"`
	MergeStyle   string    `json:"merge_style"`
	DeleteBranch bool     `json:"delete_branch"`
	Enabled     bool      `json:"enabled"`
	ScheduledAt time.Time `json:"scheduled_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (a *AutoMerge) String() string {
	return fmt.Sprintf("AutoMerge{pr_id=%d, style=%s, delete=%t}", a.PullRequestID, a.MergeStyle, a.DeleteBranch)
}

func (a *AutoMerge) ShouldMerge() bool {
	if !a.Enabled {
		return false
	}
	return true
}

func (a *AutoMerge) Enable() {
	a.Enabled = true
	a.ScheduledAt = time.Now()
}

func (a *AutoMerge) Disable() {
	a.Enabled = false
}

type MergeRequest struct {
	Doer         int64     `json:"doer"`
	HeadCommitID string   `json:"head_commit_id"`
	MergeStyle   string    `json:"merge_style"`
	DeleteBranch bool     `json:"delete_branch"`
	ForceMerge  bool     `json:"force_merge"`
}

func (r *MergeRequest) String() string {
	return fmt.Sprintf("MergeRequest{style=%s, delete=%t, force=%t}", r.MergeStyle, r.DeleteBranch, r.ForceMerge)
}

func (r *MergeRequest) Validate() error {
	if r.MergeStyle == "" {
		r.MergeStyle = "merge"
	}
	valid := map[string]bool{
		"merge":   true,
		"rebase":  true,
		"squash":  true,
		"rebase-merge": true,
	}
	if !valid[r.MergeStyle] {
		return fmt.Errorf("invalid merge style: %s", r.MergeStyle)
	}
	return nil
}

func GetMergeStyleFromString(style string) string {
	style = strings.ToLower(style)
	switch style {
	case "merge", "rebase", "squash", "rebase-merge":
		return style
	default:
		return "merge"
	}
}

type MergeResponse struct {
	SHA     string `json:"sha"`
	Merged bool   `json:"merged"`
	Message string `json:"message"`
}

func (r *MergeResponse) String() string {
	if r.Merged {
		return fmt.Sprintf("Merged as %s", r.SHA)
	}
	return r.Message
}

type PRList []PullRequest

func (l PRList) FilterDraft() PRList {
	var draft PRList
	for _, pr := range l {
		if pr.Draft {
			draft = append(draft, pr)
		}
	}
	return draft
}

func (l PRList) FilterOpen() PRList {
	var open PRList
	for _, pr := range l {
		if pr.State == models.PullRequestStateOpen {
			open = append(open, pr)
		}
	}
	return open
}

func (l PRList) FilterByBranch(branch string) PRList {
	var filtered PRList
	for _, pr := range l {
		if pr.BaseBranch == branch || pr.HeadBranch == branch {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

type PRState string

const (
	PRStateDraft   PRState = "draft"
	PRStateReady   PRState = "ready"
	PRStateWaiting PRState = "waiting"
	PRStateMerged  PRState = "merged"
)

func (p *PullRequest) GetState() PRState {
	if p.Merged {
		return PRStateMerged
	}
	if p.Draft {
		return PRStateDraft
	}
	return PRStateReady
}

type PRModifier struct {
	db *sql.DB
}

func NewPRModifier(db *sql.DB) *PRModifier {
	return &PRModifier{db: db}
}

func (m *PRModifier) SetDraft(pr *PullRequest, draft bool) error {
	pr.SetDraft(draft)
	return m.Update(pr)
}

func (m *PRModifier) Ready(pr *PullRequest) error {
	pr.MarkReady()
	return m.Update(pr)
}

func (m *PRModifier) EnableAutoMerge(pr *PullRequest, merge *MergeRequest) error {
	autoMerge := &AutoMerge{
		PullRequestID: pr.ID,
		Doer:         merge.Doer,
		MergeStyle:   merge.MergeStyle,
		DeleteBranch: merge.DeleteBranch,
		Enabled:     true,
		CreatedAt:   time.Now(),
	}
	return saveAutoMerge(autoMerge)
}

func (m *PRModifier) DisableAutoMerge(pr *PullRequest) error {
	return updateAutoMergeDisable(pr.ID)
}

func (m *PRModifier) Update(pr *PullRequest) error {
	return nil
}

func saveAutoMerge(a *AutoMerge) error { return nil }
func updateAutoMergeDisable(id int64) error { return nil }

type PullRequestService struct {
	db *sql.DB
}

func NewPullRequestService(db *sql.DB) *PullRequestService {
	return &PullRequestService{db: db}
}

func (s *PullRequestService) ListAutoMergeable(prs []PullRequest) []PullRequest {
	var autoMergeable []PullRequest
	for i := range prs {
		if prs[i].CanAutoMerge() {
			autoMergeable = append(autoMergeable, prs[i])
		}
	}
	return autoMergeable
}

func (s *PullRequestService) CheckAutoMerge(pr *PullRequest) (*MergeResponse, error) {
	if pr.IsMergeable() {
		return &MergeResponse{Merged: true, SHA: "auto-merged"}, nil
	}
	return &MergeResponse{Merged: false, Message: "not ready for auto merge"}, nil
}

func init() {}