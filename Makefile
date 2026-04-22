# Makefile para o projeto Comissionamento.

DB_HOST ?= localhost
DB_PORT ?= 5433
DB_USER ?= comissionamento
DB_PASS ?= comissionamento
DB_NAME ?= comissionamento

DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# ============================================================
# Desenvolvimento
# ============================================================

.PHONY: dev
dev: db-up migrate
	@echo "🚀 Iniciando desenvolvimento..."
	@echo "   Backend Go:  http://localhost:8080"
	@echo "   Frontend:    http://localhost:5173"
	@echo ""
	@echo "Use 'make dev-api' e 'make dev-web' em terminais separados."

.PHONY: db-up
db-up:
	docker compose up -d postgres
	@echo "⏳ Aguardando PostgreSQL ficar pronto..."
	@until docker compose exec -T postgres pg_isready -U $(DB_USER) > /dev/null 2>&1; do sleep 1; done
	@echo "✅ PostgreSQL pronto!"

.PHONY: dev-api
dev-api:
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/server

.PHONY: dev-web
dev-web:
	cd web && npm run dev

# ============================================================
# Migrations
# ============================================================

.PHONY: migrate
migrate: db-up
	@echo "📦 Aplicando migrations..."
	@for f in migrations/*.up.sql; do \
		echo "  → $$f"; \
		docker compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f /dev/stdin < $$f 2>&1 || true; \
	done
	@echo "✅ Migrations aplicadas!"

.PHONY: migrate-down
migrate-down:
	@echo "⚠️  Desfazendo migrations..."
	@for f in $$(ls -r migrations/*.down.sql); do \
		echo "  ← $$f"; \
		docker compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f /dev/stdin < $$f 2>&1 || true; \
	done
	@echo "✅ Migrations desfeitas!"

.PHONY: migrate-reset
migrate-reset: migrate-down migrate
	@echo "🔄 Banco resetado!"

# ============================================================
# Build
# ============================================================

.PHONY: build
build: build-web
	go build -o bin/server ./cmd/server

.PHONY: build-web
build-web:
	cd web && npm run build

# ============================================================
# Testes
# ============================================================

.PHONY: test
test:
	go test ./... -v -cover

.PHONY: test-web
test-web:
	cd web && npm test

# ============================================================
# Limpeza
# ============================================================

.PHONY: clean
clean:
	rm -rf bin/
	cd web && rm -rf dist/

.PHONY: clean-all
clean-all: clean
	docker compose down -v
