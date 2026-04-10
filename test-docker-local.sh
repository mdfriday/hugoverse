#!/bin/bash
set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 Hugoverse Docker 本地测试（命令行版本）"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 1. 检查 Docker
echo -e "${BLUE}📦 Step 1: 检查 Docker 命令行...${NC}"

# 检查 Docker 命令是否存在
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker 命令未找到${NC}"
    echo ""
    echo "推荐安装方式（纯命令行，无需 Docker Desktop）："
    echo ""
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "🍺 macOS - 使用 Colima + Docker CLI："
        echo "  # 安装 Colima 和 Docker CLI"
        echo "  brew install colima docker docker-compose"
        echo ""
        echo "  # 启动 Colima"
        echo "  colima start --cpu 2 --memory 4"
        echo ""
        echo "  # 验证"
        echo "  docker --version"
    else
        echo "🐧 Linux - 安装 Docker Engine："
        echo "  curl -fsSL https://get.docker.com | sh"
        echo "  sudo usermod -aG docker \$USER"
        echo "  newgrp docker"
    fi
    echo ""
    exit 1
fi

# 检查 Docker daemon 是否运行
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker daemon 未运行${NC}"
    echo ""
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "如果使用 Colima，请启动："
        echo "  colima start --cpu 2 --memory 4"
        echo ""
        echo "查看状态："
        echo "  colima status"
    else
        echo "启动 Docker 服务："
        echo "  sudo systemctl start docker"
    fi
    echo ""
    exit 1
fi

# 显示 Docker 信息
DOCKER_VERSION=$(docker --version)
echo -e "${GREEN}✅ Docker CLI: $DOCKER_VERSION${NC}"

# 检查是否是 Colima
if docker context ls 2>/dev/null | grep -q colima; then
    echo -e "${GREEN}✅ 运行环境: Colima${NC}"
elif docker info 2>/dev/null | grep -q "Docker Desktop"; then
    echo -e "${YELLOW}⚠️  检测到 Docker Desktop (建议使用 Colima)${NC}"
else
    echo -e "${GREEN}✅ 运行环境: Docker Engine${NC}"
fi

# 检查 docker compose（支持 V1 和 V2）
COMPOSE_CMD=""
if docker compose version > /dev/null 2>&1; then
    # Docker Compose V2 (作为 docker 子命令)
    COMPOSE_CMD="docker compose"
    COMPOSE_VERSION=$(docker compose version --short 2>&1 | head -1)
    echo -e "${GREEN}✅ Docker Compose V2: $COMPOSE_VERSION${NC}"
elif command -v docker-compose > /dev/null 2>&1; then
    # Docker Compose V1 (独立命令)
    COMPOSE_CMD="docker-compose"
    COMPOSE_VERSION=$(docker-compose version --short 2>&1)
    echo -e "${GREEN}✅ Docker Compose V1: $COMPOSE_VERSION${NC}"
else
    echo -e "${RED}❌ Docker Compose 未安装${NC}"
    echo "安装 docker-compose:"
    echo "  brew install docker-compose"
    exit 1
fi

echo ""

# 2. 检查 Go 编译
echo -e "${BLUE}📦 Step 2: 测试 Go 编译...${NC}"
if go build -o /tmp/hugoverse-test ./cmd/hugoverse 2>&1; then
    echo -e "${GREEN}✅ Go 编译成功${NC}"
    rm -f /tmp/hugoverse-test
else
    echo -e "${RED}❌ Go 编译失败${NC}"
    exit 1
fi
echo ""

# 3. 准备本地配置
echo -e "${BLUE}📝 Step 3: 准备本地测试配置...${NC}"
if [ ! -f .env.local ]; then
    cat > .env.local << 'ENVEOF'
# Hugoverse 本地测试配置
AUTO_INIT=true
DOMAIN=localhost
SERVER_IP=127.0.0.1
SITE_NAME=Hugoverse-Local

# Admin
ADMIN_EMAIL=admin@localhost
ADMIN_PASSWORD=test123456

# CouchDB
COUCHDB_USER=admin
COUCHDB_PASSWORD=test123456
COUCHDB_DB_PREFIX=userdb-

# DNSPod（本地不需要）
DNSPOD_ENABLED=false
DNSPOD_ID=
DNSPOD_SECRET=

