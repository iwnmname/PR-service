package team

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

func (r *Repository) Create(ctx context.Context, team *domain.DBTeam) error {
	query := `
		INSERT INTO teams (team_name, created_at)
		VALUES ($1, $2)
	`

	_, err := r.getDB(ctx).ExecContext(ctx, query, team.TeamName, team.CreatedAt)

	if err != nil {
		return fmt.Errorf("create team: %w", err)
	}

	return nil
}

func (r *Repository) GetByName(ctx context.Context, teamName domain.TeamName) (*domain.DBTeam, error) {
	query := `
		SELECT team_name, created_at
		FROM teams
		WHERE team_name = $1
	`

	var team domain.DBTeam
	err := r.getDB(ctx).QueryRowContext(ctx, query, teamName).Scan(
		&team.TeamName,
		&team.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get team by name: %w", err)
	}

	return &team, nil
}

func (r *Repository) Exists(ctx context.Context, teamName domain.TeamName) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`

	var exists bool
	err := r.getDB(ctx).QueryRowContext(ctx, query, teamName).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("check team exists: %w", err)
	}

	return exists, nil
}
