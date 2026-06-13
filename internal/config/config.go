// Package config loads runtime configuration from the environment.
package config

import "os"

// Config holds the server's runtime configuration.
type Config struct {
	// HTTPAddr is the listen address for the HTTP API.
	HTTPAddr string
	// DatabaseURL is the Postgres connection string (pgx/libpq DSN or URL).
	DatabaseURL string
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	addr := os.Getenv("LIBRARY_HTTP_ADDR")
	if addr == "" {
		if port := os.Getenv("PORT"); port != "" {
			addr = ":" + port
		} else {
			addr = ":8080"
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://library:library@localhost:5432/library?sslmode=disable"
	}

	return Config{HTTPAddr: addr, DatabaseURL: dbURL}
}
