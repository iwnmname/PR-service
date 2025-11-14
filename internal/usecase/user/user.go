package user

import (
	"context"
	"fmt"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

type UserRepository interface {
	GetByID(ctx context.Context, userID domain.UserID) (*domain.DBUser, error)
	SetActive(ctx context.Context, userID domain.UserID, isActive bool) error
}

type Usecase struct {
	userRepo UserRepository
}

func NewUsecase(userRepo UserRepository) *Usecase {
	return &Usecase{
		userRepo: userRepo,
	}
}

func (u *Usecase) SetActive(ctx context.Context, req domain.SetUserActiveRequest) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		return nil, nil
	}

	if err := u.userRepo.SetActive(ctx, req.UserID, req.IsActive); err != nil {
		return nil, fmt.Errorf("set active: %w", err)
	}

	return &domain.User{
		UserID:   user.UserID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: req.IsActive,
	}, nil
}
