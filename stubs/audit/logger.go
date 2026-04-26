package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
)

type Entry struct {
	ID         int64     `json:"id" gorm:"primaryKey"`
	RepoID    *int64    `json:"repo_id" gorm:"index"`
	OrgID    *int64    `json:"org_id" gorm:"index"`
	UserID   int64     `json:"user_id" gorm:"index"`
	Action   string    `json:"action" gorm:"index"`
	IP       string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Details  string    `json:"details" gorm:"type:jsonb"`
	Created  time.Time `json:"created" gorm:"index"`
}

func (e *Entry) String() string {
	return fmt.Sprintf("AuditEntry{id=%d, action=%s, user=%d}", e.ID, e.Action, e.UserID)
}

func (e *Entry) GetDetails() map[string]interface{} {
	if e.Details == "" {
		return nil
	}
	var details map[string]interface{}
	json.Unmarshal([]byte(e.Details), &details)
	return details
}

func (e *Entry) SetDetails(details map[string]interface{}) {
	data, _ := json.Marshal(details)
	e.Details = string(data)
}

func (e *Entry) IsSensitive() bool {
	sensitive := []string{
		"user.login",
		"user.change_password",
		"repo.transfer",
		"org.transfer",
		"hook.create",
		"public_key.create",
	}
	for _, s := range sensitive {
		if e.Action == s {
			return true
		}
	}
	return false
}

type EntryList []Entry

func (l EntryList) FilterByAction(action string) EntryList {
	var filtered EntryList
	for _, e := range l {
		if e.Action == action {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (l EntryList) FilterByUser(userID int64) EntryList {
	var filtered EntryList
	for _, e := range l {
		if e.UserID == userID {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (l EntryList) FilterByDate(start, end time.Time) EntryList {
	var filtered EntryList
	for _, e := range l {
		if e.Created.After(start) && e.Created.Before(end) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

type Filter struct {
	Action   string     `form:"action"`
	UserID  int64      `form:"user_id"`
	RepoID  int64      `form:"repo_id"`
	OrgID   int64      `form:"org_id"`
	Start   *time.Time  `form:"start"`
	End     *time.Time `form:"end"`
	IP      string     `form:"ip"`
	Page    int        `form:"page"`
	Limit   int        `form:"limit"`
}

func (f *Filter) Defaults() {
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}

type EntryStore struct {
	db *sql.DB
}

func NewEntryStore() *EntryStore {
	return &EntryStore{}
}

func (s *EntryStore) List(filter Filter) ([]Entry, int, error) {
	return nil, 0, nil
}

func (s *EntryStore) ListByRepo(repoID int64, filter Filter) ([]Entry, int, error) {
	entries := []Entry{
		{ID: 1, RepoID: &repoID, UserID: 1, Action: "repo.create", Created: time.Now()},
		{ID: 2, RepoID: &repoID, UserID: 1, Action: "repo.push", Created: time.Now()},
	}
	return entries, len(entries), nil
}

func (s *EntryStore) ListByOrg(orgID int64, filter Filter) ([]Entry, int, error) {
	entries := []Entry{
		{ID: 1, OrgID: &orgID, UserID: 1, Action: "org.create", Created: time.Now()},
	}
	return entries, len(entries), nil
}

func (s *EntryStore) Get(id int64) (*Entry, error) {
	return nil, nil
}

func (s *EntryStore) Create(entry *Entry) error {
	entry.Created = time.Now()
	return nil
}

func (s *EntryStore) DeleteOlderThan(d time.Duration) error {
	return nil
}

type Handler struct {
	store *EntryStore
}

func NewHandler(store *EntryStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) HandleList(w http.ResponseWriter, req *http.Request) {
	filter := Filter{
		Action: req.URL.Query().Get("action"),
		Page:   1,
	}

	if repoID := req.URL.Query().Get("repo_id"); repoID != "" {
		filter.RepoID, _ = strconv.ParseInt(repoID, 10, 64)
	}

	entries, total, err := h.store.List(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Total", strconv.Itoa(total))
	data, _ := json.Marshal(entries)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *Handler) HandleExport(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=audit.csv")

	fmt.Fprintf(w, "ID,Action,User,IP,Date\n")
}

type AuditResponse struct {
	Data  []Entry `json:"data"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
}

func (e *Entry) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":        e.ID,
		"action":    e.Action,
		"user_id":   e.UserID,
		"ip":        e.IP,
		"created":   e.Created,
		"repo_id":   e.RepoID,
		"org_id":   e.OrgID,
	}
}

func (e *Entry) ActionLabel() string {
	labels := map[string]string{
		"repo.create": "Repository created",
		"repo.delete": "Repository deleted",
		"repo.push": "Git push",
		"repo.fork": "Repository forked",
		"issue.create": "Issue created",
		"issue.close": "Issue closed",
		"pull_request.create": "Pull request created",
		"pull_request.merge": "Pull request merged",
		"user.login": "User logged in",
		"user.logout": "User logged out",
		"public_key.create": "Deploy key added",
		"public_key.delete": "Deploy key removed",
	}

	if label, ok := labels[e.Action]; ok {
		return label
	}
	return e.Action
}

func FormatAction(action string) string {
	if idx := strings.LastIndex(action, "."); idx != -1 {
		parts := strings.Split(action, ".")
		if len(parts) == 2 {
			return fmt.Sprintf("%s %s", parts[1], parts[0])
		}
	}
	return action
}

func init() {}