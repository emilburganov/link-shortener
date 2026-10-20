package link

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")

	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), url)

	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func TestPostgresRepositoryInsert(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	repo := NewPostgresRepository(pool)

	code, err := GenerateCode(12)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM links WHERE code = $1", code)
	})

	maxClicks := int64(3)

	t.Run("stores the link", func(t *testing.T) {
		err := repo.Insert(ctx, code, "https://example.com", &maxClicks)

		if err != nil {
			t.Fatalf("Insert() error: %v", err)
		}

		var (
			url    string
			clicks int64
			limit  *int64
		)

		row := pool.QueryRow(ctx, "SELECT url, clicks, max_clicks FROM links WHERE code = $1", code)
		err = row.Scan(&url, &clicks, &limit)

		if err != nil {
			t.Fatalf("select inserted row: %v", err)
		}

		if url != "https://example.com" || clicks != 0 || limit == nil || *limit != maxClicks {
			t.Errorf("row = (%q, %d, %v), want (example.com, 0, %d)", url, clicks, limit, maxClicks)
		}
	})

	t.Run("duplicate code returns ErrCodeTaken", func(t *testing.T) {
		err := repo.Insert(ctx, code, "https://other.example.com", nil)

		if !errors.Is(err, ErrCodeTaken) {
			t.Errorf("Insert() error = %v, want ErrCodeTaken", err)
		}
	})

	t.Run("other constraint violation is not ErrCodeTaken", func(t *testing.T) {
		err := repo.Insert(ctx, code+"x", "", nil) // пустой url нарушает CHECK

		if err == nil || errors.Is(err, ErrCodeTaken) {
			t.Errorf("Insert() error = %v, want a check-constraint error", err)
		}
	})
}
