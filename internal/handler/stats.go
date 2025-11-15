package handler

import (
	"context"
	"net/http"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

type StatsUsecase interface {
	GetStatistics(ctx context.Context) (*domain.StatisticsResponse, error)
}

type StatsHandler struct {
	statsUsecase StatsUsecase
}

func NewStatsHandler(statsUsecase StatsUsecase) *StatsHandler {
	return &StatsHandler{
		statsUsecase: statsUsecase,
	}
}

func (h *StatsHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsUsecase.GetStatistics(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, domain.ErrCodeNotFound, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, stats)
}