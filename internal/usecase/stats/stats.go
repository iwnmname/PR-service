package stats

import (
	"context"
	"fmt"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

type StatsRepository interface {
	GetTotalUsers(ctx context.Context) (int, error)
	GetTotalTeams(ctx context.Context) (int, error)
	GetTotalPRs(ctx context.Context) (int, error)
	GetPRsByStatus(ctx context.Context) ([]domain.PRStatusStat, error)
	GetTopReviewers(ctx context.Context, limit int) ([]domain.UserAssignmentStat, error)
}

type Usecase struct {
	statsRepo StatsRepository
}

func NewUsecase(statsRepo StatsRepository) *Usecase {
	return &Usecase{
		statsRepo: statsRepo,
	}
}

func (u *Usecase) GetStatistics(ctx context.Context) (*domain.StatisticsResponse, error) {
	totalUsers, err := u.statsRepo.GetTotalUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get total users: %w", err)
	}

	totalTeams, err := u.statsRepo.GetTotalTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("get total teams: %w", err)
	}

	totalPRs, err := u.statsRepo.GetTotalPRs(ctx)
	if err != nil {
		return nil, fmt.Errorf("get total prs: %w", err)
	}

	prsByStatus, err := u.statsRepo.GetPRsByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("get prs by status: %w", err)
	}

	topReviewers, err := u.statsRepo.GetTopReviewers(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("get top reviewers: %w", err)
	}

	return &domain.StatisticsResponse{
		TotalUsers:   totalUsers,
		TotalTeams:   totalTeams,
		TotalPRs:     totalPRs,
		PRsByStatus:  prsByStatus,
		TopReviewers: topReviewers,
	}, nil
}