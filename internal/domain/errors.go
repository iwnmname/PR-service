package domain

import "fmt"

type ErrorCode string

const (
	ErrCodeTeamExists  ErrorCode = "TEAM_EXISTS"
	ErrCodePRExists    ErrorCode = "PR_EXISTS"
	ErrCodePRMerged    ErrorCode = "PR_MERGED"
	ErrCodeNotAssigned ErrorCode = "NOT_ASSIGNED"
	ErrCodeNoCandidate ErrorCode = "NO_CANDIDATE"
	ErrCodeNotFound    ErrorCode = "NOT_FOUND"
)

type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

func NewErrorResponse(code ErrorCode, message string) ErrorResponse {
	return ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
		},
	}
}

func NewTeamExistsError(teamName TeamName) ErrorResponse {
	return NewErrorResponse(
		ErrCodeTeamExists,
		fmt.Sprintf("team '%s' already exists", teamName),
	)
}

func NewPRExistsError(prID PRID) ErrorResponse {
	return NewErrorResponse(
		ErrCodePRExists,
		fmt.Sprintf("PR '%s' already exists", prID),
	)
}

func NewPRMergedError() ErrorResponse {
	return NewErrorResponse(
		ErrCodePRMerged,
		"cannot reassign on merged PR",
	)
}

func NewNotAssignedError() ErrorResponse {
	return NewErrorResponse(
		ErrCodeNotAssigned,
		"reviewer is not assigned to this PR",
	)
}

func NewNoCandidateError() ErrorResponse {
	return NewErrorResponse(
		ErrCodeNoCandidate,
		"no active replacement candidate in team",
	)
}

func NewNotFoundError(resource string) ErrorResponse {
	return NewErrorResponse(
		ErrCodeNotFound,
		fmt.Sprintf("%s not found", resource),
	)
}