# Master License（测试免费版）
MASTER_LICENSE=

# Enterprise
AUTO_GENERATE_ENTERPRISE_LICENSE=true
ENTERPRISE_LICENSE_PLAN=enterprise
ENTERPRISE_LICENSE_COUNT=1
AUTO_CONFIGURE_ENTERPRISE_SITE=true
ENTERPRISE_SITE_DOMAIN=localhost

# Backup
BACKUP_ENABLED=true
BACKUP_RETENTION_DAYS=7

# Ports（避免与系统服务冲突）
HTTP_PORT=8080
HTTPS_PORT=8443
VERSION=local-dev
LOG_LEVEL=debug
ENVEOF
    echo -e "${GREEN}✅ 创建 .env.local${NC}"
else
    echo -e "${YELLOW}⚠️  .env.local 已存在，跳过${NC}"
fi
echo ""

# 4. 验证 docker-compose.yml
echo -e "${BLUE}🔍 Step 4: 验证 docker-compose.yml...${NC}"
if $COMPOSE_CMD --env-file .env.local config > /dev/null 2>&1; then
    echo -e "${GREEN}✅ docker-compose.yml 语法正确${NC}"
else
    echo -e "${RED}❌ docker-compose.yml 有错误${NC}"
    $COMPOSE_CMD --env-file .env.local config
    exit 1
fi
echo ""

# 5. 清理旧容器（如果存在）
echo -e "${BLUE}🧹 Step 5: 清理旧容器...${NC}"
if $COMPOSE_CMD --env-file .env.local ps -q > /dev/null 2>&1; then
    $COMPOSE_CMD --env-file .env.local down -v > /dev/null 2>&1 || true
    echo -e "${GREEN}✅ 清理完成${NC}"
else
    echo -e "${GREEN}✅ 无需清理${NC}"
fi
echo ""

# 6. 构建镜像
echo -e "${BLUE}🔨 Step 6: 构建 Docker 镜像（需要 2-5 分钟）...${NC}"
echo -e "${YELLOW}提示: Colima 首次构建可能较慢，请耐心等待${NC}"
$COMPOSE_CMD --env-file .env.local build
echo -e "${GREEN}✅ 镜像构建完成${NC}"
echo ""

# 7. 启动服务
echo -e "${BLUE}🚀 Step 7: 启动所有服务...${NC}"
$COMPOSE_CMD --env-file .env.local up -d
echo -e "${GREEN}✅ 服务已启动${NC}"
echo ""

# 8. 等待服务就绪
echo -e "${BLUE}⏳ Step 8: 等待服务启动（60秒）...${NC}"
for i in {1..60}; do
    printf "."
    sleep 1
    
    # 每 10 秒检查一次健康状态
    if [ $((i % 10)) -eq 0 ]; then
        if curl -sf http://localhost:8080/api/health > /dev/null 2>&1; then
            echo ""
            echo -e "${GREEN}✅ 服务已就绪（${i}秒）${NC}"
            break
        fi
    fi
done
echo ""
echo ""

# 9. 检查服务状态
echo -e "${BLUE}📊 Step 9: 检查服务状态...${NC}"
$COMPOSE_CMD --env-file .env.local ps
echo ""

