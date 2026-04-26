package bridges

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var (
	ErrInvalidFormat  = errors.New("invalid format")
	ErrImportFailed  = errors.New("import failed")
	ErrUnsupported  = errors.New("unsupported platform")
)

type Bridge struct {
	Name     string
	Platform string
	Format   Format
	Parser   func(data []byte) (*WebhookConfig, error)
}

type Format string

const (
	FormatGitHub   Format = "github"
	FormatGitLab  Format = "gitlab"
	FormatMattermost Format = "mattermost"
	FormatJira    Format = "jira"
	FormatSlack   Format = "slack"
)

type WebhookConfig struct {
	Name      string
	URL       string
	Events    []string
	Secret   string
	Active   bool
	Config   map[string]string
}

func ImportGitHub(data []byte) (*WebhookConfig, error) {
	var config struct {
		URL    string `json:"url"`
		Events []string `json:"events"`
		Secret string `json:"secret"`
		Active bool   `json:"active"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
	}

	events := make([]string, 0)
	for _, event := range config.Events {
		events = append(events, mapGitHubEvent(event))
	}

	return &WebhookConfig{
		Name:    "github-imported",
		URL:     config.URL,
		Events:  events,
		Secret: config.Secret,
		Active: config.Active,
	}, nil
}

func ImportGitLab(data []byte) (*WebhookConfig, error) {
	var config struct {
		URL            string `json:"url"`
		Token          string `json:"token"`
		PushEvents     bool   `json:"push_events"`
		TagPushEvents  bool   `json:"tag_push_events"`
		IssuesEvents   bool   `json:"issues_events"`
		MergeRequests bool   `json:"merge_requests_events"`
		NoteEvents    bool   `json:"note_events"`
		ReleaseEvents bool   `json:"release_events"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
	}

	events := make([]string, 0)
	if config.PushEvents {
		events = append(events, "push")
	}
	if config.TagPushEvents {
		events = append(events, "tag_push")
	}
	if config.IssuesEvents {
		events = append(events, "issue")
	}
	if config.MergeRequests {
		events = append(events, "pull_request")
	}
	if config.NoteEvents {
		events = append(events, "note")
	}
	if config.ReleaseEvents {
		events = append(events, "release")
	}

	return &WebhookConfig{
		Name:    "gitlab-imported",
		URL:     config.URL,
		Events:  events,
		Secret: config.Token,
		Active: true,
	}, nil
}

func ImportMattermost(data []byte) (*WebhookConfig, error) {
	var config struct {
		URL     string `json:"url"`
		Channel string `json:"channel"`
		Token   string `json:"token"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
	}

	return &WebhookConfig{
		Name: "mattermost-imported",
		URL:  config.URL,
		Events: []string{"push", "pull_request", "issue"},
		Active: true,
		Config: map[string]string{
			"channel": config.Channel,
			"token":   config.Token,
		},
	}, nil
}

func ImportJira(data []byte) (*WebhookConfig, error) {
	var config struct {
		URL    string `json:"url"`
		Events []string `json:"events"`
		Secret string `json:"secret"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
	}

	events := make([]string, 0)
	for _, event := range config.Events {
		events = append(events, mapJiraEvent(event))
	}

	return &WebhookConfig{
		Name:    "jira-imported",
		URL:     config.URL,
		Events:  events,
		Secret: config.Secret,
		Active: true,
	}, nil
}

func ImportSlack(data []byte) (*WebhookConfig, error) {
	var config struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
	}

	return &WebhookConfig{
		Name:    "slack-imported",
		URL:     config.URL,
		Events:  []string{"push", "pull_request", "issue", "release"},
		Secret: config.Token,
		Active: true,
	}, nil
}

func mapGitHubEvent(event string) string {
	mapping := map[string]string{
		"push":               "push",
		"pull_request":       "pull_request",
		"issues":           "issue",
		"issue_comment":     "issue_comment",
		"release":          "release",
		"deployment":       "deployment",
		"deployment_status": "deployment_status",
		"check_run":        "check_run",
		"check_suite":      "check_suite",
		"commit_comment":    "commit_comment",
		"fork":             "fork",
		"gollum":           "gollum",
		"label":            "label",
		"member":           "member",
		"milestone":        "milestone",
		"page_build":       "page_build",
		"project":         "project",
		"project_card":     "project_card",
		"project_column":  "project_column",
		"public":          "public",
		"pull_request_review": "pull_request_review",
		"pull_request_review_thread": "pull_request_review_thread",
		"pull_request_thread":      "pull_request_thread",
		"push":                   "push",
		"repository":             "repository",
		"repository_import":     "repository_import",
		"repository_vulnerability_alert": "repository_vulnerability_alert",
		"security_advisoryories": "security_advisory",
		"security_alert":        "security_alert",
		"status":               "status",
		"watch":                "watch",
		"workflow_dispatch":    "workflow_dispatch",
		"workflow_run":        "workflow_run",
	}

	if mapped, ok := mapping[event]; ok {
		return mapped
	}
	return event
}

