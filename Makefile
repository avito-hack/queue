GO := go
NPM := npm
NPX := npx
LINTER := golangci-lint
COMPOSE := docker compose
BIN_DIR := $(CURDIR)/bin

GO_SERVICES := queue tickets avito-adapter

.PHONY: \
	generate generate-queue generate-tickets generate-avito-adapter \
	lint lint-tickets lint-avito-adapter lint-frontend lint-queue \
	test test-tickets test-avito-adapter test-frontend test-queue \
	build build-queue build-tickets build-avito-adapter build-frontend \
	clean up down logs

generate: $(addprefix generate-,$(GO_SERVICES))

generate-queue:
	cd services/queue && $(GO) generate ./...

generate-tickets:
	cd services/tickets && $(GO) generate ./...

generate-avito-adapter:
	cd services/avito-adapter && $(GO) generate ./...

lint: $(addprefix lint-,$(GO_SERVICES)) lint-frontend

lint-tickets: generate-tickets
	cd services/tickets && $(LINTER) run

lint-avito-adapter:
	cd services/avito-adapter && $(LINTER) run

lint-queue:
	$(NPX) --yes @redocly/cli@1.34.5 lint \
		--skip-rule struct \
		--skip-rule no-empty-servers \
		--skip-rule security-defined \
		--skip-rule info-license \
		--skip-rule no-unused-components \
		services/queue/api/openapi.yaml

lint-frontend:
	cd services/frontend && $(NPM) ci && $(NPM) run lint

test: $(addprefix test-,$(GO_SERVICES))

test-tickets: generate-tickets lint-tickets
	cd services/tickets && $(GO) test ./...

test-avito-adapter: generate-avito-adapter lint-avito-adapter
	cd services/avito-adapter && $(GO) test ./...

test-queue: generate-queue lint-queue
	cd services/queue && $(GO) test ./...

test-frontend:
	cd services/frontend && $(NPM) ci && $(NPM) run lint

build: $(addprefix build-,$(GO_SERVICES)) build-frontend

build-tickets: generate-tickets
	mkdir -p $(BIN_DIR)
	cd services/tickets && $(GO) build -o $(BIN_DIR)/tickets ./cmd/app

build-avito-adapter: generate-avito-adapter
	mkdir -p $(BIN_DIR)
	cd services/avito-adapter && $(GO) build -o $(BIN_DIR)/avito-adapter ./cmd/app

build-queue: generate-queue
	mkdir -p $(BIN_DIR)
	cd services/queue && $(GO) build -o $(BIN_DIR)/queue ./cmd/app

build-frontend:
	cd services/frontend && $(NPM) ci && $(NPM) run build

clean:
	rm -rf \
		$(BIN_DIR) \
		services/frontend/dist \
		services/frontend/dist-ssr \
		services/queue/gen \
		services/tickets/gen
	rm -f services/avito-adapter/gen/server/server.gen.go

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f
