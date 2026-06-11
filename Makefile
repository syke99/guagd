CONFIG        ?= environment.json
ST_DB_URL     ?= $(shell jq -r '.SUPERTOKENS_DB_URL' $(CONFIG))
ST_API_KEY    ?= $(shell jq -r '.SUPERTOKENS_API_KEY' $(CONFIG))
MIGRATION_URL ?= $(shell jq -r '.MIGRATION_URL' $(CONFIG))
ST_CONTAINER   = guagd-supertokens

.PHONY: dev app supertokens stop logs migrate

dev: supertokens app

supertokens:
	@if docker ps -q -f name=$(ST_CONTAINER) | grep -q .; then \
		echo "supertokens already running"; \
	else \
		docker run -d --name $(ST_CONTAINER) \
			-p 3567:3567 \
			-e POSTGRESQL_CONNECTION_URI="$(ST_DB_URL)" \
			-e API_KEYS="$(ST_API_KEY)" \
			supertokens/supertokens-postgresql:latest; \
	fi
	@echo "waiting for supertokens..." && until curl -sf http://localhost:3567/hello >/dev/null; do sleep 1; done && echo "supertokens ready"

app:
	go run ./cmd/main.go --config $(CONFIG)

stop:
	-docker stop $(ST_CONTAINER)
	-docker rm $(ST_CONTAINER)

logs:
	docker logs -f $(ST_CONTAINER)

migrate:
	@export PATH="/opt/homebrew/opt/libpq/bin:$$PATH"; \
	psql "$(MIGRATION_URL)" -c "CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now());" > /dev/null; \
	for f in migrations/*.sql; do \
		name=$$(basename $$f); \
		already=$$(psql "$(MIGRATION_URL)" -tAc "SELECT COUNT(*) FROM schema_migrations WHERE filename = '$$name'"); \
		if [ "$$already" = "0" ]; then \
			echo "running $$f..."; \
			psql "$(MIGRATION_URL)" -f "$$f" || exit 1; \
			psql "$(MIGRATION_URL)" -c "INSERT INTO schema_migrations (filename) VALUES ('$$name');" > /dev/null; \
		else \
			echo "skipping $$f (already applied)"; \
		fi; \
	done
	@echo "migrations complete"
