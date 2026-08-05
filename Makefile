GO := go
LINTER := golangci-lint
COMPOSE := docker compose

SERVICES := \
	services/queue \
	services/tickets \
	services/queue-gateway

.PHONY: generate lint test up down logs

generate:
	@for service in $(SERVICES); do \
		(cd $$service && $(GO) generate ./...); \
	done

lint:
	@for service in $(SERVICES); do \
		(cd $$service && $(LINTER) run); \
	done

test:
	@for service in $(SERVICES); do \
		(cd $$service && $(GO) test ./...); \
	done

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f