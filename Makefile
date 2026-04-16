.PHONY: help build build-docker up down restart logs clean test test-local verify-local \
        docker-login docker-build-caddy docker-build-hugoverse docker-build-all \
        docker-push-caddy docker-push-hugoverse docker-push-all release \
        aliyun-login aliyun-pull-couchdb aliyun-tag-couchdb aliyun-tag-caddy aliyun-tag-hugoverse aliyun-tag-all \
        aliyun-push-couchdb aliyun-push-caddy aliyun-push-hugoverse aliyun-push-all \
        publish-all release-all

# Version management
VERSION ?= latest
COUCHDB_VERSION ?= 3.3

# Platform configuration (for multi-architecture builds)
PLATFORM ?= linux/amd64
BUILD_PLATFORMS ?= linux/amd64,linux/arm64

# Docker Hub configuration
DOCKER_ORG = mdfriday
CADDY_IMAGE = $(DOCKER_ORG)/caddy
HUGOVERSE_IMAGE = $(DOCKER_ORG)/hugoverse
COUCHDB_IMAGE = couchdb

# Aliyun Container Registry configuration
ALIYUN_REGISTRY = registry.cn-hangzhou.aliyuncs.com
ALIYUN_REGISTRY_VPC = registry-vpc.cn-hangzhou.aliyuncs.com
ALIYUN_NAMESPACE = mdfriday
ALIYUN_CADDY_IMAGE = $(ALIYUN_REGISTRY)/$(ALIYUN_NAMESPACE)/caddy
ALIYUN_HUGOVERSE_IMAGE = $(ALIYUN_REGISTRY)/$(ALIYUN_NAMESPACE)/hugoverse
ALIYUN_COUCHDB_IMAGE = $(ALIYUN_REGISTRY)/$(ALIYUN_NAMESPACE)/couchdb

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
	@echo "📦 Docker Hub:"
	@echo "  make docker-login          - Login to Docker Hub"
	@echo "  make docker-build-caddy    - Build Caddy image"
	@echo "  make docker-build-hugoverse- Build Hugoverse image"
	@echo "  make docker-build-all      - Build all images"
	@echo "  make docker-push-caddy     - Push Caddy image to Docker Hub"
	@echo "  make docker-push-hugoverse - Push Hugoverse image to Docker Hub"
	@echo "  make docker-push-all       - Push all images to Docker Hub"
	@echo "  make release               - Build and push to Docker Hub"
	@echo ""
	@echo "☁️  Aliyun Container Registry:"
	@echo "  make aliyun-login              - Login to Aliyun Registry"
	@echo "  make aliyun-pull-couchdb       - Pull CouchDB from Docker Hub"
	@echo "  make aliyun-tag-couchdb        - Tag CouchDB for Aliyun"
	@echo "  make aliyun-tag-caddy          - Tag Caddy for Aliyun"
	@echo "  make aliyun-tag-hugoverse      - Tag Hugoverse for Aliyun"
	@echo "  make aliyun-tag-all            - Tag all images for Aliyun"
	@echo "  make aliyun-push-couchdb       - Push CouchDB to Aliyun"
	@echo "  make aliyun-push-caddy         - Push Caddy to Aliyun"
	@echo "  make aliyun-push-hugoverse     - Push Hugoverse to Aliyun"
	@echo "  make aliyun-push-all           - Push all images to Aliyun"
	@echo ""
	@echo "🚀 Publish to All Registries:"
	@echo "  make publish-all               - Push to Docker Hub + Aliyun"
	@echo "  make release-all               - Build and push to all registries"
	@echo ""
	@echo "💡 Examples:"
	@echo "  make release VERSION=2.0.0                    - Release to Docker Hub only"
	@echo "  make release-all VERSION=2.0.0                - Release to all registries"
	@echo "  make aliyun-push-all                          - Push existing images to Aliyun"
	@echo "  make aliyun-pull-couchdb                      - Pull CouchDB:3.3"
	@echo "  make aliyun-push-couchdb                      - Push CouchDB to Aliyun"
	@echo ""
	@echo "🏗️  Platform Options:"
	@echo "  make docker-build-all PLATFORM=linux/amd64    - Build for AMD64 (x86_64)"
	@echo "  make docker-build-all PLATFORM=linux/arm64    - Build for ARM64 (Apple Silicon)"
	@echo "  make aliyun-pull-couchdb PLATFORM=linux/amd64 - Pull AMD64 CouchDB"
	@echo ""
	@echo "  Default PLATFORM: linux/amd64 (for cloud servers)"
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

