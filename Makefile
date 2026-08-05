GO := go
NPM := npm
NPX := npx
LINTER := golangci-lint
COMPOSE := docker compose
BIN_DIR := $(CURDIR)/bin

GO_SERVICES := tickets avito-adapter

.PHONY: \
	generate generate-tickets generate-avito-adapter \
	lint lint-tickets lint-avito-adapter lint-frontend lint-queue \
	test test-tickets test-avito-adapter test-frontend test-queue \
	build build-tickets build-avito-adapter build-frontend \
	up down logs

generate: $(addprefix generate-,$(GO_SERVICES))

generate-tickets:
	cd services/tickets && $(GO) generate ./...

generate-avito-adapter:
	cd services/avito-adapter && $(GO) generate ./...

lint: lint-tickets lint-avito-adapter lint-frontend lint-queue

lint-tickets:
	cd services/tickets && $(LINTER) run

lint-avito-adapter:
	cd services/avito-adapter && $(LINTER) run

lint-frontend:
	cd services/frontend && $(NPM) ci && $(NPM) run lint

lint-queue:
	$(NPX) --yes @redocly/cli@1.34.5 lint \
		--skip-rule struct \
		--skip-rule no-empty-servers \
		--skip-rule security-defined \
		--skip-rule info-license \
		--skip-rule no-unused-components \
		services/queue/api/openapi.yaml

test: test-tickets test-avito-adapter test-frontend test-queue

test-tickets:
	cd services/tickets && $(GO) test ./...

test-avito-adapter:
	cd services/avito-adapter && $(GO) test ./...

test-frontend:
	cd services/frontend && $(NPM) ci && $(NPM) run lint

test-queue: lint-queue

build: $(addprefix build-,$(GO_SERVICES)) build-frontend

build-tickets:
	mkdir -p $(BIN_DIR)
	cd services/tickets && $(GO) build -o $(BIN_DIR)/tickets ./cmd/app

build-avito-adapter:
	mkdir -p $(BIN_DIR)
	cd services/avito-adapter && $(GO) build -o $(BIN_DIR)/avito-adapter ./cmd/app

build-frontend:
	cd services/frontend && $(NPM) ci && $(NPM) run build

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f
