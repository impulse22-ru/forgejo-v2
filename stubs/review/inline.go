package pr

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
	ErrCommentNotFound = errors.New("comment not found")
	ErrInvalidLine     = errors.New("invalid line number")
)

type ReviewComment struct {
	ID             int64     `json:"id" gorm:"primaryKey"`
	ReviewID       int64     `json:"review_id" gorm:"index"`
	IssueID        int64     `json:"issue_id" gorm:"index"`
	CommitID       string    `json:"commit_id" gorm:"index"`
	Path           string    `json:"path"`
	Line           int       `json:"line"`
	OriginalLine   int       `json:"original_line"`
	DiffHunk       string    `json:"diff_hunk"`
	Content        string    `json:"content"`
	Suggestion     string    `json:"suggestion,omitempty"`
	ParentID      int64     `json:"parent_id" gorm:"index"`
	UserID        int64     `json:"user_id" gorm:"index"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	ResolvedByID  *int64    `json:"resolved_by_id"`
}

func (c *ReviewComment) String() string {
	return fmt.Sprintf("ReviewComment{id=%d, path=%s, line=%d}", c.ID, c.Path, c.Line)
}

func (c *ReviewComment) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("path is required")
	}
	if c.Line < 0 {
		return ErrInvalidLine
	}
	return nil
}

func (c *ReviewComment) IsResolved() bool {
	return c.ResolvedAt != nil
}

func (c *ReviewComment) Resolve(userID int64) {
	now := time.Now()
	c.ResolvedAt = &now
	c.ResolvedByID = &userID
}

func (c *ReviewComment) Unresolve() {
	c.ResolvedAt = nil
	c.ResolvedByID = nil
}

func (c *ReviewComment) ToLegacyComment() *models.Comment {
	return &models.Comment{
		ID:        c.ID,
		Type:      models.CommentTypeReview,
		IssueID:   c.IssueID,
		Content:   c.Path + ":" + strconv.Itoa(c.Line),
		Body:      c.Content,
		ParentID:  c.ParentID,
		CreatedAt: c.CreatedAt,
	}
}

type ReviewCommentList []ReviewComment

func (l ReviewCommentList) GetDiff(hunk, path string, line int) *ReviewComment {
	for _, c := range l {
		if c.Path == path && c.Line == line {
			return &c
		}
	}
	return nil
}

func (l ReviewCommentList) FilterByCommit(commitID string) ReviewCommentList {
	var filtered ReviewCommentList
	for _, c := range l {
		if c.CommitID == commitID {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (l ReviewCommentList) FilterResolved(resolved bool) ReviewCommentList {
	var filtered ReviewCommentList
	for _, c := range l {
		if c.IsResolved() == resolved {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

type DiffViewer struct {
	comments map[string]map[int][]*ReviewComment
}

func NewDiffViewer() *DiffViewer {
	return &DiffViewer{
		comments: make(map[string]map[int][]*ReviewComment),
	}
}

func (d *DiffViewer) AddComment(path string, line int, comment *ReviewComment) {
	if _, ok := d.comments[path]; !ok {
		d.comments[path] = make(map[int][]*ReviewComment)
	}
	d.comments[path][line] = append(d.comments[path][line], comment)
}

func (d *DiffViewer) GetComments(path string, line int) []*ReviewComment {
	if _, ok := d.comments[path]; !ok {
		return nil
	}
	return d.comments[path][line]
}

func (d *DiffViewer) GetCommentsInRange(path string, start, end int) []*ReviewComment {
	var comments []*ReviewComment
	for _, line := range d.comments[path] {
		if line >= start && line <= end {
			comments = append(comments, line...)
		}
	}
	return comments
}

func (d *DiffViewer) Marshal() string {
	var lines []string
	for path, lineComments := range d.comments {
		for line, comments := range lineComments {
			for _, c := range comments {
				lines = append(lines, fmt.Sprintf("%s:%d: %s", path, line, c.Content))
			}
		}
	}
	return strings.Join(lines, "\n")
}

type DiffParser struct {
	commitID    string
	oldCommitID string
	oldContent string
	newContent string
}

func ParseDiff(raw string) (*DiffParser, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "diff --git") {
		return nil, fmt.Errorf("not a valid diff")
	}

	return &DiffParser{
		newContent: raw,
	}, nil
}

func (p *DiffParser) ParseHunk(hunk string) (oldStart, oldCount, newStart, newCount int, err error) {
	hunkLine := strings.TrimSpace(hunk)
	if !strings.HasPrefix(hunkLine, "@@") {
		return 0, 0, 0, 0, fmt.Errorf("invalid hunk header")
	}

	parts := strings.Fields(hunkLine[4:])
	if len(parts) < 2 {
		return 0, 0, 0, 0, fmt.Errorf("incomplete hunk header")
	}

	oldSpec := parts[0]
	newSpec := parts[1]

	if n, err := fmt.Sscanf(oldSpec, "-%d,%d", &oldStart, &oldCount); err != nil {
		if n, err = fmt.Sscanf(oldSpec, "-%d", &oldStart); err != nil {
			return 0, 0, 0, 0, err
		}
		oldCount = 1
	}

	if n, err := fmt.Sscanf(newSpec, "+%d,%d", &newStart, &newCount); err != nil {
		if n, err = fmt.Sscanf(newSpec, "+%d", &newStart); err != nil {
			return 0, 0, 0, 0, err
		}
		newCount = 1
	}

	return oldStart, oldCount, newStart, newCount, nil
}

func ParseLineNumber(line string) int {
	line = strings.TrimLeft(line, "+-")
	line = strings.TrimSpace(line)
	if line == "" {
		return 0
	}

	num, err := strconv.Atoi(line)
	if err != nil {
		return 0
	}
	return num
}

func BuildDiffHunk(oldStart, oldCount, newStart, newCount int) string {
	return fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount)
}

type CodeSuggestion struct {
	Suggestion  string `json:"suggestion"`
	Path       string `json:"path"`
	Line       int    `json:"line"`
	Confidence float32 `json:"confidence"`
}

func (s *CodeSuggestion) Apply(content string) string {
	lines := strings.Split(content, "\n")
	if s.Line > 0 && s.Line <= len(lines) {
		lines[s.Line-1] = s.Suggestion
	}
	return strings.Join(lines, "\n")
}

func ParseSuggestion(suggestion string) (*CodeSuggestion, error) {
	parts := strings.SplitN(suggestion, "\n", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid suggestion format")
	}

	return &CodeSuggestion{
		Suggestion:  strings.TrimSpace(parts[1]),
		Confidence: 1.0,
	}, nil
}

type CommentThread struct {
	comments []*ReviewComment
	resolved bool
}

func NewCommentThread() *CommentThread {
	return &CommentThread{
		comments: make([]*ReviewComment, 0),
		comments: make([]*ReviewComment, 0),
	}
}

func (t *CommentThread) Add(comment *ReviewComment) {
	t.comments = append(t.comments, comment)
	t.resolved = comment.IsResolved()
}

func (t *CommentThread) Resolve(userID int64) {
	for _, c := range t.comments {
		if !c.IsResolved() {
			c.Resolve(userID)
		}
	}
	t.resolved = true
}

func (t *CommentThread) Unresolve() {
	for _, c := range t.comments {
		c.Unresolve()
	}
	t.resolved = false
}

func (t *CommentThread) First() *ReviewComment {
	if len(t.comments) > 0 {
		return t.comments[0]
	}
	return nil
}

func (t *CommentThread) Last() *ReviewComment {
	if len(t.comments) > 0 {
		return t.comments[len(t.comments)-1]
	}
	return nil
}

func inlineCommentPattern(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\.]+$`, name)
	return matched
}

var wellKnownComments = map[string]int{
	"CONTRIBUTING.md":  50,
	"LICENSE":         50,
	"README.md":      100,
	"CODEOWNERS":     50,
}

func GetSuggestedCommentCount(filename string) int {
	for name, count := range wellKnownComments {
		if strings.HasSuffix(filename, name) {
			return count
		}
	}
	return 10
}

func init() {}