# ========== Docker Hub Commands ==========

# Login to Docker Hub
docker-login:
	@echo "🔐 Logging in to Docker Hub..."
	@docker login
	@echo "✅ Docker login successful"

# Build Caddy image
docker-build-caddy:
	@echo "🏗️  Building Caddy image..."
	@echo "   Version: $(VERSION)"
	@echo "   Platform: $(PLATFORM)"
	@if command -v docker-buildx >/dev/null 2>&1 || docker buildx version >/dev/null 2>&1; then \
		echo "   Using buildx for cross-platform build"; \
		if [ "$(VERSION)" = "latest" ]; then \
			docker buildx build --platform $(PLATFORM) \
				-t $(CADDY_IMAGE):latest \
				-f docker/caddy/Dockerfile \
				--load \
				docker/caddy/; \
		else \
			docker buildx build --platform $(PLATFORM) \
				-t $(CADDY_IMAGE):$(VERSION) \
				-t $(CADDY_IMAGE):latest \
				-f docker/caddy/Dockerfile \
				--load \
				docker/caddy/; \
		fi; \
	else \
		echo "   Using standard docker build (buildx not available)"; \
		echo "   ⚠️  Note: Building for current platform only"; \
		if [ "$(VERSION)" = "latest" ]; then \
			docker build \
				-t $(CADDY_IMAGE):latest \
				-f docker/caddy/Dockerfile \
				docker/caddy/; \
		else \
			docker build \
				-t $(CADDY_IMAGE):$(VERSION) \
				-t $(CADDY_IMAGE):latest \
				-f docker/caddy/Dockerfile \
				docker/caddy/; \
		fi; \
	fi
	@echo "✅ Caddy image built: $(CADDY_IMAGE):$(VERSION)"

# Build Hugoverse image
docker-build-hugoverse:
	@echo "🏗️  Building Hugoverse image..."
	@echo "   Version: $(VERSION)"
	@echo "   Platform: $(PLATFORM)"
	@if command -v docker-buildx >/dev/null 2>&1 || docker buildx version >/dev/null 2>&1; then \
		echo "   Using buildx for cross-platform build"; \
		if [ "$(VERSION)" = "latest" ]; then \
			docker buildx build --platform $(PLATFORM) \
				-t $(HUGOVERSE_IMAGE):latest \
				-f docker/hugoverse/Dockerfile \
				--load \
				.; \
		else \
			docker buildx build --platform $(PLATFORM) \
				-t $(HUGOVERSE_IMAGE):$(VERSION) \
				-t $(HUGOVERSE_IMAGE):latest \
				-f docker/hugoverse/Dockerfile \
				--load \
				.; \
		fi; \
	else \
		echo "   Using standard docker build (buildx not available)"; \
		echo "   ⚠️  Note: Building for current platform only"; \
		if [ "$(VERSION)" = "latest" ]; then \
			docker build \
				-t $(HUGOVERSE_IMAGE):latest \
				-f docker/hugoverse/Dockerfile \
				.; \
		else \
			docker build \
				-t $(HUGOVERSE_IMAGE):$(VERSION) \
				-t $(HUGOVERSE_IMAGE):latest \
				-f docker/hugoverse/Dockerfile \
				.; \
		fi; \
	fi
	@echo "✅ Hugoverse image built: $(HUGOVERSE_IMAGE):$(VERSION)"

# Build all images
docker-build-all: docker-build-caddy docker-build-hugoverse
	@echo ""
	@echo "✅ All images built successfully"
	@echo "   - $(CADDY_IMAGE):$(VERSION)"
	@echo "   - $(HUGOVERSE_IMAGE):$(VERSION)"

# Push Caddy image
docker-push-caddy:
	@echo "📤 Pushing Caddy image to Docker Hub..."
	@if [ "$(VERSION)" = "latest" ]; then \
		docker push $(CADDY_IMAGE):latest; \
	else \
		docker push $(CADDY_IMAGE):$(VERSION); \
		docker push $(CADDY_IMAGE):latest; \
	fi
	@echo "✅ Caddy image pushed: $(CADDY_IMAGE):$(VERSION)"

