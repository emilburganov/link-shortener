package link

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Insert(
	ctx context.Context, code, url string, maxClicks *int64,
) error {
	_, err := r.pool.Exec(ctx, insertQuery, code, url, maxClicks)

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation &&
		pgErr.ConstraintName == codeConstraint {
		return ErrCodeTaken
	}

	return err
}

func (r *PostgresRepository) Visit(ctx context.Context, code string) (string, error) {
	var url string

	err := r.pool.QueryRow(ctx, visitQuery, code).Scan(&url)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}

	if err != nil {
		return "", fmt.Errorf("visit link: %w", err)
	}

	return url, nil
}
