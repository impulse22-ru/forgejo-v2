package codeowners

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"
)

var (
	ErrInvalidCodeOwner = errors.New("invalid code owner entry")
	ErrFileNotFound    = errors.New("codeowners file not found")
)

type Entry struct {
	Pattern    string   `json:"pattern"`
	Owners    []string `json:"owners"`
	LineNum   int      `json:"line_num"`
	Line      string   `json:"line"`
	IsGlob    bool     `json:"is_glob"`
	IsNegated bool     `json:"is_negated"`
	Sections  []Entry  `json:"sections"`
}

func (e *Entry) String() string {
	return fmt.Sprintf("%s %s", e.Pattern, strings.Join(e.Owners, ", "))
}

func (e *Entry) Match(path string) bool {
	pattern := e.Pattern
	if e.IsNegated {
		pattern = pattern[1:]
	}

	regex, err := GlobToRegex(pattern)
	if err != nil {
		return false
	}

	return regex.MatchString(path)
}

func (e *Entry) IsSection() bool {
	return len(e.Sections) > 0
}

type CodeOwner struct {
	Login    string `json:"login"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	TeamName string `json:"team_name,omitempty"`
	Email   string `json:"email,omitempty"`
}

func (c *CodeOwner) String() string {
	if c.TeamName != "" {
		return "@" + c.TeamName
	}
	return "@" + c.Login
}

func ParseEntry(line string, lineNum int) (*Entry, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, ErrInvalidCodeOwner
	}

	pattern := fields[0]
	owners := fields[1:]
	isGlob := strings.ContainsAny(pattern, "*?[")
	isNegated := strings.HasPrefix(pattern, "!")

	if isNegated {
		pattern = pattern[1:]
	}

	entry := &Entry{
		Pattern:  pattern,
		Owners:   owners,
		LineNum:  lineNum,
		Line:    line,
		IsGlob:  isGlob,
		IsNegated: isNegated,
	}

	return entry, nil
}

func GlobToRegex(pattern string) (*regexp.Regexp, error) {
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "**/", "(.*/)?")
	pattern = strings.ReplaceAll(pattern, "**", ".*")
	pattern = strings.ReplaceAll(pattern, "*", "[^/]*")
	pattern = strings.ReplaceAll(pattern, "?", ".")
	pattern = "^" + pattern + "$"

	return regexp.Compile(pattern)
}

type File struct {
	Entries []Entry
	Path    string
	RepoID  int64
}

func (f *File) Parse() error {
	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		entry, err := ParseEntry(scanner.Text(), lineNum)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}
		if entry != nil {
			f.Entries = append(f.Entries, *entry)
		}
	}

	return scanner.Err()
}

func (f *File) FindOwners(path string) []CodeOwner {
	var owners []CodeOwner

	path = strings.TrimPrefix(path, "/")

	for _, entry := range f.Entries {
		if !entry.Match(path) {
			continue
		}

		for _, owner := range entry.Owners {
			owners = appendIfMissing(owners, parseOwner(owner))
		}
	}

	return owners
}

func parseOwner(owner string) CodeOwner {
	owner = strings.TrimPrefix(owner, "@")

	if strings.Contains(owner, "/") {
		parts := strings.SplitN(owner, "/", 2)
		return CodeOwner{
			Type:    "team",
			TeamName: owner,
			Name:    parts[1],
		}
	}

	return CodeOwner{
		Type:  "user",
		Login: owner,
	}
}

func appendIfMissing(owners []CodeOwner, newOwner CodeOwner) []CodeOwner {
	for _, owner := range owners {
		if owner.Login == newOwner.Login && owner.TeamName == newOwner.TeamName {
			return owners
		}
	}
	return append(owners, newOwner)
}

type RuleEngine struct {
	files map[string]*File
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		files: make(map[string]*File),
	}
}

func (r *RuleEngine) Load(repoID int64, path string) error {
	codeownersPath := filepath.Join(path, "CODEOWNERS")
	if _, err := os.Stat(codeownersPath); os.IsNotExist(err) {
		codeownersPath = filepath.Join(path, ".github", "CODEOWNERS")
		if _, err := os.Stat(codeownersPath); os.IsNotExist(err) {
			return ErrFileNotFound
		}
	}

	file := &File{
		Path:   codeownersPath,
		RepoID: repoID,
	}

	if err := file.Parse(); err != nil {
		return err
	}

	r.files[fmt.Sprintf("%d", repoID)] = file
	return nil
}

func (r *RuleEngine) GetOwners(repoID int64, path string) []CodeOwner {
	file, ok := r.files[fmt.Sprintf("%d", repoID)]
	if !ok {
		return nil
	}
	return file.FindOwners(path)
}

func (r *RuleEngine) HasCodeOwners(repoID int64) bool {
	_, ok := r.files[fmt.Sprintf("%d", repoID)]
	return ok
}

type Rule struct {
	RequiredReviewers int  `json:"required_reviewers"`
	Blockers        int  `json:"blocking_reviewers"`
	AutoRequest    bool `json:"auto_request"`
	RequiredTypes []string `json:"required_types"`
}

func ParseRule(lines []string) (*Rule, error) {
	rule := &Rule{
		RequiredReviewers: 1,
		AutoRequest:     true,
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "#")
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "required-reviewers":
			rule.RequiredReviewers, _ = strconv.Atoi(value)
		case "blocking-reviewers":
			rule.Blockers, _ = strconv.Atoi(value)
		case "auto-request":
			rule.AutoRequest = value == "true"
		case "required-types":
			rule.RequiredTypes = strings.Split(value, ",")
		}
	}

	return rule, nil
}

func (r *Rule) Validate() error {
	if r.RequiredReviewers < 0 {
		return fmt.Errorf("required reviewers must be non-negative")
	}
	if r.Blockers < 0 {
		return fmt.Errorf("blocking reviewers must be non-negative")
	}
	if r.Blockers > r.RequiredReviewers {
		return fmt.Errorf("blocking reviewers cannot exceed required reviewers")
	}
	return nil
}

type ReviewCheck struct {
	RequiredApprovals []CodeOwner
	BlockApprovals   []CodeOwner
	Pending        []CodeOwner
	Approved       []CodeOwner
	ChangesRequired []CodeOwner
}

func (c *ReviewCheck) IsApproved() bool {
	return len(c.Approved) >= 1 && len(c.ChangesRequired) == 0
}

func (c *ReviewCheck) NeedsMoreReviewers() bool {
	return len(c.Approved) < 1
}

func FindCodeownersFile(repoPath string) (string, error) {
	patterns := []string{
		filepath.Join(repoPath, "CODEOWNERS"),
		filepath.Join(repoPath, ".github", "CODEOWNERS"),
		filepath.Join(repoPath, "docs", "CODEOWNERS"),
	}

	for _, pattern := range patterns {
		if _, err := os.Stat(pattern); err == nil {
			return pattern, nil
		}
	}

	return "", ErrFileNotFound
}

func ValidateOwner(owner string) error {
	if owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}

	owner = strings.TrimPrefix(owner, "@")
	if owner == "" {
		return fmt.Errorf("invalid owner format")
	}

	parts := strings.Split(owner, "/")
	if len(parts) == 2 {
		if !isValidTeamName(parts[1]) {
			return fmt.Errorf("invalid team name: %s", parts[1])
		}
	} else if len(parts) == 1 {
		if !isValidUsername(owner) {
			return fmt.Errorf("invalid username: %s", owner)
		}
	}

	return nil
}

func isValidUsername(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9\-]*$`, name)
	return matched
}

func isValidTeamName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9\-]*$`, name)
	return matched
}

type CodeOwnerEntry struct {
	Pattern string
	Owners  []string
}

func ParseCodeOwnersFile(content string) ([]CodeOwnerEntry, error) {
	var entries []CodeOwnerEntry
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		entry, err := ParseEntry(line, lineNum)
		if err != nil {
			return nil, err
		}
		if entry != nil {
			entries = append(entries, CodeOwnerEntry{
				Pattern: entry.Pattern,
				Owners:  entry.Owners,
			})
		}
	}

	return entries, scanner.Err()
}

func SortBySpecificity(entries []CodeOwnerEntry) {
	sort.Slice(entries, func(i, j int) bool {
		pri := specificity(entries[i].Pattern)
		prij := specificity(entries[j].Pattern)
		return pri > prij
	})
}

func specificity(pattern string) int {
	specific := 0
	for _, c := range pattern {
		switch c {
		case '*':
			specific += 1
		case '?':
			specific += 2
		case '/':
			specific += 10
		default:
			specific += 1
		}
	}
	return specific
}

func init() {}