# Push Hugoverse image
docker-push-hugoverse:
	@echo "📤 Pushing Hugoverse image to Docker Hub..."
	@if [ "$(VERSION)" = "latest" ]; then \
		docker push $(HUGOVERSE_IMAGE):latest; \
	else \
		docker push $(HUGOVERSE_IMAGE):$(VERSION); \
		docker push $(HUGOVERSE_IMAGE):latest; \
	fi
	@echo "✅ Hugoverse image pushed: $(HUGOVERSE_IMAGE):$(VERSION)"

# Push all images
docker-push-all: docker-push-caddy docker-push-hugoverse
	@echo ""
	@echo "✅ All images pushed successfully"
	@echo "   - $(CADDY_IMAGE):$(VERSION)"
	@echo "   - $(HUGOVERSE_IMAGE):$(VERSION)"

# Release: build and push all images
release:
	@if [ "$(VERSION)" = "latest" ]; then \
		echo "⚠️  Warning: Releasing with 'latest' tag"; \
		echo "   Use 'make release VERSION=x.y.z' to specify a version"; \
		read -p "Continue? (y/N): " confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			echo "❌ Release cancelled"; \
			exit 1; \
		fi; \
	fi
	@echo ""
	@echo "📦 Releasing Hugoverse $(VERSION)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Step 1/3: Building images..."
	@$(MAKE) docker-build-all VERSION=$(VERSION)
	@echo ""
	@echo "Step 2/3: Verifying Docker login..."
	@docker info > /dev/null 2>&1 || (echo "❌ Docker not running" && exit 1)
	@if ! docker info 2>/dev/null | grep -q "Username:"; then \
		echo "⚠️  Not logged in to Docker Hub"; \
		$(MAKE) docker-login; \
	fi
	@echo ""
	@echo "Step 3/3: Pushing images to Docker Hub..."
	@$(MAKE) docker-push-all VERSION=$(VERSION)
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ Release $(VERSION) published successfully!"
	@echo ""
	@echo "📋 Published images:"
	@echo "   • $(CADDY_IMAGE):$(VERSION)"
	@echo "   • $(HUGOVERSE_IMAGE):$(VERSION)"
	@if [ "$(VERSION)" != "latest" ]; then \
		echo "   • $(CADDY_IMAGE):latest"; \
		echo "   • $(HUGOVERSE_IMAGE):latest"; \
	fi
	@echo ""
	@echo "🚀 Deploy on your server:"
	@echo "   1. Copy docker-compose.yml and .env.example to your server"
	@echo "   2. Create .env.local with your configuration"
	@echo "   3. Run: docker-compose --env-file .env.local pull"
	@echo "   4. Run: docker-compose --env-file .env.local up -d"
	@echo ""

# ========== Aliyun Container Registry Commands ==========

# Login to Aliyun Container Registry
aliyun-login:
	@echo "🔐 Logging in to Aliyun Container Registry..."
	@echo "   Registry: $(ALIYUN_REGISTRY)"
	@echo ""
	@echo "Please enter your Aliyun credentials:"
	@docker login $(ALIYUN_REGISTRY)
	@echo "✅ Aliyun login successful"

# Pull CouchDB from Docker Hub
aliyun-pull-couchdb:
	@echo "📥 Pulling CouchDB $(COUCHDB_VERSION) from Docker Hub..."
	@if docker pull --help 2>&1 | grep -q -- --platform; then \
		echo "   Platform: $(PLATFORM)"; \
		docker pull --platform $(PLATFORM) $(COUCHDB_IMAGE):$(COUCHDB_VERSION); \
	else \
		echo "   ⚠️  Note: --platform not supported, pulling default architecture"; \
		docker pull $(COUCHDB_IMAGE):$(COUCHDB_VERSION); \
	fi
	@echo "✅ CouchDB image pulled: $(COUCHDB_IMAGE):$(COUCHDB_VERSION)"

# Tag CouchDB image for Aliyun
aliyun-tag-couchdb:
	@echo "🏷️  Tagging CouchDB image for Aliyun..."
	@docker tag $(COUCHDB_IMAGE):$(COUCHDB_VERSION) $(ALIYUN_COUCHDB_IMAGE):$(COUCHDB_VERSION)
	@docker tag $(COUCHDB_IMAGE):$(COUCHDB_VERSION) $(ALIYUN_COUCHDB_IMAGE):latest
	@echo "✅ Tagged: $(ALIYUN_COUCHDB_IMAGE):$(COUCHDB_VERSION)"
	@echo "✅ Tagged: $(ALIYUN_COUCHDB_IMAGE):latest"

