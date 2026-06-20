.PHONY: run build tidy swag docker-up docker-down

run:
	go run main.go

build:
	go build -o bin/go-api main.go

tidy:
	go mod tidy

swag:
	@if ! command -v swag >/dev/null 2>&1 && [ ! -x "$$(go env GOPATH)/bin/swag" ]; then \
		echo "swag not found; installing v1.16.3..."; \
		go install github.com/swaggo/swag/cmd/swag@v1.16.3; \
	fi; \
	SWAG=$$(command -v swag || echo "$$(go env GOPATH)/bin/swag"); \
	"$$SWAG" init -g main.go -o docs

docker-up:
	docker compose up -d

docker-down:
	docker compose down
