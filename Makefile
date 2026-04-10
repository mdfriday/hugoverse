.PHONY: help build build-docker up down restart logs clean test test-local verify-local

# Default target
help:
	@echo "Hugoverse - Self-hosted Obsidian Sync & Publish Platform"
	@echo ""
	@echo "🔧 Development:"
	@echo "  make build         - Build Go binary"
	@echo "  make test          - Run Go tests"
	@echo "  make verify-local  - Verify local environment (no Docker start)"
	@echo "  make test-local    - Full local Docker test (with service start)"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make build-docker  - Build Docker images"
	@echo "  make up            - Start all services (production)"
	@echo "  make up-local      - Start all services (local dev)"
	@echo "  make down          - Stop all services"
	@echo "  make restart       - Restart all services"
	@echo "  make logs          - View logs"
	@echo "  make clean         - Clean up containers and volumes"
	@echo ""
	@echo "📦 Release:"
	@echo "  make release       - Build and push Docker images"
	@echo ""

# Build Go binary
build:
	@echo "🔨 Building Hugoverse binary..."
	CGO_ENABLED=1 go build -o bin/hugoverse ./cmd/hugoverse
	@echo "✅ Build complete: bin/hugoverse"

# Build Docker images
build-docker:
	@echo "🐳 Building Docker images..."
	docker-compose build
	@echo "✅ Docker images built"

# Start services
up:
	@echo "🚀 Starting services..."
	docker-compose up -d
	@echo "✅ Services started"
	@echo ""
	@echo "View logs: make logs"
	@echo "Stop: make down"

# Stop services
down:
	@echo "🛑 Stopping services..."
	docker-compose down
	@echo "✅ Services stopped"

# Restart services
restart:
	@echo "🔄 Restarting services..."
	docker-compose restart
	@echo "✅ Services restarted"

# View logs
logs:
	docker-compose logs -f

# Clean up
clean:
	@echo "🧹 Cleaning up..."
	docker-compose down -v
	rm -rf bin/
	@echo "✅ Cleanup complete"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...
	@echo "✅ Tests complete"

# Verify local environment (no Docker start)
verify-local:
	@echo "🔍 Verifying local environment..."
	@bash verify-local.sh

# Full local Docker test
test-local:
	@echo "🧪 Running full local Docker test..."
	@bash test-docker-local.sh

# Start services (local dev with .env.local)
up-local:
	@echo "🚀 Starting services (local dev mode)..."
	@if [ ! -f .env.local ]; then \
		echo "⚠️  .env.local not found, creating from .env.example..."; \
		cp .env.example .env.local; \
		sed -i.bak 's/DOMAIN=.*/DOMAIN=localhost/' .env.local; \
		sed -i.bak 's/HTTP_PORT=80/HTTP_PORT=8080/' .env.local; \
		sed -i.bak 's/HTTPS_PORT=443/HTTPS_PORT=8443/' .env.local; \
		rm -f .env.local.bak; \
	fi
	docker-compose --env-file .env.local up -d
	@echo "✅ Services started"
	@echo ""
	@echo "📍 Access at: http://localhost:8080/admin"
	@echo "📋 View logs: make logs-local"

# View logs (local)
logs-local:
	docker-compose --env-file .env.local logs -f

# Stop services (local)
down-local:
	docker-compose --env-file .env.local down

# Clean (local)
clean-local:
	docker-compose --env-file .env.local down -v
	rm -rf .env.local

# Release: build and push images
release:
	@echo "📦 Building release images..."
	@read -p "Enter version (e.g., 2.0.0): " VERSION; \
	echo "Building version $$VERSION..."; \
	docker build -t mdfriday/hugoverse:$$VERSION -t mdfriday/hugoverse:latest -f docker/hugoverse/Dockerfile .; \
	docker build -t mdfriday/hugoverse-caddy:$$VERSION -t mdfriday/hugoverse-caddy:latest -f docker/caddy/Dockerfile docker/caddy/; \
	echo "Pushing to Docker Hub..."; \
	docker push mdfriday/hugoverse:$$VERSION; \
	docker push mdfriday/hugoverse:latest; \
	docker push mdfriday/hugoverse-caddy:$$VERSION; \
	docker push mdfriday/hugoverse-caddy:latest; \
	echo "✅ Release $$VERSION published"

# Install
install:
	@echo "📦 Installing Hugoverse..."
	@bash install.sh

# Generate license (example)
license:
	@echo "🔑 Generating license..."
	@echo "Example usage:"
	@echo '  docker-compose exec hugoverse /app/hugoverse license generate \'
	@echo '    -email admin@example.com \'
	@echo '    -password yourpassword \'
	@echo '    -plan enterprise \'
	@echo '    -count 1'
