package time

import (
	"time"
)

type TimeEntry struct {
	ID          int64
	IssueID    int64
	UserID     int64
	TimeSpent  time.Duration
	TimeEstimate time.Duration
	Created    time.Time
	Updated    time.Time
}

func ParseDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	return d * time.Hour, err
}

func FormatDuration(d time.Duration) string {
	h := d.Hours()
	if h < 1 {
		return "0m"
	}
	return string(rune(h)) + "h"
}

type TimeTracker struct {
	entries map[int64][]TimeEntry
}

func NewTimeTracker() *TimeTracker {
	return &TimeTracker{
		entries: make(map[int64][]TimeEntry),
	}
}

func (t *TimeTracker) AddEntry(issueID int64, entry TimeEntry) {
	t.entries[issueID] = append(t.entries[issueID], entry)
}

func (t *TimeTracker) GetTotalTime(issueID int64) time.Duration {
	var total time.Duration
	for _, entry := range t.entries[issueID] {
		total += entry.TimeSpent
	}
	return total
}

func (t *TimeTracker) GetEstimatedTime(issueID int64) time.Duration {
	if entries, ok := t.entries[issueID]; ok && len(entries) > 0 {
		return entries[0].TimeEstimate
	}
	return 0
}

func (t *TimeTracker) GetRemainingTime(issueID int64) time.Duration {
	return t.GetEstimatedTime(issueID) - t.GetTotalTime(issueID)
}

type TimeReport struct {
	UserID    int64
	Date     string
	Duration time.Duration
}

type Vote struct {
	ID        int64
	IssueID   int64
	UserID    int64
	Direction string
	Created   time.Time
}

type IssueVoting struct {
	votes map[int64][]Vote
}

func NewIssueVoting() *IssueVoting {
	return &IssueVoting{
		votes: make(map[int64][]Vote),
	}
}

func (v *IssueVoting) Upvote(issueID, userID int64) {
	v.votes[issueID] = append(v.votes[issueID], Vote{
		IssueID:   issueID,
		UserID:    userID,
		Direction: "up",
		Created:  time.Now(),
	})
}

func (v *IssueVoting) RemoveVote(issueID, userID int64) {
	entries := v.votes[issueID]
	for i, vote := range entries {
		if vote.UserID == userID {
			v.votes[issueID] = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}

func (v *IssueVoting) GetVoteCount(issueID int64) int {
	return len(v.votes[issueID])
}

func (v *IssueVoting) HasVoted(issueID, userID int64) bool {
	for _, vote := range v.votes[issueID] {
		if vote.UserID == userID {
			return true
		}
	}
	return false
}