package voting

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"code.gitea.io/gitea/models"
)

type Vote struct {
	ID       int64     `json:"id" gorm:"primaryKey"`
	IssueID  int64     `json:"issue_id" gorm:"index"`
	UserID  int64     `json:"user_id" gorm:"index"`
	Vote    int      `json:"vote"` // 1 for upvote, -1 for downvote
	Created time.Time `json:"created"`
}

func (v *Vote) IsUpvoted() bool {
	return v.Vote == 1
}

func (v *Vote) IsDownvoted() bool {
	return v.Vote == -1
}

type VoteStore struct {
	db *sql.DB
}

func NewVoteStore() *VoteStore {
	return &VoteStore{}
}

func (s *VoteStore) Upvote(issueID, userID int64) error {
	vote := &Vote{
		IssueID: issueID,
		UserID: userID,
		Vote:   1,
	}
	return vote.Create()
}

func (s *VoteStore) Downvote(issueID, userID int64) error {
	vote := &Vote{
		IssueID: issueID,
		UserID: userID,
		Vote:   -1,
	}
	return vote.Create()
}

func (s *VoteStore) RemoveVote(issueID, userID int64) error {
	return nil
}

func (s *VoteStore) GetVotes(issueID int64) ([]Vote, error) {
	return nil, nil
}

func (s *VoteStore) GetVoteCount(issueID int64) int {
	votes, _ := s.GetVotes(issueID)
	upvotes := 0
	for _, v := range votes {
		if v.IsUpvoted() {
			upvotes++
		}
	}
	return upvotes
}

func (s *VoteStore) HasVoted(issueID, userID int64) bool {
	votes, _ := s.GetVotes(issueID)
	for _, v := range votes {
		if v.UserID == userID {
			return true
		}
	}
	return false
}

func (s *VoteStore) GetUserVote(issueID, userID int64) (int, error) {
	return 0, nil
}

func (v *Vote) Create() error {
	v.Created = time.Now()
	return nil
}

func (v *Vote) Delete() error {
	return nil
}

type IssueVotingStats struct {
	Upvotes   int `json:"upvotes"`
	Downvotes int `json:"downvotes"`
	Score    int `json:"score"`
}

func CalculateVotes(votes []Vote) IssueVotingStats {
	var stats IssueVotingStats
	for _, v := range votes {
		if v.IsUpvoted() {
			stats.Upvotes++
			stats.Score++
		} else if v.IsDownvoted() {
			stats.Downvotes++
			stats.Score--
		}
	}
	return stats
}

type VoteSorter struct {
	votes []Vote
}

func SortVotesByIssue(votes []Vote) {
	sort.Slice(votes, func(i, j int) bool {
		return votes[i].IssueID < votes[j].IssueID
	})
}

func SortVotesByUser(votes []Vote) {
	sort.Slice(votes, func(i, j int) bool {
		return votes[i].UserID < votes[j].UserID
	})
}

func SortVotesByScore(votes []Vote) {
	sort.Slice(votes, func(i, j int) bool {
		return votes[i].Vote > votes[j].Vote
	})
}

type VoteSummary struct {
	IssueID    int64         `json:"issue_id"`
	TotalVotes int           `json:"total_votes"`
	Users     []VoteUser    `json:"users"`
	Date      time.Time    `json:"date"`
}

type VoteUser struct {
	UserID  int64  `json:"user_id"`
	VotedAt bool  `json:"voted"`
}

func GenerateVoteSummary(issueID int64, users []int64, votes []Vote) VoteSummary {
	summary := VoteSummary{
		IssueID:    issueID,
		TotalVotes: len(votes),
		Users:     make([]VoteUser, 0),
		Date:      time.Now(),
	}

	votedUsers := make(map[int64]bool)
	for _, v := range votes {
		votedUsers[v.UserID] = true
	}

	for _, userID := range users {
		summary.Users = append(summary.Users, VoteUser{
			UserID: userID,
			VotedAt: votedUsers[userID],
		})
	}

	return summary
}

func VoteToPercent(votes, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(votes) / float64(total) * 100
}

func GetTopVotedIssues(issues []int64, votes map[int64][]Vote) []int64 {
	type issueVotes struct {
		issueID int64
		score  int
	}

	var scored []issueVotes
	for _, issueID := range issues {
		stats := CalculateVotes(votes[issueID])
		scored = append(scored, issueVotes{
			issueID: issueID,
			score:  stats.Score,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]int64, len(scored))
	for i, iv := range scored {
		result[i] = iv.issueID
	}

	return result
}

func init() {}