# Tag Caddy image for Aliyun
aliyun-tag-caddy:
	@echo "🏷️  Tagging Caddy image for Aliyun..."
	@if [ "$(VERSION)" = "latest" ]; then \
		docker tag $(CADDY_IMAGE):latest $(ALIYUN_CADDY_IMAGE):latest; \
		echo "✅ Tagged: $(ALIYUN_CADDY_IMAGE):latest"; \
	else \
		docker tag $(CADDY_IMAGE):$(VERSION) $(ALIYUN_CADDY_IMAGE):$(VERSION); \
		docker tag $(CADDY_IMAGE):latest $(ALIYUN_CADDY_IMAGE):latest; \
		echo "✅ Tagged: $(ALIYUN_CADDY_IMAGE):$(VERSION)"; \
		echo "✅ Tagged: $(ALIYUN_CADDY_IMAGE):latest"; \
	fi

# Tag Hugoverse image for Aliyun
aliyun-tag-hugoverse:
	@echo "🏷️  Tagging Hugoverse image for Aliyun..."
	@if [ "$(VERSION)" = "latest" ]; then \
		docker tag $(HUGOVERSE_IMAGE):latest $(ALIYUN_HUGOVERSE_IMAGE):latest; \
		echo "✅ Tagged: $(ALIYUN_HUGOVERSE_IMAGE):latest"; \
	else \
		docker tag $(HUGOVERSE_IMAGE):$(VERSION) $(ALIYUN_HUGOVERSE_IMAGE):$(VERSION); \
		docker tag $(HUGOVERSE_IMAGE):latest $(ALIYUN_HUGOVERSE_IMAGE):latest; \
		echo "✅ Tagged: $(ALIYUN_HUGOVERSE_IMAGE):$(VERSION)"; \
		echo "✅ Tagged: $(ALIYUN_HUGOVERSE_IMAGE):latest"; \
	fi

# Tag all images for Aliyun
aliyun-tag-all: aliyun-tag-couchdb aliyun-tag-caddy aliyun-tag-hugoverse
	@echo ""
	@echo "✅ All images tagged for Aliyun successfully"
	@echo "   - $(ALIYUN_COUCHDB_IMAGE):$(COUCHDB_VERSION)"
	@echo "   - $(ALIYUN_CADDY_IMAGE):$(VERSION)"
	@echo "   - $(ALIYUN_HUGOVERSE_IMAGE):$(VERSION)"

# Push CouchDB image to Aliyun
aliyun-push-couchdb: aliyun-tag-couchdb
	@echo "📤 Pushing CouchDB image to Aliyun Registry..."
	@docker push $(ALIYUN_COUCHDB_IMAGE):$(COUCHDB_VERSION)
	@docker push $(ALIYUN_COUCHDB_IMAGE):latest
	@echo "✅ CouchDB image pushed: $(ALIYUN_COUCHDB_IMAGE):$(COUCHDB_VERSION)"

# Push Caddy image to Aliyun
aliyun-push-caddy: aliyun-tag-caddy
	@echo "📤 Pushing Caddy image to Aliyun Registry..."
	@if [ "$(VERSION)" = "latest" ]; then \
		docker push $(ALIYUN_CADDY_IMAGE):latest; \
	else \
		docker push $(ALIYUN_CADDY_IMAGE):$(VERSION); \
		docker push $(ALIYUN_CADDY_IMAGE):latest; \
	fi
	@echo "✅ Caddy image pushed: $(ALIYUN_CADDY_IMAGE):$(VERSION)"

# Push Hugoverse image to Aliyun
aliyun-push-hugoverse: aliyun-tag-hugoverse
	@echo "📤 Pushing Hugoverse image to Aliyun Registry..."
	@if [ "$(VERSION)" = "latest" ]; then \
		docker push $(ALIYUN_HUGOVERSE_IMAGE):latest; \
	else \
		docker push $(ALIYUN_HUGOVERSE_IMAGE):$(VERSION); \
		docker push $(ALIYUN_HUGOVERSE_IMAGE):latest; \
	fi
	@echo "✅ Hugoverse image pushed: $(ALIYUN_HUGOVERSE_IMAGE):$(VERSION)"

