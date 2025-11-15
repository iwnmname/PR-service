package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

type TeamUsecase interface {
	CreateTeam(ctx context.Context, req domain.CreateTeamRequest) (*domain.Team, error)
	GetTeam(ctx context.Context, teamName domain.TeamName) (*domain.Team, error)
}

type TeamHandler struct {
	teamUsecase TeamUsecase
}

func NewTeamHandler(teamUsecase TeamUsecase) *TeamHandler {
	return &TeamHandler{
		teamUsecase: teamUsecase,
	}
}

func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, domain.ErrCodeNotFound, "invalid request body")
		return
	}

	team, err := h.teamUsecase.CreateTeam(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrTeamAlreadyExists) {
			respondError(w, http.StatusBadRequest, domain.ErrCodeTeamExists, "team_name already exists")
			return
		}

		respondError(w, http.StatusInternalServerError, domain.ErrCodeNotFound, "internal server error")
		return
	}

	response := domain.CreateTeamResponse{
		Team: *team,
	}

	respondJSON(w, http.StatusCreated, response)
}

func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := domain.TeamName(r.URL.Query().Get("team_name"))

	if teamName == "" {
		respondError(w, http.StatusBadRequest, domain.ErrCodeNotFound, "team_name is required")
		return
	}

	team, err := h.teamUsecase.GetTeam(r.Context(), teamName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, domain.ErrCodeNotFound, "internal server error")
		return
	}

	if team == nil {
		respondError(w, http.StatusNotFound, domain.ErrCodeNotFound, "team not found")
		return
	}

	respondJSON(w, http.StatusOK, team)
}
