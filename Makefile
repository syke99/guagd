CONFIG        ?= environment.json
ST_DB_URL     ?= $(shell jq -r '.SUPERTOKENS_DB_URL' $(CONFIG))
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
	for f in migrations/*.sql; do \
		echo "running $$f..."; \
		psql "$(MIGRATION_URL)" -f "$$f" || exit 1; \
	done
	@echo "migrations complete"
