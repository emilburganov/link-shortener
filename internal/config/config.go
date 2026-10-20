package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	BaseURL     string
	GinMode     string
	DatabaseURL string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	var missing []string
	required := func(key string) string {
		v := os.Getenv(key)

		if v == "" {
			missing = append(missing, key)
		}

		return v
	}

	port := required("APP_PORT")
	user := required("POSTGRES_USER")
	password := required("POSTGRES_PASSWORD")
	db := required("POSTGRES_DB")
	sslmode := required("POSTGRES_SSLMODE")

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("required env variables aren`t set: %v", missing)
	}

	host := getenv("POSTGRES_HOST", "postgres")
	dbPort := getenv("POSTGRES_PORT", "5432")

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, dbPort),
		Path:     db,
		RawQuery: url.Values{"sslmode": {sslmode}}.Encode(),
	}

	baseURL, err := parseBaseURL(getenv("BASE_URL", "http://localhost:"+port))

	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:        port,
		BaseURL:     baseURL,
		GinMode:     os.Getenv("GIN_MODE"),
		DatabaseURL: dsn.String(),
	}, nil
}

func parseBaseURL(raw string) (string, error) {
	u, err := url.Parse(raw)

	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("BASE_URL must be an http or https address, got %q", raw)
	}

	return strings.TrimRight(raw, "/"), nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
