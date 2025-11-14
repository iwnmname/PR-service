package pr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/iwnmname/PR-service.git/internal/domain"
	"github.com/iwnmname/PR-service.git/internal/pkg/transaction"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) getDB(ctx context.Context) interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if tx, ok := transaction.GetTx(ctx); ok {
		return tx
	}
	return r.db
}

func (r *Repository) Create(ctx context.Context, pr *domain.DBPullRequest) error {
	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at, merged_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.getDB(ctx).ExecContext(
		ctx,
		query,
		pr.PullRequestID,
		pr.PullRequestName,
		pr.AuthorID,
		pr.Status,
		pr.CreatedAt,
		pr.MergedAt,
	)

	if err != nil {
		return fmt.Errorf("create pull request: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, prID domain.PRID) (*domain.DBPullRequest, error) {
	query := `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`

	var pr domain.DBPullRequest
	err := r.getDB(ctx).QueryRowContext(ctx, query, prID).Scan(
		&pr.PullRequestID,
		&pr.PullRequestName,
		&pr.AuthorID,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get pull request by id: %w", err)
	}

	return &pr, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, prID domain.PRID, status domain.PRStatus, mergedAt *sql.NullTime) error {
	query := `
		UPDATE pull_requests
		SET status = $2, merged_at = $3
		WHERE pull_request_id = $1
	`

	_, err := r.getDB(ctx).ExecContext(ctx, query, prID, status, mergedAt)

	if err != nil {
		return fmt.Errorf("update pr status: %w", err)
	}

	return nil
}

func (r *Repository) Exists(ctx context.Context, prID domain.PRID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`

	var exists bool
	err := r.getDB(ctx).QueryRowContext(ctx, query, prID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("check pr exists: %w", err)
	}

	return exists, nil
}

func (r *Repository) AssignReviewer(ctx context.Context, reviewer *domain.DBReviewer) error {
	query := `
		INSERT INTO pr_reviewers (pull_request_id, user_id, assigned_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (pull_request_id, user_id) DO NOTHING
	`

	_, err := r.getDB(ctx).ExecContext(
		ctx,
		query,
		reviewer.PullRequestID,
		reviewer.UserID,
		reviewer.AssignedAt,
	)

	if err != nil {
		return fmt.Errorf("assign reviewer: %w", err)
	}

	return nil
}

func (r *Repository) RemoveReviewer(ctx context.Context, prID domain.PRID, userID domain.UserID) error {
	query := `
		DELETE FROM pr_reviewers
		WHERE pull_request_id = $1 AND user_id = $2
	`

	_, err := r.getDB(ctx).ExecContext(ctx, query, prID, userID)

	if err != nil {
		return fmt.Errorf("remove reviewer: %w", err)
	}

	return nil
}

func (r *Repository) GetReviewers(ctx context.Context, prID domain.PRID) ([]domain.UserID, error) {
	query := `
		SELECT user_id
		FROM pr_reviewers
		WHERE pull_request_id = $1
		ORDER BY assigned_at
	`

	rows, err := r.getDB(ctx).QueryContext(ctx, query, prID)
	if err != nil {
		return nil, fmt.Errorf("get reviewers: %w", err)
	}
	defer rows.Close()

	var reviewers []domain.UserID
	for rows.Next() {
		var userID domain.UserID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan reviewer: %w", err)
		}
		reviewers = append(reviewers, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return reviewers, nil
}

func (r *Repository) IsReviewer(ctx context.Context, prID domain.PRID, userID domain.UserID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM pr_reviewers
			WHERE pull_request_id = $1 AND user_id = $2
		)
	`

	var exists bool
	err := r.getDB(ctx).QueryRowContext(ctx, query, prID, userID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("check is reviewer: %w", err)
	}

	return exists, nil
}

func (r *Repository) GetPRsByReviewer(ctx context.Context, userID domain.UserID) ([]*domain.DBPullRequest, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		INNER JOIN pr_reviewers rev ON pr.pull_request_id = rev.pull_request_id
		WHERE rev.user_id = $1
		ORDER BY pr.created_at DESC
	`

	rows, err := r.getDB(ctx).QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get prs by reviewer: %w", err)
	}
	defer rows.Close()

	var prs []*domain.DBPullRequest
	for rows.Next() {
		var pr domain.DBPullRequest
		if err := rows.Scan(
			&pr.PullRequestID,
			&pr.PullRequestName,
			&pr.AuthorID,
			&pr.Status,
			&pr.CreatedAt,
			&pr.MergedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pr: %w", err)
		}
		prs = append(prs, &pr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return prs, nil
}
