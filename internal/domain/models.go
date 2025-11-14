package domain

import "time"

type UserID string
type TeamName string
type PRID string

type User struct {
	UserID   UserID   `json:"user_id"`
	Username string   `json:"username"`
	TeamName TeamName `json:"team_name"`
	IsActive bool     `json:"is_active"`
}

type TeamMember struct {
	UserID   UserID `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	TeamName TeamName     `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type PRStatus string

const (
	PRStatusOpen   PRStatus = "OPEN"
	PRStatusMerged PRStatus = "MERGED"
)

type PullRequest struct {
	PullRequestID     PRID     `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          UserID   `json:"author_id"`
	Status            PRStatus `json:"status"`
	AssignedReviewers []UserID `json:"assigned_reviewers"`
	CreatedAt         *string  `json:"createdAt,omitempty"`
	MergedAt          *string  `json:"mergedAt,omitempty"`
}

type PullRequestShort struct {
	PullRequestID   PRID     `json:"pull_request_id"`
	PullRequestName string   `json:"pull_request_name"`
	AuthorID        UserID   `json:"author_id"`
	Status          PRStatus `json:"status"`
}

type CreateTeamRequest struct {
	TeamName TeamName     `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type CreateTeamResponse struct {
	Team Team `json:"team"`
}

type GetTeamResponse Team

type SetUserActiveRequest struct {
	UserID   UserID `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type SetUserActiveResponse struct {
	User User `json:"user"`
}

type CreatePRRequest struct {
	PullRequestID   PRID   `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        UserID `json:"author_id"`
}

type CreatePRResponse struct {
	PR PullRequest `json:"pr"`
}

type MergePRRequest struct {
	PullRequestID PRID `json:"pull_request_id"`
}

type MergePRResponse struct {
	PR PullRequest `json:"pr"`
}

type ReassignReviewerRequest struct {
	PullRequestID PRID   `json:"pull_request_id"`
	OldUserID     UserID `json:"old_user_id"`
}

type ReassignReviewerResponse struct {
	PR         PullRequest `json:"pr"`
	ReplacedBy UserID      `json:"replaced_by"`
}

type GetUserReviewsResponse struct {
	UserID       UserID             `json:"user_id"`
	PullRequests []PullRequestShort `json:"pull_requests"`
}

type DBUser struct {
	UserID    UserID
	Username  string
	TeamName  TeamName
	IsActive  bool
	CreatedAt time.Time
}

type DBTeam struct {
	TeamName  TeamName
	CreatedAt time.Time
}

type DBPullRequest struct {
	PullRequestID   PRID
	PullRequestName string
	AuthorID        UserID
	Status          PRStatus
	CreatedAt       time.Time
	MergedAt        *time.Time
}

type DBReviewer struct {
	PullRequestID PRID
	UserID        UserID
	AssignedAt    time.Time
}
