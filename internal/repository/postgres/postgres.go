package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emilburganov/link-shortener/internal/link"
)

const (
	uniqueViolation = "23505"
	codeConstraint  = "links_code_key"
)

const insertQuery = `
	INSERT INTO links (code, url, max_clicks)
	VALUES ($1, $2, $3)`

const visitQuery = `
	UPDATE links
	SET clicks = clicks + 1
	WHERE code = $1
		AND (expires_at IS NULL OR expires_at > now())
		AND (max_clicks IS NULL OR clicks < max_clicks)
	RETURNING url`

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Insert(
	ctx context.Context, code, url string, maxClicks *int64,
) error {
	_, err := r.pool.Exec(ctx, insertQuery, code, url, maxClicks)

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation &&
		pgErr.ConstraintName == codeConstraint {
		return link.ErrCodeTaken
	}

	return err
}

func (r *Repository) Visit(ctx context.Context, code string) (string, error) {
	var url string

	err := r.pool.QueryRow(ctx, visitQuery, code).Scan(&url)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", link.ErrNotFound
	}

	if err != nil {
		return "", fmt.Errorf("visit link: %w", err)
	}

	return url, nil
}
