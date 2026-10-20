package link

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	codeLength        = 7
	maxInsertAttempts = 5
)

var (
	ErrCodeTaken        = errors.New("code already taken")
	ErrInvalidMaxClicks = errors.New("max_clicks must be positive")
	ErrInvalidExpiresAt = errors.New("expires_at must be in the future")
	ErrNotFound         = errors.New("link not found")
	ErrGone             = errors.New("link is no longer available")
)

type NewLink struct {
	URL       string
	MaxClicks *int64
	ExpiresAt *time.Time
}

type Repository interface {
	Insert(ctx context.Context, code string, in NewLink) error
	Visit(ctx context.Context, code string) (string, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) Create(ctx context.Context, in NewLink) (string, error) {
	if err := validateURL(in.URL); err != nil {
		return "", err
	}

	if in.MaxClicks != nil && *in.MaxClicks <= 0 {
		return "", ErrInvalidMaxClicks
	}

	if in.ExpiresAt != nil && !in.ExpiresAt.After(s.now()) {
		return "", ErrInvalidExpiresAt
	}

	for range maxInsertAttempts {
		code, err := GenerateCode(codeLength)

		if err != nil {
			return "", err
		}

		err = s.repo.Insert(ctx, code, in)

		if errors.Is(err, ErrCodeTaken) {
			continue
		}

		if err != nil {
			return "", fmt.Errorf("insert link: %w", err)
		}

		return code, nil
	}

	return "", fmt.Errorf(
		"no unique code after %d attempts: %w", maxInsertAttempts, ErrCodeTaken,
	)
}

func (s *Service) Visit(ctx context.Context, code string) (string, error) {
	return s.repo.Visit(ctx, code)
}
