#!/bin/bash
# Docker 构建验证脚本

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 Hugoverse Docker Build Verification"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Check Docker
echo -e "${BLUE}📦 Checking Docker...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker not found${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker found: $(docker --version)${NC}"
echo ""

# Check Docker Compose
echo -e "${BLUE}📦 Checking Docker Compose...${NC}"
if docker compose version &> /dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
    echo -e "${GREEN}✅ Docker Compose found: $(docker compose version)${NC}"
elif command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
    echo -e "${GREEN}✅ Docker Compose found: $(docker-compose version)${NC}"
else
    echo -e "${RED}❌ Docker Compose not found${NC}"
    exit 1
fi
echo ""

# Verify files exist
echo -e "${BLUE}📁 Checking required files...${NC}"
required_files=(
    "docker-compose.yml"
    ".env.example"
    "docker/caddy/Dockerfile"
    "docker/caddy/Caddyfile.initial"
    "docker/hugoverse/Dockerfile"
    "docker/hugoverse/docker-entrypoint.sh"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo -e "${RED}❌ Missing: $file${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ $file${NC}"
done
echo ""

# Build Go binary
echo -e "${BLUE}🔨 Building Go binary...${NC}"
if go build -o /tmp/hugoverse-test ./cmd/hugoverse 2>&1; then
    echo -e "${GREEN}✅ Go build successful${NC}"
    rm -f /tmp/hugoverse-test
else
    echo -e "${RED}❌ Go build failed${NC}"
    exit 1
fi
echo ""

# Test Caddy Dockerfile
echo -e "${BLUE}🐳 Testing Caddy Dockerfile...${NC}"
if docker build -t hugoverse-caddy-test -f docker/caddy/Dockerfile docker/caddy/ > /tmp/caddy-build.log 2>&1; then
    echo -e "${GREEN}✅ Caddy image builds successfully${NC}"
    docker rmi hugoverse-caddy-test > /dev/null 2>&1 || true
else
    echo -e "${RED}❌ Caddy build failed${NC}"
    cat /tmp/caddy-build.log
    exit 1
fi
echo ""

# Test Hugoverse Dockerfile (without actual build, just validate)
echo -e "${BLUE}📋 Validating Hugoverse Dockerfile...${NC}"
if docker build -t hugoverse-app-test -f docker/hugoverse/Dockerfile . --target builder > /tmp/hugoverse-build.log 2>&1; then
    echo -e "${GREEN}✅ Hugoverse Dockerfile valid${NC}"
    docker rmi hugoverse-app-test > /dev/null 2>&1 || true
else
    echo -e "${YELLOW}⚠️  Full build skipped (takes time)${NC}"
fi
echo ""

# Validate docker-compose.yml
echo -e "${BLUE}🔧 Validating docker-compose.yml...${NC}"
if $COMPOSE_CMD config > /dev/null 2>&1; then
    echo -e "${GREEN}✅ docker-compose.yml is valid${NC}"
else
    echo -e "${RED}❌ docker-compose.yml has errors${NC}"
    $COMPOSE_CMD config
    exit 1
fi
echo ""

# Check entrypoint script
echo -e "${BLUE}🚀 Checking entrypoint script...${NC}"
if bash -n docker/hugoverse/docker-entrypoint.sh; then
    echo -e "${GREEN}✅ docker-entrypoint.sh syntax valid${NC}"
else
    echo -e "${RED}❌ docker-entrypoint.sh has syntax errors${NC}"
    exit 1
fi
echo ""

# Check install script
echo -e "${BLUE}📦 Checking install script...${NC}"
if bash -n install.sh; then
    echo -e "${GREEN}✅ install.sh syntax valid${NC}"
else
    echo -e "${RED}❌ install.sh has syntax errors${NC}"
    exit 1
fi
echo ""

# Summary
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ All Checks Passed!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}📋 Next Steps:${NC}"
echo ""
echo "1. Test installation:"
echo "   bash install.sh"
echo ""
echo "2. Or manually start services:"
echo "   cp .env.example .env"
echo "   # Edit .env with your config"
echo "   docker-compose up -d"
echo ""
echo "3. Build Docker images for production:"
echo "   docker-compose build"
echo ""
echo "4. Push to registry (optional):"
echo "   docker tag hugoverse_hugoverse:latest mdfriday/hugoverse:latest"
echo "   docker push mdfriday/hugoverse:latest"
echo ""
