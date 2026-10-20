package link

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	insertErrs []error
	calls      int
	lastCode   string
	lastIn     NewLink
}

func (f *fakeRepo) Insert(_ context.Context, code string, in NewLink) error {
	f.lastCode = code
	f.lastIn = in

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

	now := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)
	earlier := now.Add(-time.Hour)

	tests := []struct {
		name      string
		in        NewLink
		repoErrs  []error
		wantCalls int
		wantErr   error
	}{
		{
			name:      "success on first try",
			in:        NewLink{URL: "https://example.com"},
			wantCalls: 1,
		},
		{
			name:      "success with max clicks",
			in:        NewLink{URL: "https://example.com", MaxClicks: &five},
			wantCalls: 1,
		},
		{
			name:      "success with expiry in the future",
			in:        NewLink{URL: "https://example.com", ExpiresAt: &later},
			wantCalls: 1,
		},
		{
			name:      "retries after collisions",
			in:        NewLink{URL: "https://example.com"},
			repoErrs:  []error{ErrCodeTaken, ErrCodeTaken},
			wantCalls: 3,
		},
		{
			name: "gives up after max attempts",
			in:   NewLink{URL: "https://example.com"},
			repoErrs: []error{
				ErrCodeTaken, ErrCodeTaken, ErrCodeTaken, ErrCodeTaken, ErrCodeTaken,
			},
			wantCalls: maxInsertAttempts,
			wantErr:   ErrCodeTaken,
		},
		{
			name:      "db error is not retried",
			in:        NewLink{URL: "https://example.com"},
			repoErrs:  []error{errDB},
			wantCalls: 1,
			wantErr:   errDB,
		},
		{
			name:    "invalid url never reaches db",
			in:      NewLink{URL: "nope"},
			wantErr: ErrInvalidURL,
		},
		{
			name:    "zero max clicks never reaches db",
			in:      NewLink{URL: "https://example.com", MaxClicks: &zero},
			wantErr: ErrInvalidMaxClicks,
		},
		{
			name:    "negative max clicks never reaches db",
			in:      NewLink{URL: "https://example.com", MaxClicks: &negative},
			wantErr: ErrInvalidMaxClicks,
		},
		{
			name:    "expiry in the past never reaches db",
			in:      NewLink{URL: "https://example.com", ExpiresAt: &earlier},
			wantErr: ErrInvalidExpiresAt,
		},
		{
			name:    "expiry equal to now never reaches db",
			in:      NewLink{URL: "https://example.com", ExpiresAt: &now},
			wantErr: ErrInvalidExpiresAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{insertErrs: tt.repoErrs}
			svc := NewService(repo)
			svc.now = func() time.Time { return now }

			code, err := svc.Create(context.Background(), tt.in)

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

			if repo.lastIn != tt.in {
				t.Errorf("repository got %+v, want %+v", repo.lastIn, tt.in)
			}
		})
	}
}
