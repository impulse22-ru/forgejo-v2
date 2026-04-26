package review

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"
)

var (
	ErrRuleNotFound = errors.New("rule not found")
	ErrInvalidRule   = errors.New("invalid rule")
)

type AssignmentRule struct {
	Name      string        `json:"name"`
	Condition AssignmentCondition `json:"condition"`
	Reviewers []ReviewerSpec `json:"reviewers"`
	Options  AssignmentOptions `json:"options"`
	Enabled  bool         `json:"enabled"`
	RepoID   int64        `json:"repo_id"`
}

func (r *AssignmentRule) String() string {
	return fmt.Sprintf("Rule{name=%s, enabled=%t}", r.Name, r.Enabled)
}

func (r *AssignmentRule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Reviewers) == 0 {
		return fmt.Errorf("at least one reviewer is required")
	}
	return r.Condition.Validate()
}

type AssignmentCondition struct {
	ChangedFiles   []string `json:"changed_files"`
	ChangedDirs  []string `json:"changed_dirs"`
	Extensions  []string `json:"extensions"`
	Additions   int     `json:"additions"`
	Deletions   int     `json:"deletions"`
	HasLabel   string   `json:"has_label"`
	LacksLabel  string   `json:"lacks_label"`
	HasFile    string   `json:"has_file"`
	Regex     string   `json:"regex"`
	Author    string   `json:"author"`
}

func (c *AssignmentCondition) Validate() error {
	if c.Regex != "" {
		_, err := regexp.Compile(c.Regex)
		if err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	}
	return nil
}

