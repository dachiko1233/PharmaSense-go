.PHONY: setup db-up db-down db-logs migrate-up migrate-down seed sqlc dev-backend dev-frontend test build

DB_URL ?= postgres://postgres:postgres@localhost:5432/pharmasense?sslmode=disable

setup:
	cd backend && go mod download
	cd frontend && npm install

db-up:
	docker compose up -d postgres
	@echo "Waiting for Postgres..."
	@until docker exec pharmasense-db pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
	@echo "Postgres ready on localhost:5432"

db-down:
	docker compose down

db-logs:
	docker compose logs -f postgres

migrate-up:
	migrate -path backend/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path backend/migrations -database "$(DB_URL)" down

sqlc:
	cd backend && sqlc generate

seed:
	cd backend && go run ./cmd/seed

dev-backend:
	cd backend && go run ./cmd/api

dev-frontend:
	cd frontend && npm run dev

test:
	cd backend && go test ./...

build:
	cd backend && go build -o bin/api ./cmd/api
	cd frontend && npm run build