func mapJiraEvent(event string) string {
	mapping := map[string]string{
		"jira:issue_created":      "issue.created",
		"jira:issue_updated":    "issue.updated",
		"jira:issue_deleted":   "issue.deleted",
		"jira:issue_linked":    "issue.linked",
		"jira:issue_unlinked":  "issue.unlinked",
		"jira:comment_created": "comment.created",
		"jira:comment_updated": "comment.updated",
		"jira:comment_deleted": "comment.deleted",
		"sprint_started":       "sprint.started",
		"sprint_closed":        "sprint.closed",
	}

	if mapped, ok := mapping[event]; ok {
		return mapped
	}
	return event
}

type BridgeImporter struct {
	bridges map[Format]Bridge
}

func NewBridgeImporter() *BridgeImporter {
	importer := &BridgeImporter{
		bridges: make(map[Format]Bridge),
	}

	importer.bridges[FormatGitHub] = Bridge{
		Name:     "GitHub",
		Platform: "github",
		Format:  FormatGitHub,
		Parser:  ImportGitHub,
	}

	importer.bridges[FormatGitLab] = Bridge{
		Name:     "GitLab",
		Platform: "gitlab",
		Format:  FormatGitLab,
		Parser:  ImportGitLab,
	}

	importer.bridges[FormatMattermost] = Bridge{
		Name:     "Mattermost",
		Platform: "mattermost",
		Format:  FormatMattermost,
		Parser:  ImportMattermost,
	}

	importer.bridges[FormatJira] = Bridge{
		Name:     "Jira",
		Platform: "jira",
		Format:  FormatJira,
		Parser:  ImportJira,
	}

	importer.bridges[FormatSlack] = Bridge{
		Name:     "Slack",
		Platform: "slack",
		Format:  FormatSlack,
		Parser:  ImportSlack,
	}

	return importer
}

func (i *BridgeImporter) Import(format Format, data []byte) (*WebhookConfig, error) {
	bridge, ok := i.bridges[format]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupported, format)
	}

	return bridge.Parser(data)
}

func (i *BridgeImporter) ImportFromURL(url string) (*WebhookConfig, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImportFailed, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImportFailed, err)
	}

	contentType := resp.Header.Get("Content-Type")
	var format Format
	switch {
	case strings.Contains(contentType, "github"):
		format = FormatGitHub
	case strings.Contains(contentType, "gitlab"):
		format = FormatGitLab
	default:
		format = FormatGitHub
	}

	return i.Import(format, data)
}

func (i *BridgeImporter) DetectFormat(data []byte) Format {
	var test struct {
		URL      string `json:"url"`
		Token   string `json:"token"`
		Channel string `json:"channel"`
	}

	if err := json.Unmarshal(data, &test); err != "" {
		return ""
	}

	if test.Channel != "" {
		return FormatMattermost
	}
	if test.Token != "" && test.URL != "" {
		if strings.Contains(test.URL, "slack") {
			return FormatSlack
		}
		if strings.Contains(test.URL, "jira") {
			return FormatJira
		}
	}
	if test.URL != "" {
		return FormatGitHub
	}

	return ""
}

func (i *BridgeImporter) ListFormats() []string {
	formats := make([]string, 0)
	for format := range i.bridges {
		formats = append(formats, string(format))
	}
	return formats
}

type WebhookMigration struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	RepoID       int64     `json:"repo_id" gorm:"index"`
	FromPlatform string   `json:"from_platform"`
	Config      string   `json:"config" gorm:"type:text"`
	Status     string   `json:"status"`
	Error       string   `json:"error" gorm:"type:text"`
	ImportedAt  time.Time `json:"imported_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (m *WebhookMigration) ToWebhook() (*WebhookConfig, error) {
	return ImportGitHub([]byte(m.Config))
}

func DetectPlatform(url string) string {
	lowerURL := strings.ToLower(url)
	if strings.Contains(lowerURL, "github") {
		return "github"
	}
	if strings.Contains(lowerURL, "gitlab") {
		return "gitlab"
	}
	if strings.Contains(lowerURL, "mattermost") {
		return "mattermost"
	}
	if strings.Contains(lowerURL, "jira") {
		return "jira"
	}
	if strings.Contains(lowerURL, "slack") {
		return "slack"
	}
	return "unknown"
}

func ValidateURL(url string) error {
	parsed, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.URL.Scheme == "" {
		return fmt.Errorf("missing scheme")
	}
	if parsed.URL.Host == "" {
		return fmt.Errorf("missing host")
	}
	return nil
}

func GenerateWebhookID() string {
	return uuid.MustV4(uuid.NewV4()).String()
}

type ExportFormat struct {
	Version   string           `json:"version"`
	Name     string           `json:"name"`
	Platform string           `json:"platform"`
	Exported time.Time       `json:"exported_at"`
	Config  *WebhookConfig   `json:"config"`
}

func (e *ExportFormat) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e *ExportFormat) FromJSON(data []byte) error {
	return json.Unmarshal(data, e)
}

func init() {}