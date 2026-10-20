package link

import (
	"context"
	"errors"
	"fmt"
)

const (
	codeLength        = 7
	maxInsertAttempts = 5
)

var (
	ErrCodeTaken        = errors.New("code already taken")
	ErrInvalidMaxClicks = errors.New("max_clicks must be positive")
	ErrNotFound         = errors.New("link not found")
)

type Repository interface {
	Insert(ctx context.Context, code, url string, maxClicks *int64) error
	Visit(ctx context.Context, code string) (string, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, rawURL string, maxClicks *int64) (string, error) {
	if err := validateURL(rawURL); err != nil {
		return "", err
	}

	if maxClicks != nil && *maxClicks <= 0 {
		return "", ErrInvalidMaxClicks
	}

	for range maxInsertAttempts {
		code, err := GenerateCode(codeLength)

		if err != nil {
			return "", err
		}

		err = s.repo.Insert(ctx, code, rawURL, maxClicks)

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
