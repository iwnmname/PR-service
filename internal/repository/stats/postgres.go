package stats

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/iwnmname/PR-service.git/internal/domain"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetTotalUsers(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`

	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get total users: %w", err)
	}

	return count, nil
}

func (r *Repository) GetTotalTeams(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM teams`

	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get total teams: %w", err)
	}

	return count, nil
}

func (r *Repository) GetTotalPRs(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM pull_requests`

	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get total prs: %w", err)
	}

	return count, nil
}

func (r *Repository) GetPRsByStatus(ctx context.Context) ([]domain.PRStatusStat, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM pull_requests
		GROUP BY status
		ORDER BY status
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get prs by status: %w", err)
	}
	defer rows.Close()

	var stats []domain.PRStatusStat
	for rows.Next() {
		var stat domain.PRStatusStat
		if err := rows.Scan(&stat.Status, &stat.Count); err != nil {
			return nil, fmt.Errorf("scan pr status stat: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return stats, nil
}

func (r *Repository) GetTopReviewers(ctx context.Context, limit int) ([]domain.UserAssignmentStat, error) {
	query := `
		SELECT 
			u.user_id,
			u.username,
			COUNT(pr.user_id) as assignments_count
		FROM users u
		LEFT JOIN pr_reviewers pr ON u.user_id = pr.user_id
		GROUP BY u.user_id, u.username
		ORDER BY assignments_count DESC, u.username
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get top reviewers: %w", err)
	}
	defer rows.Close()

	var stats []domain.UserAssignmentStat
	for rows.Next() {
		var stat domain.UserAssignmentStat
		if err := rows.Scan(&stat.UserID, &stat.Username, &stat.AssignmentsCount); err != nil {
			return nil, fmt.Errorf("scan reviewer stat: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return stats, nil
}