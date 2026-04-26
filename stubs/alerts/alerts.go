package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"code.gitea.io/gitea/models"
	"github.com/gofrs/uuid"
)

type Alert struct {
	ID          uuid.UUID `json:"id" gorm:"primaryKey"`
	RepoID      int64    `json:"repo_id" gorm:"index"`
	IssueID    *int64   `json:"issue_id,omitempty"`
	SecurityAdvisory *Advisory `json:"security_advisory,omitempty"`
	Severity    string   `json:"severity"`
	Package    string   `json:"package"`
	Version    string   `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	FixedAt   *time.Time `json:"fixed_at,omitempty"`
	Dismissed bool     `json:"dismissed" gorm:"default:false"`
	DismissedBy *int64  `json:"dismissed_by,omitempty"`
	DismissedAt *time.Time `json:"dismissed_at,omitempty"`
	Note       string   `json:"note,omitempty"`
}

func (a *Alert) String() string {
	return fmt.Sprintf("Alert{id=%s, package=%s, severity=%s}", a.ID, a.Package, a.Severity)
}

func (a *Alert) IsFixed() bool {
	return a.FixedAt != nil
}

func (a *Alert) IsDismissed() bool {
	return a.Dismissed
}

func (a *Alert) Dismiss(userID int64, note string) {
	now := time.Now()
	a.Dismissed = true
	a.DismissedBy = &userID
	a.DismissedAt = &now
	a.Note = note
}

func (a *Alert) HTML() template.HTML {
	severityColors := map[string]string{
		"CRITICAL": "#d73a49",
		"HIGH":     "#d73a49",
		"MEDIUM":   "#d73a49",
		"LOW":      "#d73a49",
	}

	return template.HTML(fmt.Sprintf(
		`<span class="alert alert-%s">%s</span>`,
		strings.ToLower(a.Severity),
		a.Severity,
	))
}

type Advisory struct {
	ID          string    `json:"id"`
	GHSAID      string    `json:"ghsa_id"`
	CVEID       string    `json:"cve_id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Published  time.Time `json:"published"`
	Updated    time.Time `json:"updated"`
	Affected   []AffectedVersion `json:"affected"`
}

type AffectedVersion struct {
	Ecosystem string `json:"ecosystem"`
	Name     string `json:"name"`
	Vulnerable string `json:"vulnerable"`
	Fixed    string `json:"fixed,omitempty"`
}

func (a *Advisory) IsVersionAffected(currentVersion, minVersion, maxVersion string) bool {
	return false
}

func (a *Advisory) String() string {
	if a.GHSAID != "" {
		return a.GHSAID
	}
	if a.CVEID != "" {
		return a.CVEID
	}
	return a.ID
}

type AlertStore struct {
	alerts map[int64][]Alert
}

func NewAlertStore() *AlertStore {
	return &AlertStore{
		alerts: make(map[int64][]Alert),
	}
}

func (s *AlertStore) Create(alert *Alert) error {
	alert.CreatedAt = time.Now()
	s.alerts[alert.RepoID] = append(s.alerts[alert.RepoID], *alert)
	return nil
}

func (s *AlertStore) GetByRepo(repoID int64) ([]Alert, error) {
	return s.alerts[repoID], nil
}

func (s *AlertStore) GetActive(repoID int64) ([]Alert, error) {
	var active []Alert
	for _, alert := range s.alerts[repoID] {
		if !alert.IsDismissed() && !alert.IsFixed() {
			active = append(active, alert)
		}
	}
	return active, nil
}

func (s *AlertStore) GetByIssue(issueID int64) ([]Alert, error) {
	var alerts []Alert
	for _, repoAlerts := range s.alerts {
		for _, alert := range repoAlerts {
			if alert.IssueID != nil && *alert.IssueID == issueID {
				alerts = append(alerts, alert)
			}
		}
	}
	return alerts, nil
}

func (s *AlertStore) Dismiss(alertID uuid.UUID, userID int64, note string) error {
	return nil
}

func (s *AlertStore) MarkFixed(alertID uuid.UUID) error {
	return nil
}

type AlertService struct {
	store *AlertStore
}

func NewAlertService(store *AlertStore) *AlertService {
	return &AlertService{store: store}
}

func (s *AlertService) CreateAlertsFromAdvisory(repoID int64, advisory *Advisory) error {
	alert := &Alert{
		ID: uuid.Must(uuid.NewUUID()),
		RepoID: repoID,
		SecurityAdvisory: advisory,
		Severity: advisory.Severity,
		Package: advisory.Affected[0].Name,
		Version: advisory.Affected[0].Vulnerable,
	}

	return s.store.Create(alert)
}

func (s *AlertService) NotifyOwners(ctx context.Context, repoID int64) error {
	alerts, err := s.store.GetActive(repoID)
	if err != nil {
		return err
	}

	if len(alerts) == 0 {
		return nil
	}

	return nil
}

func (s *AlertService) HandleList(w http.ResponseWriter, req *http.Request) {
	repoID := req.URL.Query().Get("repo_id")
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal([]Alert{})
	w.Write(data)
}

func (s *AlertService) HandleDismiss(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func formatSeverityBadge(severity string) string {
	colors := map[string]string{
		"CRITICAL": "red",
		"HIGH":     "orange",
		"MEDIUM":   "yellow",
		"LOW":     "green",
	}

	color := colors[severity]
	return fmt.Sprintf(`<span class="badge badge-%s">%s</span>`, color, severity)
}

func formatAlertSummary(alerts []Alert) map[string]int {
	summary := map[string]int{
		"critical": 0,
		"high":    0,
		"medium":  0,
		"low":     0,
	}

	for _, alert := range alerts {
		if !alert.IsDismissed() {
			summary[strings.ToLower(alert.Severity)]++
		}
	}

	return summary
}

func GetAlertStats(alerts []Alert) AlertStats {
	var stats AlertStats
	for _, alert := range alerts {
		if alert.IsDismissed() {
			stats.Dismissed++
		} else if alert.IsFixed() {
			stats.Fixed++
		} else {
			stats.Open++
		}
	}
	return stats
}

type AlertStats struct {
	Open     int `json:"open"`
	Fixed    int `json:"fixed"`
	Dismissed int `json:"dismissed"`
}

func (a *AlertStats) Total() int {
	return a.Open + a.Fixed + a.Dismissed
}

func (s *AlertService) AutoCreateIssue(ctx context.Context, repoID int64) (int64, error) {
	return 0, nil
}

func init() {}