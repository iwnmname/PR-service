package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

type UserUsecase interface {
	SetActive(ctx context.Context, req domain.SetUserActiveRequest) (*domain.User, error)
}

type UserHandler struct {
	userUsecase UserUsecase
}

func NewUserHandler(userUsecase UserUsecase) *UserHandler {
	return &UserHandler{
		userUsecase: userUsecase,
	}
}

func (h *UserHandler) SetActive(w http.ResponseWriter, r *http.Request) {
	var req domain.SetUserActiveRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, domain.ErrCodeNotFound, "invalid request body")
		return
	}

	user, err := h.userUsecase.SetActive(r.Context(), req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, domain.ErrCodeNotFound, "internal server error")
		return
	}

	if user == nil {
		respondError(w, http.StatusNotFound, domain.ErrCodeNotFound, "user not found")
		return
	}

	response := domain.SetUserActiveResponse{
		User: *user,
	}

	respondJSON(w, http.StatusOK, response)
}
