package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emilburganov/link-shortener/internal/link"
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

func TestRepositoryInsert(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	repo := NewRepository(pool)

	code, err := link.GenerateCode(12)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM links WHERE code = $1", code)
	})

	maxClicks := int64(3)
	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)

	t.Run("stores the link", func(t *testing.T) {
		err := repo.Insert(ctx, code, link.NewLink{
			URL:       "https://example.com",
			MaxClicks: &maxClicks,
			ExpiresAt: &expiresAt,
		})

		if err != nil {
			t.Fatalf("Insert() error: %v", err)
		}

		var (
			url     string
			clicks  int64
			limit   *int64
			expires *time.Time
		)

		row := pool.QueryRow(
			ctx, "SELECT url, clicks, max_clicks, expires_at FROM links WHERE code = $1", code,
		)
		err = row.Scan(&url, &clicks, &limit, &expires)

		if err != nil {
			t.Fatalf("select inserted row: %v", err)
		}

		if url != "https://example.com" || clicks != 0 {
			t.Errorf("url and clicks = (%q, %d), want (example.com, 0)", url, clicks)
		}

		if limit == nil || *limit != maxClicks {
			t.Errorf("max_clicks = %v, want %d", limit, maxClicks)
		}

		if expires == nil || !expires.Equal(expiresAt) {
			t.Errorf("expires_at = %v, want %v", expires, expiresAt)
		}
	})

	t.Run("duplicate code returns ErrCodeTaken", func(t *testing.T) {
		err := repo.Insert(ctx, code, link.NewLink{URL: "https://other.example.com"})

		if !errors.Is(err, link.ErrCodeTaken) {
			t.Errorf("Insert() error = %v, want ErrCodeTaken", err)
		}
	})

	t.Run("other constraint violation is not ErrCodeTaken", func(t *testing.T) {
		err := repo.Insert(ctx, code+"x", link.NewLink{})

		if err == nil || errors.Is(err, link.ErrCodeTaken) {
			t.Errorf("Insert() error = %v, want a check-constraint error", err)
		}
	})
}

func seedLink(t *testing.T, pool *pgxpool.Pool, maxClicks *int64, expiresAt *time.Time) string {
	t.Helper()

	code, err := link.GenerateCode(12)

	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(
		context.Background(),
		"INSERT INTO links (code, url, max_clicks, expires_at) VALUES ($1, $2, $3, $4)",
		code, "https://example.com/target", maxClicks, expiresAt,
	)

	if err != nil {
		t.Fatalf("seed link: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM links WHERE code = $1", code)
	})

	return code
}

func clicksOf(t *testing.T, pool *pgxpool.Pool, code string) int64 {
	t.Helper()

	var clicks int64
	row := pool.QueryRow(context.Background(), "SELECT clicks FROM links WHERE code = $1", code)

	if err := row.Scan(&clicks); err != nil {
		t.Fatalf("select clicks: %v", err)
	}

	return clicks
}

func TestRepositoryVisit(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	repo := NewRepository(pool)

	two := int64(2)
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name        string
		maxClicks   *int64
		expiresAt   *time.Time
		visits      int
		wantVisited int
		wantErr     error
	}{
		{
			name:        "unlimited link",
			visits:      3,
			wantVisited: 3,
		},
		{
			name:        "stops at the click limit",
			maxClicks:   &two,
			visits:      5,
			wantVisited: 2,
			wantErr:     link.ErrGone,
		},
		{
			name:        "expired link",
			expiresAt:   &past,
			visits:      1,
			wantVisited: 0,
			wantErr:     link.ErrGone,
		},
		{
			name:        "not yet expired link",
			expiresAt:   &future,
			visits:      1,
			wantVisited: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := seedLink(t, pool, tt.maxClicks, tt.expiresAt)
			visited := 0

			for range tt.visits {
				url, err := repo.Visit(ctx, code)

				if err != nil && tt.wantErr != nil && errors.Is(err, tt.wantErr) {
					continue
				}

				if err != nil {
					t.Fatalf("Visit() error = %v, want %v", err, tt.wantErr)
				}

				if url != "https://example.com/target" {
					t.Errorf("Visit() url = %q", url)
				}

				visited++
			}

			if visited != tt.wantVisited {
				t.Errorf("successful visits = %d, want %d", visited, tt.wantVisited)
			}

			if got := clicksOf(t, pool, code); got != int64(tt.wantVisited) {
				t.Errorf("clicks in db = %d, want %d", got, tt.wantVisited)
			}
		})
	}

	t.Run("unknown code", func(t *testing.T) {
		_, err := repo.Visit(ctx, "doesnotexist")

		if !errors.Is(err, link.ErrNotFound) {
			t.Errorf("Visit() error = %v, want ErrNotFound", err)
		}
	})
}

func TestRepositoryVisitConcurrent(t *testing.T) {
	const (
		limit   = 5
		workers = 30
	)

	pool := newTestPool(t)
	repo := NewRepository(pool)
	maxClicks := int64(limit)
	code := seedLink(t, pool, &maxClicks, nil)

	var (
		wg         sync.WaitGroup
		visited    atomic.Int32
		unexpected atomic.Int32
	)

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := repo.Visit(context.Background(), code)

			switch {
			case err == nil:
				visited.Add(1)
			case !errors.Is(err, link.ErrGone):
				unexpected.Add(1)
			}
		}()
	}

	wg.Wait()

	if got := visited.Load(); got != limit {
		t.Errorf("successful visits = %d, want exactly %d", got, limit)
	}

	if got := unexpected.Load(); got != 0 {
		t.Errorf("%d visits failed with an error other than ErrGone", got)
	}

	if got := clicksOf(t, pool, code); got != limit {
		t.Errorf("clicks in db = %d, want %d", got, limit)
	}
}
