run-api:
	cd services/api && go run cmd/app/main.go

run-ai:
	cd services/ai && python -m app.main

test-api:
	cd services/api && go test ./...

build-api:
	cd services/api && go build ./cmd/app

.PHONY: run-api run-ai test-api build-api
