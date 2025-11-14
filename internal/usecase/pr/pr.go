package pr

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

type UserRepository interface {
	GetByID(ctx context.Context, userID domain.UserID) (*domain.DBUser, error)
	GetActiveByTeam(ctx context.Context, teamName domain.TeamName) ([]*domain.DBUser, error)
}

type PRRepository interface {
	Create(ctx context.Context, pr *domain.DBPullRequest) error
	GetByID(ctx context.Context, prID domain.PRID) (*domain.DBPullRequest, error)
	UpdateStatus(ctx context.Context, prID domain.PRID, status domain.PRStatus, mergedAt *sql.NullTime) error
	Exists(ctx context.Context, prID domain.PRID) (bool, error)
	AssignReviewer(ctx context.Context, reviewer *domain.DBReviewer) error
	RemoveReviewer(ctx context.Context, prID domain.PRID, userID domain.UserID) error
	GetReviewers(ctx context.Context, prID domain.PRID) ([]domain.UserID, error)
	IsReviewer(ctx context.Context, prID domain.PRID, userID domain.UserID) (bool, error)
	GetPRsByReviewer(ctx context.Context, userID domain.UserID) ([]*domain.DBPullRequest, error)
}

type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Usecase struct {
	userRepo  UserRepository
	prRepo    PRRepository
	txManager TransactionManager
}

func NewUsecase(userRepo UserRepository, prRepo PRRepository, txManager TransactionManager) *Usecase {
	return &Usecase{
		userRepo:  userRepo,
		prRepo:    prRepo,
		txManager: txManager,
	}
}