# 10. 健康检查
echo -e "${BLUE}🏥 Step 10: 健康检查...${NC}"
HEALTH=$(curl -s http://localhost:8080/api/health 2>&1)
if echo "$HEALTH" | grep -q "healthy"; then
    echo -e "${GREEN}✅ Hugoverse 健康检查通过${NC}"
    if command -v jq &> /dev/null; then
        echo "$HEALTH" | jq .
    else
        echo "$HEALTH"
    fi
else
    echo -e "${RED}❌ Hugoverse 健康检查失败${NC}"
    echo ""
    echo "查看最近日志："
    $COMPOSE_CMD --env-file .env.local logs --tail=50 hugoverse
    echo ""
    echo -e "${YELLOW}提示：服务可能需要更多时间启动，请稍后再试${NC}"
    echo "手动检查："
    echo "  $COMPOSE_CMD --env-file .env.local logs -f hugoverse"
fi
echo ""

# 11. 查看生成的 License
echo -e "${BLUE}🔑 Step 11: 查看生成的 License...${NC}"
LICENSES=$($COMPOSE_CMD --env-file .env.local logs hugoverse 2>&1 | grep -A 2 "License Key:" | head -15)
if [ -n "$LICENSES" ]; then
    echo -e "${GREEN}$LICENSES${NC}"
else
    echo -e "${YELLOW}⚠️  未找到 License 信息，请检查日志：${NC}"
    echo "  $COMPOSE_CMD --env-file .env.local logs hugoverse | grep -A 5 'License'"
fi
echo ""

# 12. 测试 API 端点
echo -e "${BLUE}🧪 Step 12: 测试 API 端点...${NC}"

# 测试健康检查
if curl -sf http://localhost:8080/api/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ /api/health${NC}"
else
    echo -e "${RED}❌ /api/health${NC}"
fi

# 测试 Admin 面板
if curl -sf http://localhost:8080/admin > /dev/null 2>&1; then
    echo -e "${GREEN}✅ /admin${NC}"
else
    echo -e "${YELLOW}⚠️  /admin (可能需要更多时间初始化)${NC}"
fi

# 测试 CouchDB 代理
if curl -sf http://localhost:8080/_up > /dev/null 2>&1; then
    echo -e "${GREEN}✅ CouchDB 代理${NC}"
else
    echo -e "${YELLOW}⚠️  CouchDB 代理 (可能需要配置)${NC}"
fi
echo ""

# 显示 Colima 信息（如果使用）
if docker context ls 2>/dev/null | grep -q colima; then
    echo -e "${BLUE}🐳 Colima 信息：${NC}"
    echo "  状态: $(colima status 2>&1 | head -1)"
    echo "  查看详情: colima status"
    echo "  查看日志: colima logs"
    echo ""
fi

# 13. 总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ 本地测试完成！${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${BLUE}📍 访问地址：${NC}"
echo "   Admin 面板: http://localhost:8080/admin"
echo "   API 健康:   http://localhost:8080/api/health"
echo "   CouchDB:    http://localhost:8080/_utils"
echo ""
echo -e "${BLUE}🔐 登录信息：${NC}"
echo "   Email:    admin@localhost"
echo "   Password: test123456"
echo ""
echo -e "${BLUE}📋 常用命令：${NC}"
echo "   查看所有日志:"
echo "     $COMPOSE_CMD --env-file .env.local logs -f"
echo ""
echo "   查看特定服务:"
echo "     $COMPOSE_CMD --env-file .env.local logs -f hugoverse"
echo "     $COMPOSE_CMD --env-file .env.local logs -f caddy"
echo "     $COMPOSE_CMD --env-file .env.local logs -f couchdb"
echo ""
echo "   重启服务:"
echo "     $COMPOSE_CMD --env-file .env.local restart"
echo ""
echo "   停止服务:"
echo "     $COMPOSE_CMD --env-file .env.local down"
echo ""
echo "   完全清理（包括数据）:"
echo "     $COMPOSE_CMD --env-file .env.local down -v"
echo ""

if docker context ls 2>/dev/null | grep -q colima; then
    echo -e "${BLUE}🔧 Colima 管理：${NC}"
    echo "   查看状态:"
    echo "     colima status"
    echo ""
    echo "   停止 Colima:"
    echo "     colima stop"
    echo ""
    echo "   重启 Colima:"
    echo "     colima restart"
    echo ""
    echo "   删除 Colima (清理所有):"
    echo "     colima delete"
    echo ""
fi

echo -e "${BLUE}🔧 调试技巧：${NC}"
echo "   进入容器:"
echo "     $COMPOSE_CMD --env-file .env.local exec hugoverse sh"
echo ""
echo "   查看容器状态:"
echo "     $COMPOSE_CMD --env-file .env.local ps"
echo ""
echo "   重新构建镜像:"
echo "     $COMPOSE_CMD --env-file .env.local build --no-cache"
echo ""
echo -e "${YELLOW}💡 提示：${NC}"
echo "   - 本地测试使用端口 8080 (HTTP) 和 8443 (HTTPS)"
echo "   - Master License 验证会失败（网络原因），这是正常的"
echo "   - 系统会自动降级到免费版（1 个 License）"
echo "   - 如需修改配置，编辑 .env.local 后重启服务"
if docker context ls 2>/dev/null | grep -q colima; then
    echo "   - Colima 使用更少资源，推荐用于开发环境"
fi
echo ""
