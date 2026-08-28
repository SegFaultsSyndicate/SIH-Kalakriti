# services/channel-svc/Makefile

.PHONY: build test lint generate

build:
	go build -o bin/channel-svc ./cmd/channel-svc

test:
	go test ./...

lint:
	golangci-lint run ./...

generate:
	# sqlc generates queries
	sqlc generate -f ../../sqlc.yaml

# Example: export IndiaHandmade catalog
# make export-indiahandmade FORMAT=json OUTPUT=catalog.json
export-indiahandmade:
	@echo "Export requires running service with catalog data"
	@echo "Call GET /export/indiahandmade?format=json|csv"