# Push all images to Aliyun
aliyun-push-all: aliyun-tag-all aliyun-push-couchdb aliyun-push-caddy aliyun-push-hugoverse
	@echo ""
	@echo "✅ All images pushed to Aliyun successfully"
	@echo "   - $(ALIYUN_COUCHDB_IMAGE):$(COUCHDB_VERSION)"
	@echo "   - $(ALIYUN_CADDY_IMAGE):$(VERSION)"
	@echo "   - $(ALIYUN_HUGOVERSE_IMAGE):$(VERSION)"

# ========== Publish to All Registries ==========

# Push to Docker Hub + Aliyun
publish-all:
	@echo ""
	@echo "🚀 Publishing to All Registries"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Step 1/2: Pushing to Docker Hub..."
	@$(MAKE) docker-push-all VERSION=$(VERSION)
	@echo ""
	@echo "Step 2/2: Pushing to Aliyun Registry..."
	@$(MAKE) aliyun-push-all VERSION=$(VERSION)
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ Published to all registries successfully!"
	@echo ""
	@echo "📋 Docker Hub:"
	@echo "   • $(CADDY_IMAGE):$(VERSION)"
	@echo "   • $(HUGOVERSE_IMAGE):$(VERSION)"
	@echo ""
	@echo "📋 Aliyun Registry:"
	@echo "   • $(ALIYUN_COUCHDB_IMAGE):$(COUCHDB_VERSION)"
	@echo "   • $(ALIYUN_CADDY_IMAGE):$(VERSION)"
	@echo "   • $(ALIYUN_HUGOVERSE_IMAGE):$(VERSION)"
	@echo ""

# Build and push to all registries
release-all:
	@if [ "$(VERSION)" = "latest" ]; then \
		echo "⚠️  Warning: Releasing with 'latest' tag"; \
		echo "   Use 'make release-all VERSION=x.y.z' to specify a version"; \
		read -p "Continue? (y/N): " confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			echo "❌ Release cancelled"; \
			exit 1; \
		fi; \
	fi
	@echo ""
	@echo "📦 Releasing Hugoverse $(VERSION) to All Registries"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Step 1/4: Building images..."
	@$(MAKE) docker-build-all VERSION=$(VERSION)
	@echo ""
	@echo "Step 2/4: Verifying Docker Hub login..."
	@docker info > /dev/null 2>&1 || (echo "❌ Docker not running" && exit 1)
	@if ! docker info 2>/dev/null | grep -q "Username:"; then \
		echo "⚠️  Not logged in to Docker Hub"; \
		$(MAKE) docker-login; \
	fi
	@echo ""
	@echo "Step 3/4: Pushing to Docker Hub..."
	@$(MAKE) docker-push-all VERSION=$(VERSION)
	@echo ""
	@echo "Step 4/4: Pushing to Aliyun Registry..."
	@$(MAKE) aliyun-push-all VERSION=$(VERSION)
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ Release $(VERSION) published to all registries!"
	@echo ""
	@echo "📋 Docker Hub:"
	@echo "   • $(CADDY_IMAGE):$(VERSION)"
	@echo "   • $(HUGOVERSE_IMAGE):$(VERSION)"
	@if [ "$(VERSION)" != "latest" ]; then \
		echo "   • $(CADDY_IMAGE):latest"; \
		echo "   • $(HUGOVERSE_IMAGE):latest"; \
	fi
	@echo ""
	@echo "📋 Aliyun Registry:"
	@echo "   • $(ALIYUN_COUCHDB_IMAGE):$(COUCHDB_VERSION)"
	@echo "   • $(ALIYUN_COUCHDB_IMAGE):latest"
	@echo "   • $(ALIYUN_CADDY_IMAGE):$(VERSION)"
	@echo "   • $(ALIYUN_HUGOVERSE_IMAGE):$(VERSION)"
	@if [ "$(VERSION)" != "latest" ]; then \
		echo "   • $(ALIYUN_CADDY_IMAGE):latest"; \
		echo "   • $(ALIYUN_HUGOVERSE_IMAGE):latest"; \
	fi
	@echo ""
	@echo "🚀 Deploy on your server:"
	@echo ""
	@echo "From Docker Hub:"
	@echo "   docker-compose --env-file .env.local pull"
	@echo ""
	@echo "From Aliyun Registry (faster in China):"
	@echo "   docker-compose -f docker-compose.yml -f docker-compose.aliyun.yml --env-file .env.local pull"
	@echo ""

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
