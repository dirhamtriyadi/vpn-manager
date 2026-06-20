SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

BACKEND_DIR := backend
FRONTEND_DIR := frontend
FRONTEND_HOST ?= 0.0.0.0
FRONTEND_PORT ?= 5173

# Use whichever Node package manager is installed (pnpm preferred, npm fallback)
# so the frontend targets work regardless of the developer's setup. Override with
# `make PKG=npm <target>`.
PKG ?= $(shell command -v pnpm >/dev/null 2>&1 && echo pnpm || echo npm)
# swag is installed by `go install` into $(go env GOPATH)/bin, which is often not
# on PATH; resolve it explicitly and auto-install on demand.
GOPATH_BIN := $(shell go env GOPATH)/bin
SWAG_VERSION := v1.16.3

.PHONY: help env install setup dev dev-db backend frontend db-up db-down docker-up docker-down docker-logs backend-run backend-test backend-build backend-tidy backend-swag frontend-dev frontend-install frontend-lint frontend-build frontend-preview lint test build clean status

help: ## Tampilkan daftar command
	@awk 'BEGIN {FS = ":.*##"; printf "\nVPN Manager Makefile\n\nUsage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\nContoh:\n  make dev              # jalankan backend + frontend sekaligus\n  make backend          # jalankan backend saja\n  make frontend         # jalankan frontend saja\n  make docker-up        # jalankan full-stack (DB + API + frontend) via Docker\n\n"

env: ## Buat file .env dari .env.example jika belum ada
	@if [ ! -f $(BACKEND_DIR)/.env ] && [ -f $(BACKEND_DIR)/.env.example ]; then cp $(BACKEND_DIR)/.env.example $(BACKEND_DIR)/.env; echo "created $(BACKEND_DIR)/.env"; fi
	@if [ ! -f $(FRONTEND_DIR)/.env ] && [ -f $(FRONTEND_DIR)/.env.example ]; then cp $(FRONTEND_DIR)/.env.example $(FRONTEND_DIR)/.env; echo "created $(FRONTEND_DIR)/.env"; fi

install: frontend-install ## Install dependency frontend

setup: env install ## Setup awal monorepo untuk development

dev: env ## Jalankan backend + frontend sekaligus dari root
	@echo "Starting backend and frontend. Press Ctrl+C to stop both."
	@set -e; \
	(cd $(BACKEND_DIR) && go run main.go) & backend_pid=$$!; \
	(cd $(FRONTEND_DIR) && $(PKG) run dev -- --host $(FRONTEND_HOST) --port $(FRONTEND_PORT)) & frontend_pid=$$!; \
	trap 'echo "Stopping..."; kill $$backend_pid $$frontend_pid 2>/dev/null || true; wait $$backend_pid $$frontend_pid 2>/dev/null || true' INT TERM EXIT; \
	wait -n $$backend_pid $$frontend_pid

dev-db: db-up dev ## Jalankan DB compose, lalu backend + frontend

backend: backend-run ## Jalankan backend saja

frontend: frontend-dev ## Jalankan frontend saja

db-up: ## Jalankan database Postgres dari backend/docker-compose.yml
	@cd $(BACKEND_DIR) && docker compose up -d db

db-down: ## Matikan database compose
	@cd $(BACKEND_DIR) && docker compose down

docker-up: ## Jalankan full-stack (DB + API + frontend) via Docker Compose
	@cd $(BACKEND_DIR) && docker compose up --build

docker-down: ## Matikan Docker Compose
	@cd $(BACKEND_DIR) && docker compose down

docker-logs: ## Lihat log Docker Compose
	@cd $(BACKEND_DIR) && docker compose logs -f

backend-run: env ## Jalankan Go backend lokal; gunakan sudo jika perlu CAP_NET_ADMIN
	@cd $(BACKEND_DIR) && go run main.go

backend-test: ## Jalankan test backend
	@cd $(BACKEND_DIR) && go test ./...

backend-build: ## Build binary backend
	@cd $(BACKEND_DIR) && go build -o wg-panel ./main.go

backend-tidy: ## Jalankan go mod tidy
	@cd $(BACKEND_DIR) && go mod tidy

backend-swag: ## Generate Swagger docs backend (auto-install swag jika belum ada)
	@if ! command -v swag >/dev/null 2>&1 && [ ! -x "$(GOPATH_BIN)/swag" ]; then \
		echo "swag not found; installing $(SWAG_VERSION)..."; \
		go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION); \
	fi; \
	SWAG=$$(command -v swag || echo "$(GOPATH_BIN)/swag"); \
	cd $(BACKEND_DIR) && "$$SWAG" init -g main.go -o docs

frontend-dev: env ## Jalankan Vite dev server
	@cd $(FRONTEND_DIR) && $(PKG) run dev -- --host $(FRONTEND_HOST) --port $(FRONTEND_PORT)

frontend-install: ## Install dependency frontend ($(PKG))
	@cd $(FRONTEND_DIR) && $(PKG) install

frontend-lint: ## Type-check frontend
	@cd $(FRONTEND_DIR) && $(PKG) run lint

frontend-build: ## Build frontend production
	@cd $(FRONTEND_DIR) && $(PKG) run build

frontend-preview: ## Preview hasil build frontend
	@cd $(FRONTEND_DIR) && $(PKG) run preview -- --host $(FRONTEND_HOST) --port $(FRONTEND_PORT)

lint: frontend-lint ## Jalankan lint/type-check semua yang tersedia

test: backend-test ## Jalankan test semua yang tersedia

build: backend-build frontend-build ## Build backend dan frontend

clean: ## Bersihkan artifact build umum
	@rm -rf $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/tsconfig.tsbuildinfo $(BACKEND_DIR)/wg-panel $(BACKEND_DIR)/bin

status: ## Tampilkan status command penting
	@printf "go:      "; command -v go || true
	@printf "node:    "; command -v node || true
	@printf "pnpm:    "; command -v pnpm || true
	@printf "npm:     "; command -v npm || true
	@printf "pkg mgr: %s\n" "$(PKG)"
	@printf "docker:  "; command -v docker || true
	@printf "swag:    "; command -v swag || ([ -x "$(GOPATH_BIN)/swag" ] && echo "$(GOPATH_BIN)/swag") || echo "(not installed; 'make backend-swag' installs it)"
