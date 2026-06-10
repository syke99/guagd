CONFIG     ?= environment.json
DB_URL     ?= $(shell jq -r '.DATABASE_URL' $(CONFIG))
ST_CONTAINER = guagd-supertokens

.PHONY: dev app supertokens stop logs

dev: supertokens app

supertokens:
	@if docker ps -q -f name=$(ST_CONTAINER) | grep -q .; then \
		echo "supertokens already running"; \
	else \
		docker run -d --name $(ST_CONTAINER) \
			-p 3567:3567 \
			-e POSTGRESQL_CONNECTION_URI="$(DB_URL)" \
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
