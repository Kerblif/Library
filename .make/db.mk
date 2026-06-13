# Local Postgres lifecycle and goose migrations.

DATABASE_URL   ?= postgres://library:library@localhost:5432/library?sslmode=disable
MIGRATIONS_DIR := sql/migrations

GOOSE := go tool -modfile=./tools/go.mod goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)"

.PHONY: db-up db-down db-reset db-init migrate-up migrate-down migrate-status

# Start the local Postgres container and wait until it reports healthy.
db-up:
	docker compose up -d --wait postgres

# Stop and remove the container (the data volume is kept).
db-down:
	docker compose down

# Drop the data volume and start a clean Postgres.
db-reset:
	docker compose down -v
	$(MAKE) db-up

# Bring Postgres up and apply every migration.
db-init: db-up migrate-up

# Apply all pending migrations.
migrate-up:
	$(GOOSE) up

# Roll back the most recent migration.
migrate-down:
	$(GOOSE) down

# Print the migration status table.
migrate-status:
	$(GOOSE) status
