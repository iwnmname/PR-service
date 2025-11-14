package user

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

func (r *Repository) Create(ctx context.Context, user *domain.DBUser) error {
	query := `
		INSERT INTO users (user_id, username, team_name, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.getDB(ctx).ExecContext(
		ctx,
		query,
		user.UserID,
		user.Username,
		user.TeamName,
		user.IsActive,
		user.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, userID domain.UserID) (*domain.DBUser, error) {
	query := `
		SELECT user_id, username, team_name, is_active, created_at
		FROM users
		WHERE user_id = $1
	`

	var user domain.DBUser
	err := r.getDB(ctx).QueryRowContext(ctx, query, userID).Scan(
		&user.UserID,
		&user.Username,
		&user.TeamName,
		&user.IsActive,
		&user.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

func (r *Repository) Update(ctx context.Context, user *domain.DBUser) error {
	query := `
		UPDATE users
		SET username = $2, team_name = $3, is_active = $4
		WHERE user_id = $1
	`

	_, err := r.getDB(ctx).ExecContext(
		ctx,
		query,
		user.UserID,
		user.Username,
		user.TeamName,
		user.IsActive,
	)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (r *Repository) SetActive(ctx context.Context, userID domain.UserID, isActive bool) error {
	query := `
		UPDATE users
		SET is_active = $2
		WHERE user_id = $1
	`

	_, err := r.getDB(ctx).ExecContext(ctx, query, userID, isActive)

	if err != nil {
		return fmt.Errorf("set user active: %w", err)
	}

	return nil
}

func (r *Repository) GetByTeam(ctx context.Context, teamName domain.TeamName) ([]*domain.DBUser, error) {
	query := `
		SELECT user_id, username, team_name, is_active, created_at
		FROM users
		WHERE team_name = $1
		ORDER BY user_id
	`

	rows, err := r.getDB(ctx).QueryContext(ctx, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("get users by team: %w", err)
	}
	defer rows.Close()

	var users []*domain.DBUser
	for rows.Next() {
		var user domain.DBUser
		if err := rows.Scan(
			&user.UserID,
			&user.Username,
			&user.TeamName,
			&user.IsActive,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return users, nil
}

func (r *Repository) GetActiveByTeam(ctx context.Context, teamName domain.TeamName) ([]*domain.DBUser, error) {
	query := `
		SELECT user_id, username, team_name, is_active, created_at
		FROM users
		WHERE team_name = $1 AND is_active = true
		ORDER BY user_id
	`

	rows, err := r.getDB(ctx).QueryContext(ctx, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("get active users by team: %w", err)
	}
	defer rows.Close()

	var users []*domain.DBUser
	for rows.Next() {
		var user domain.DBUser
		if err := rows.Scan(
			&user.UserID,
			&user.Username,
			&user.TeamName,
			&user.IsActive,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return users, nil
}

func (r *Repository) Exists(ctx context.Context, userID domain.UserID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)`

	var exists bool
	err := r.getDB(ctx).QueryRowContext(ctx, query, userID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}

	return exists, nil
}
