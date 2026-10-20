package link

import (
	"errors"
	"net/url"
)

const maxURLLength = 2048

var ErrInvalidURL = errors.New("url must be a valid http or https address")

func validateURL(raw string) error {
	if raw == "" || len(raw) > maxURLLength {
		return ErrInvalidURL
	}

	u, err := url.Parse(raw)

	if err != nil || u.Host == "" {
		return ErrInvalidURL
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidURL
	}

	return nil
}
