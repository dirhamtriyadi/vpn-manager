.PHONY: run build tidy swag docker-up docker-down

run:
	go run main.go

build:
	go build -o bin/go-api main.go

tidy:
	go mod tidy

swag:
	swag init -g main.go -o docs

docker-up:
	docker compose up -d

docker-down:
	docker compose down
