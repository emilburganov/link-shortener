package link

import (
	"context"
	"errors"
	"testing"
)

type fakeRepo struct {
	insertErrs []error
	calls      int
	lastCode   string
}

func (f *fakeRepo) Insert(_ context.Context, code, _ string, _ *int64) error {
	f.lastCode = code

	var err error

	if f.calls < len(f.insertErrs) {
		err = f.insertErrs[f.calls]
	}

	f.calls++

	return err
}

func (f *fakeRepo) Visit(context.Context, string) (string, error) {
	return "", ErrNotFound
}

func TestServiceCreate(t *testing.T) {
	errDB := errors.New("db down")
	zero := int64(0)
	negative := int64(-5)
	five := int64(5)

	tests := []struct {
		name      string
		url       string
		maxClicks *int64
		repoErrs  []error
		wantCalls int
		wantErr   error
	}{
		{
			name:      "success on first try",
			url:       "https://example.com",
			wantCalls: 1,
		},
		{
			name:      "success with max clicks",
			url:       "https://example.com",
			maxClicks: &five,
			wantCalls: 1,
		},
		{
			name:      "retries after collisions",
			url:       "https://example.com",
			repoErrs:  []error{ErrCodeTaken, ErrCodeTaken},
			wantCalls: 3,
		},
		{
			name: "gives up after max attempts",
			url:  "https://example.com",
			repoErrs: []error{
				ErrCodeTaken, ErrCodeTaken, ErrCodeTaken, ErrCodeTaken, ErrCodeTaken,
			},
			wantCalls: maxInsertAttempts,
			wantErr:   ErrCodeTaken,
		},
		{
			name:      "db error is not retried",
			url:       "https://example.com",
			repoErrs:  []error{errDB},
			wantCalls: 1,
			wantErr:   errDB,
		},
		{
			name:    "invalid url never reaches db",
			url:     "nope",
			wantErr: ErrInvalidURL,
		},
		{
			name:      "zero max clicks never reaches db",
			url:       "https://example.com",
			maxClicks: &zero,
			wantErr:   ErrInvalidMaxClicks,
		},
		{
			name:      "negative max clicks never reaches db",
			url:       "https://example.com",
			maxClicks: &negative,
			wantErr:   ErrInvalidMaxClicks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{insertErrs: tt.repoErrs}
			svc := NewService(repo)

			code, err := svc.Create(context.Background(), tt.url, tt.maxClicks)

			if repo.calls != tt.wantCalls {
				t.Errorf("Insert called %d times, want %d", repo.calls, tt.wantCalls)
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(code) != codeLength {
				t.Errorf("code %q has length %d, want %d", code, len(code), codeLength)
			}

			if code != repo.lastCode {
				t.Errorf("returned code %q is not the one inserted last (%q)", code, repo.lastCode)
			}
		})
	}
}
