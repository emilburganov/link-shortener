package link

import (
	"context"
	"errors"

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
