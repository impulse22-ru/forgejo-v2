package review

import (
	"sort"
	"time"
)

type Reviewer int64

type Review struct {
	ID           int64
	PullRequestID int64
	ReviewerID   int64
	CommitSHA   string
	Body        string
	State       ReviewState
	Created     time.Time
	Updated    time.Time
}

type ReviewState string

const (
	ReviewStatePending  ReviewState = "PENDING"
	ReviewStateApproved ReviewState = "APPROVED"
	ReviewStateChanges Requested ReviewState = "CHANGES_REQUESTED"
	ReviewStateCommented ReviewState = "COMMENTED"
	ReviewStateDismissed ReviewState = "DISMISSED"
)

type ReviewAssignment struct {
	Name     string
	Condition ReviewCondition
	Reviewers []Reviewer
}

type ReviewCondition struct {
	ChangedFiles []string
	Additions  int
	Patterns  []string
}

type RuleEngine struct {
	rules    []ReviewAssignment
	registry *ReviewerRegistry
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules:    make([]ReviewAssignment, 0),
		registry: NewReviewerRegistry(),
	}
}

func (e *RuleEngine) AddRule(rule ReviewAssignment) {
	e.rules = append(e.rules, rule)
}

func (e *RuleEngine) FindReviewers(condition ReviewCondition) []Reviewer {
	var reviewers []Reviewer

	for _, rule := range e.rules {
		if e.matchCondition(condition, rule.Condition) {
			reviewers = append(reviewers, rule.Reviewers...)
		}
	}

	return deduplicateReviewers(reviewers)
}

func (e *RuleEngine) matchCondition(c, rule ReviewCondition) bool {
	if len(rule.ChangedFiles) > 0 {
		for _, f := range rule.ChangedFiles {
			if contains(c.ChangedFiles, f) {
				return true
			}
		}
	}

	if rule.Additions > 0 && c.Additions > rule.Additions {
		return true
	}

	return false
}

func deduplicateReviewers(reviewers []Reviewer) []Reviewer {
	seen := make(map[Reviewer]bool)
	var result []Reviewer

	for _, r := range reviewers {
		if !seen[r] {
			seen[r] = true
			result = append(result, r)
		}
	}

	return result
}

type ReviewerRegistry struct {
	reviewers map[Reviewer]string
}

func NewReviewerRegistry() *ReviewerRegistry {
	return &ReviewerRegistry{
		reviewers: make(map[Reviewer]string),
	}
}

func (r *ReviewerRegistry) Add(reviewer Reviewer, name string) {
	r.reviewers[reviewer] = name
}

func (r *ReviewerRegistry) GetName(reviewer Reviewer) string {
	return r.reviewers[reviewer]
}

type ReviewerPool struct {
	reviewers []Reviewer
	current int
}

func NewReviewerPool(reviewers ...Reviewer) *ReviewerPool {
	sort.Slice(reviewers, func(i, j int) bool {
		return reviewers[i] < reviewers[j]
	})
	return &ReviewerPool{
		reviewers: reviewers,
	}
}

func (p *ReviewerPool) Next() Reviewer {
	if len(p.reviewers) == 0 {
		return 0
	}
	reviewer := p.reviewers[p.current%len(p.reviewers)]
	p.current++
	return reviewer
}

func (p *ReviewerPool) Size() int {
	return len(p.reviewers)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}