package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
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

	return Config{
		Port:        port,
		GinMode:     os.Getenv("GIN_MODE"),
		DatabaseURL: dsn.String(),
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