func (u *Usecase) CreatePR(ctx context.Context, req domain.CreatePRRequest) (*domain.PullRequest, error) {
	exists, err := u.prRepo.Exists(ctx, req.PullRequestID)
	if err != nil {
		return nil, fmt.Errorf("check pr exists: %w", err)
	}

	if exists {
		return nil, domain.ErrPRAlreadyExists
	}

	author, err := u.userRepo.GetByID(ctx, req.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("get author: %w", err)
	}

	if author == nil {
		return nil, domain.ErrAuthorNotFound
	}

	activeMembers, err := u.userRepo.GetActiveByTeam(ctx, author.TeamName)
	if err != nil {
		return nil, fmt.Errorf("get active members: %w", err)
	}

	var candidates []*domain.DBUser
	for _, member := range activeMembers {
		if member.UserID != req.AuthorID {
			candidates = append(candidates, member)
		}
	}

	selectedReviewers := u.selectRandomReviewers(candidates, 2)

	var assignedReviewers []domain.UserID

	err = u.txManager.Do(ctx, func(ctx context.Context) error {
		dbPR := &domain.DBPullRequest{
			PullRequestID:   req.PullRequestID,
			PullRequestName: req.PullRequestName,
			AuthorID:        req.AuthorID,
			Status:          domain.PRStatusOpen,
			CreatedAt:       time.Now(),
			MergedAt:        nil,
		}

		if err := u.prRepo.Create(ctx, dbPR); err != nil {
			return fmt.Errorf("create pr: %w", err)
		}

		for _, reviewer := range selectedReviewers {
			dbReviewer := &domain.DBReviewer{
				PullRequestID: req.PullRequestID,
				UserID:        reviewer.UserID,
				AssignedAt:    time.Now(),
			}

			if err := u.prRepo.AssignReviewer(ctx, dbReviewer); err != nil {
				return fmt.Errorf("assign reviewer: %w", err)
			}

			assignedReviewers = append(assignedReviewers, reviewer.UserID)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	createdAt := time.Now().Format(time.RFC3339)

	return &domain.PullRequest{
		PullRequestID:     req.PullRequestID,
		PullRequestName:   req.PullRequestName,
		AuthorID:          req.AuthorID,
		Status:            domain.PRStatusOpen,
		AssignedReviewers: assignedReviewers,
		CreatedAt:         &createdAt,
		MergedAt:          nil,
	}, nil
}

func (u *Usecase) MergePR(ctx context.Context, req domain.MergePRRequest) (*domain.PullRequest, error) {
	pr, err := u.prRepo.GetByID(ctx, req.PullRequestID)
	if err != nil {
		return nil, fmt.Errorf("get pr: %w", err)
	}

	if pr == nil {
		return nil, nil
	}

	if pr.Status == domain.PRStatusMerged {
		reviewers, err := u.prRepo.GetReviewers(ctx, pr.PullRequestID)
		if err != nil {
			return nil, fmt.Errorf("get reviewers: %w", err)
		}

		return u.buildPRResponse(pr, reviewers), nil
	}

	mergedAt := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	if err := u.prRepo.UpdateStatus(ctx, req.PullRequestID, domain.PRStatusMerged, &mergedAt); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	reviewers, err := u.prRepo.GetReviewers(ctx, pr.PullRequestID)
	if err != nil {
		return nil, fmt.Errorf("get reviewers: %w", err)
	}

	pr.Status = domain.PRStatusMerged
	pr.MergedAt = &mergedAt.Time

	return u.buildPRResponse(pr, reviewers), nil
}

func (u *Usecase) ReassignReviewer(ctx context.Context, req domain.ReassignReviewerRequest) (*domain.PullRequest, domain.UserID, error) {
	pr, err := u.prRepo.GetByID(ctx, req.PullRequestID)
	if err != nil {
		return nil, "", fmt.Errorf("get pr: %w", err)
	}

	if pr == nil {
		return nil, "", domain.ErrNotFound
	}

	if pr.Status == domain.PRStatusMerged {
		return nil, "", domain.ErrPRMerged
	}

	isReviewer, err := u.prRepo.IsReviewer(ctx, req.PullRequestID, req.OldUserID)
	if err != nil {
		return nil, "", fmt.Errorf("check is reviewer: %w", err)
	}

	if !isReviewer {
		return nil, "", domain.ErrNotAssigned
	}

	oldReviewer, err := u.userRepo.GetByID(ctx, req.OldUserID)
	if err != nil {
		return nil, "", fmt.Errorf("get old reviewer: %w", err)
	}

	if oldReviewer == nil {
		return nil, "", domain.ErrNotFound
	}

	activeMembers, err := u.userRepo.GetActiveByTeam(ctx, oldReviewer.TeamName)
	if err != nil {
		return nil, "", fmt.Errorf("get active members: %w", err)
	}

	currentReviewers, err := u.prRepo.GetReviewers(ctx, req.PullRequestID)
	if err != nil {
		return nil, "", fmt.Errorf("get reviewers: %w", err)
	}

	reviewerMap := make(map[domain.UserID]bool)
	for _, r := range currentReviewers {
		reviewerMap[r] = true
	}
	reviewerMap[pr.AuthorID] = true

	var candidates []*domain.DBUser
	for _, member := range activeMembers {
		if !reviewerMap[member.UserID] {
			candidates = append(candidates, member)
		}
	}

	if len(candidates) == 0 {
		return nil, "", domain.ErrNoCandidate
	}

	selected := u.selectRandomReviewers(candidates, 1)
	if len(selected) == 0 {
		return nil, "", domain.ErrNoCandidate
	}

	newReviewer := selected[0]

	err = u.txManager.Do(ctx, func(ctx context.Context) error {
		if err := u.prRepo.RemoveReviewer(ctx, req.PullRequestID, req.OldUserID); err != nil {
			return fmt.Errorf("remove reviewer: %w", err)
		}

		dbReviewer := &domain.DBReviewer{
			PullRequestID: req.PullRequestID,
			UserID:        newReviewer.UserID,
			AssignedAt:    time.Now(),
		}

		if err := u.prRepo.AssignReviewer(ctx, dbReviewer); err != nil {
			return fmt.Errorf("assign reviewer: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, "", err
	}

	updatedReviewers, err := u.prRepo.GetReviewers(ctx, req.PullRequestID)
	if err != nil {
		return nil, "", fmt.Errorf("get reviewers: %w", err)
	}

	return u.buildPRResponse(pr, updatedReviewers), newReviewer.UserID, nil
}

func (u *Usecase) GetUserReviews(ctx context.Context, userID domain.UserID) ([]domain.PullRequestShort, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		return nil, nil
	}

	prs, err := u.prRepo.GetPRsByReviewer(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get prs by reviewer: %w", err)
	}

	result := make([]domain.PullRequestShort, 0, len(prs))
	for _, pr := range prs {
		result = append(result, domain.PullRequestShort{
			PullRequestID:   pr.PullRequestID,
			PullRequestName: pr.PullRequestName,
			AuthorID:        pr.AuthorID,
			Status:          pr.Status,
		})
	}

	return result, nil
}

func (u *Usecase) selectRandomReviewers(candidates []*domain.DBUser, count int) []*domain.DBUser {
	if len(candidates) == 0 {
		return nil
	}

	if len(candidates) <= count {
		return candidates
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	shuffled := make([]*domain.DBUser, len(candidates))
	copy(shuffled, candidates)

	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

func (u *Usecase) buildPRResponse(pr *domain.DBPullRequest, reviewers []domain.UserID) *domain.PullRequest {
	createdAt := pr.CreatedAt.Format(time.RFC3339)

	var mergedAt *string
	if pr.MergedAt != nil {
		formatted := pr.MergedAt.Format(time.RFC3339)
		mergedAt = &formatted
	}

	return &domain.PullRequest{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: reviewers,
		CreatedAt:         &createdAt,
		MergedAt:          mergedAt,
	}
}
