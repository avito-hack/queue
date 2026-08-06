.PHONY: generate test run

generate:
	go generate ./...

test:
	$(MAKE) generate
	go test ./...

run:
	$(MAKE) generate
	go run ./cmd/app