func (c *AssignmentCondition) Matches(ctx context.Context, pr *PRSummary) bool {
	if c.Author != "" && c.Author != pr.Author {
		return false
	}

	if len(c.ChangedFiles) > 0 {
		matched := false
		for _, f := range c.ChangedFiles {
			for _, changed := range pr.ChangedFiles {
				if matched, _ = regexp.MatchString(f, changed); matched {
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(c.Extensions) > 0 {
		matched := false
		for _, ext := range c.Extensions {
			for _, f := range pr.ChangedFiles {
				if strings.HasSuffix(f, ext) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	if c.Additions > 0 && pr.Additions < c.Additions {
		return false
	}

	if c.Deletions > 0 && pr.Deletions < c.Deletions {
		return false
	}

	if c.HasLabel != "" && !pr.HasLabel(c.HasLabel) {
		return false
	}

	if c.LacksLabel != "" && pr.HasLabel(c.LacksLabel) {
		return false
	}

	if c.Regex != "" {
		matched := false
		for _, f := range pr.ChangedFiles {
			if matched, _ = regexp.MatchString(c.Regex, f); matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

type PRSummary struct {
	Changes []string
	Additions int
	Deletions int
	Labels  []string
	Author  string
}

func (p *PRSummary) HasLabel(label string) bool {
	for _, l := range p.Labels {
		if l == label {
			return true
		}
	}
	return false
}

type ReviewerSpec struct {
	User     string `json:"user"`
	Team     string `json:"team"`
	Required bool   `json:"required"`
}

func (s *ReviewerSpec) String() string {
	if s.Team != "" {
		return "@" + s.Team
	}
	return "@" + s.User
}

type AssignmentOptions struct {
	Count               int      `json:"count"`
	RemoveExisting     bool     `json:"remove_existing"`
	Notification      string   `json:"notification"`
	ReviewRequestType  string   `json:"review_request_type"`
}

func (o *AssignmentOptions) Defaults() {
	if o.Count == 0 {
		o.Count = 1
	}
	if o.Notification == "" {
		o.Notification = "reviewer"
	}
	if o.ReviewRequestType == "" {
		o.ReviewRequestType = "reviewer"
	}
}

type RuleEngine struct {
	rules map[int64][]AssignmentRule
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: make(map[int64][]AssignmentRule),
	}
}

func (e *RuleEngine) AddRule(repoID int64, rule AssignmentRule) error {
	rule.RepoID = repoID
	if err := rule.Validate(); err != nil {
		return err
	}

	e.rules[repoID] = append(e.rules[repoID], rule)
	return nil
}

func (e *RuleEngine) RemoveRule(repoID int64, name string) error {
	rules, ok := e.rules[repoID]
	if !ok {
		return ErrRuleNotFound
	}

	for i, rule := range rules {
		if rule.Name == name {
			e.rules[repoID] = append(rules[:i], rules[i+1:]...)
			return nil
		}
	}

	return ErrRuleNotFound
}

func (e *RuleEngine) GetRules(repoID int64) []AssignmentRule {
	return e.rules[repoID]
}

func (e *RuleEngine) FindReviewers(repoID int64, pr *PRSummary) []ReviewerSpec {
	rules := e.rules[repoID]
	var reviewers []ReviewerSpec

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if rule.Condition.Matches(context.Background(), pr) {
			reviewers = append(reviewers, rule.Reviewers...)
		}
	}

	return deduplicateReviewers(reviewers)
}

func deduplicateReviewers(specs []ReviewerSpec) []ReviewerSpec {
	seen := make(map[string]bool)
	var unique []ReviewerSpec

	for _, spec := range specs {
		key := spec.User + "/" + spec.Team
		if !seen[key] {
			seen[key] = true
			unique = append(unique, spec)
		}
	}

	return unique
}

type Pool struct {
	users  []*User
	rounds int
}

func NewPool(users ...*User) *Pool {
	return &Pool{
		users:  users,
		rounds: 0,
	}
}

func (p *Pool) Next() *User {
	if len(p.users) == 0 {
		return nil
	}
	user := p.users[p.rounds%len(p.users)]
	p.rounds++
	return user
}

func (p *Pool) Size() int {
	return len(p.users)
}

type User struct {
	ID      int64
	Login   string
	TeamIDs []int64
}

type PoolStore struct {
	pools map[string]*Pool
}

func NewPoolStore() *PoolStore {
	return &PoolStore{
		pools: make(map[string]*Pool),
	}
}

func (s *PoolStore) AddUser(team string, user *User) {
	if _, ok := s.pools[team]; !ok {
		s.pools[team] = NewPool()
	}
	s.pools[team].users = append(s.pools[team].users, user)
}

func (s *PoolStore) GetPool(team string) *Pool {
	return s.pools[team]
}

func ParseRulesFile(path string) ([]AssignmentRule, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rules []AssignmentRule
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		rule, err := ParseRuleLine(line, lineNum)
		if err != nil {
			return nil, err
		}
		if rule != nil {
			rules = append(rules, *rule)
		}
	}

	return rules, scanner.Err()
}

func ParseRuleLine(line string, lineNum int) (*AssignmentRule, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, nil
	}

	rule := &AssignmentRule{
		Enabled: true,
	}

	switch fields[0] {
	case "rule":
		rule.Name = fields[1]
	case "- rule":
		rule.Name = strings.TrimPrefix(fields[1], "-")
		rule.Enabled = false
	default:
		return nil, nil
	}

	var reviewers []ReviewerSpec
	for _, field := range fields[2:] {
		reviewers = append(reviewers, ReviewerSpec{User: field})
	}
	rule.Reviewers = reviewers

	return rule, nil
}

type RotationStrategy string

const (
	RotationRandom      RotationStrategy = "random"
	RotationRoundRobin RotationStrategy = "round-robin"
	RotationLoadBalanced RotationStrategy = "load-balanced"
)

func (r *RuleEngine) AssignReviewers(ctx context.Context, repoID int64, pr *PRSummary) ([]int64, error) {
	specs := r.FindReviewers(repoID, pr)
	var userIDs []int64
	seenUsers := make(map[int64]bool)

	for _, spec := range specs {
		if spec.User != "" {
			userID, err := findUserByLogin(spec.User)
			if err == nil && !seenUsers[userID] {
				seenUsers[userID] = true
				userIDs = append(userIDs, userID)
			}
		}

		if spec.Team != "" {
			memberIDs, err := findTeamMembers(spec.Team)
			if err == nil {
				for _, userID := range memberIDs {
					if !seenUsers[userID] {
						seenUsers[userID] = true
						userIDs = append(userIDs, userID)
					}
				}
			}
		}

		if len(userIDs) >= 1 {
			break
		}
	}

	return userIDs, nil
}

func findUserByLogin(login string) (int64, error) { return 0, nil }
func findTeamMembers(team string) ([]int64, error) { return nil, nil }

func FindReviewAssignmentFile(repoPath string) (string, error) {
	patterns := []string{
		filepath.Join(repoPath, ".github", "review-agents.yml"),
		filepath.Join(repoPath, ".github", "review-agents.yaml"),
		filepath.Join(repoPath, ".forgejo", "review-agents.yml"),
	}

	for _, pattern := range patterns {
		if _, err := os.Stat(pattern); err == nil {
			return pattern, nil
		}
	}

	return "", os.ErrNotExist
}

func LoadReviewAssignmentRules(repoID int64, repoPath string) ([]AssignmentRule, error) {
	path, err := FindReviewAssignmentFile(repoPath)
	if err != nil {
		return nil, err
	}

	return ParseRulesFile(path)
}

func init() {}