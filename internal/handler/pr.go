package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

type PRUsecase interface {
	CreatePR(ctx context.Context, req domain.CreatePRRequest) (*domain.PullRequest, error)
	MergePR(ctx context.Context, req domain.MergePRRequest) (*domain.PullRequest, error)
	ReassignReviewer(ctx context.Context, req domain.ReassignReviewerRequest) (*domain.PullRequest, domain.UserID, error)
	GetUserReviews(ctx context.Context, userID domain.UserID) ([]domain.PullRequestShort, error)
}

type PRHandler struct {
	prUsecase PRUsecase
}

func NewPRHandler(prUsecase PRUsecase) *PRHandler {
	return &PRHandler{
		prUsecase: prUsecase,
	}
}

func (h *PRHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req domain.CreatePRRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, domain.ErrCodeNotFound, "invalid request body")
		return
	}

	pr, err := h.prUsecase.CreatePR(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrPRAlreadyExists) {
			respondError(w, http.StatusConflict, domain.ErrCodePRExists, "PR id already exists")
			return
		}

		if errors.Is(err, domain.ErrAuthorNotFound) {
			respondError(w, http.StatusNotFound, domain.ErrCodeNotFound, "author not found")
			return
		}

		respondError(w, http.StatusInternalServerError, domain.ErrCodeNotFound, "internal server error")
		return
	}

	response := domain.CreatePRResponse{
		PR: *pr,
	}

	respondJSON(w, http.StatusCreated, response)
}

func (h *PRHandler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req domain.MergePRRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, domain.ErrCodeNotFound, "invalid request body")
		return
	}

	pr, err := h.prUsecase.MergePR(r.Context(), req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, domain.ErrCodeNotFound, "internal server error")
		return
	}

	if pr == nil {
		respondError(w, http.StatusNotFound, domain.ErrCodeNotFound, "pull request not found")
		return
	}

	response := domain.MergePRResponse{
		PR: *pr,
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *PRHandler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req domain.ReassignReviewerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, domain.ErrCodeNotFound, "invalid request body")
		return
	}

	pr, replacedBy, err := h.prUsecase.ReassignReviewer(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrPRMerged) {
			respondError(w, http.StatusConflict, domain.ErrCodePRMerged, "cannot reassign on merged PR")
			return
		}

		if errors.Is(err, domain.ErrNotAssigned) {
			respondError(w, http.StatusConflict, domain.ErrCodeNotAssigned, "reviewer is not assigned to this PR")
			return
		}

		if errors.Is(err, domain.ErrNoCandidate) {
			respondError(w, http.StatusConflict, domain.ErrCodeNoCandidate, "no active replacement candidate in team")
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, domain.ErrCodeNotFound, "pull request or user not found")
			return
		}

		respondError(w, http.StatusInternalServerError, domain.ErrCodeNotFound, "internal server error")
		return
	}

	response := domain.ReassignReviewerResponse{
		PR:         *pr,
		ReplacedBy: replacedBy,
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *PRHandler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := domain.UserID(r.URL.Query().Get("user_id"))

	if userID == "" {
		respondError(w, http.StatusBadRequest, domain.ErrCodeNotFound, "user_id is required")
		return
	}

	prs, err := h.prUsecase.GetUserReviews(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, domain.ErrCodeNotFound, "internal server error")
		return
	}

	if prs == nil {
		respondError(w, http.StatusNotFound, domain.ErrCodeNotFound, "user not found")
		return
	}

	response := domain.GetUserReviewsResponse{
		UserID:       userID,
		PullRequests: prs,
	}

	respondJSON(w, http.StatusOK, response)
}
