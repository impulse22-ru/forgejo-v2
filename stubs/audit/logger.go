package audit

import (
	"encoding/json"
	"time"
)

type AuditEntry struct {
	ID        int64
	CreatedAt time.Time
	UserID    int64
	Action    string
	Details   map[string]interface{}
	IP        string
	RepoID    *int64
	OrgID     *int64
}

func (a *AuditEntry) MarshalJSON() (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{
		"id":         a.ID,
		"created_at": a.CreatedAt,
		"user_id":    a.UserID,
		"action":     a.Action,
		"details":    a.Details,
		"ip":         a.IP,
	})
}

type AuditLogger struct {
	entries []AuditEntry
}

func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		entries: make([]AuditEntry, 0),
	}
}

func (l *AuditLogger) Log(action string, userID int64, details map[string]interface{}) {
	l.entries = append(l.entries, AuditEntry{
		CreatedAt: time.Now(),
		UserID:    userID,
		Action:    action,
		Details:   details,
	})
}

func (l *AuditLogger) Query(filter AuditFilter) []AuditEntry {
	var result []AuditEntry
	for _, entry := range l.entries {
		if filter.Matches(entry) {
			result = append(result, entry)
		}
	}
	return result
}

type AuditFilter struct {
	Action  string `json:"action"`
	UserID  int64  `json:"user_id"`
	Start   *time.Time
	End    *time.Time
	RepoID  *int64
	OrgID   *int64
}

func (f *AuditFilter) Matches(entry AuditEntry) bool {
	if f.Action != "" && entry.Action != f.Action {
		return false
	}
	if f.UserID > 0 && entry.UserID != f.UserID {
		return false
	}
	if f.Start != nil && entry.CreatedAt.Before(*f.Start) {
		return false
	}
	if f.End != nil && entry.CreatedAt.After(*f.End) {
		return false
	}
	return true
}

func (f *AuditFilter) ToSQL() string {
	return ""
}