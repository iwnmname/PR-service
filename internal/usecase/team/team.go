package team

import (
	"context"
	"fmt"
	"time"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.DBUser) error
	Update(ctx context.Context, user *domain.DBUser) error
	GetByID(ctx context.Context, userID domain.UserID) (*domain.DBUser, error)
	GetByTeam(ctx context.Context, teamName domain.TeamName) ([]*domain.DBUser, error)
}

type TeamRepository interface {
	Create(ctx context.Context, team *domain.DBTeam) error
	Exists(ctx context.Context, teamName domain.TeamName) (bool, error)
}

type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Usecase struct {
	userRepo  UserRepository
	teamRepo  TeamRepository
	txManager TransactionManager
}

func NewUsecase(userRepo UserRepository, teamRepo TeamRepository, txManager TransactionManager) *Usecase {
	return &Usecase{
		userRepo:  userRepo,
		teamRepo:  teamRepo,
		txManager: txManager,
	}
}

func (u *Usecase) CreateTeam(ctx context.Context, req domain.CreateTeamRequest) (*domain.Team, error) {
	exists, err := u.teamRepo.Exists(ctx, req.TeamName)
	if err != nil {
		return nil, fmt.Errorf("check team exists: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("team already exists")
	}

	var createdMembers []domain.TeamMember

	err = u.txManager.Do(ctx, func(ctx context.Context) error {
		dbTeam := &domain.DBTeam{
			TeamName:  req.TeamName,
			CreatedAt: time.Now(),
		}

		if err := u.teamRepo.Create(ctx, dbTeam); err != nil {
			return fmt.Errorf("create team: %w", err)
		}

		for _, member := range req.Members {
			existingUser, err := u.userRepo.GetByID(ctx, member.UserID)
			if err != nil {
				return fmt.Errorf("get user: %w", err)
			}

			if existingUser == nil {
				dbUser := &domain.DBUser{
					UserID:    member.UserID,
					Username:  member.Username,
					TeamName:  req.TeamName,
					IsActive:  member.IsActive,
					CreatedAt: time.Now(),
				}

				if err := u.userRepo.Create(ctx, dbUser); err != nil {
					return fmt.Errorf("create user: %w", err)
				}
			} else {
				existingUser.Username = member.Username
				existingUser.TeamName = req.TeamName
				existingUser.IsActive = member.IsActive

				if err := u.userRepo.Update(ctx, existingUser); err != nil {
					return fmt.Errorf("update user: %w", err)
				}
			}

			createdMembers = append(createdMembers, member)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &domain.Team{
		TeamName: req.TeamName,
		Members:  createdMembers,
	}, nil
}

func (u *Usecase) GetTeam(ctx context.Context, teamName domain.TeamName) (*domain.Team, error) {
	exists, err := u.teamRepo.Exists(ctx, teamName)
	if err != nil {
		return nil, fmt.Errorf("check team exists: %w", err)
	}

	if !exists {
		return nil, nil
	}

	users, err := u.userRepo.GetByTeam(ctx, teamName)
	if err != nil {
		return nil, fmt.Errorf("get users by team: %w", err)
	}

	members := make([]domain.TeamMember, 0, len(users))
	for _, user := range users {
		members = append(members, domain.TeamMember{
			UserID:   user.UserID,
			Username: user.Username,
			IsActive: user.IsActive,
		})
	}

	return &domain.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}
