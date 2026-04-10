#!/bin/bash
# 快速验证脚本 - 只检查不启动服务（支持 Colima）

set -e

echo "🔍 Hugoverse Docker 快速验证（命令行版本）"
echo ""

# 检查 Docker
echo "1️⃣  检查 Docker CLI..."
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 命令未安装"
    echo ""
    echo "推荐安装方式（纯命令行，无需 Docker Desktop）："
    echo ""
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "🍺 macOS - 使用 Colima（推荐）："
        echo "  # 安装"
        echo "  brew install colima docker docker-compose"
        echo ""
        echo "  # 启动 Colima (2 CPU, 4GB 内存)"
        echo "  colima start --cpu 2 --memory 4"
        echo ""
        echo "  # 查看状态"
        echo "  colima status"
        echo ""
        echo "🎯 为什么选择 Colima？"
        echo "  ✅ 轻量级，占用资源少"
        echo "  ✅ 纯命令行，无 GUI"
        echo "  ✅ 完全兼容 Docker CLI"
        echo "  ✅ 启动快，性能好"
    else
        echo "🐧 Linux - 安装 Docker Engine："
        echo "  curl -fsSL https://get.docker.com | sh"
        echo "  sudo usermod -aG docker \$USER"
        echo "  newgrp docker"
    fi
    echo ""
    exit 1
fi

if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker daemon 未运行"
    echo ""
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "如果使用 Colima："
        echo "  colima start --cpu 2 --memory 4"
        echo ""
        echo "如果使用 Docker Desktop："
        echo "  # Docker Desktop 不推荐，建议改用 Colima"
    else
        echo "启动 Docker 服务："
        echo "  sudo systemctl start docker"
    fi
    echo ""
    exit 1
fi

DOCKER_VERSION=$(docker --version | awk '{print $3}')
echo "✅ Docker CLI: $DOCKER_VERSION"

# 检查运行环境
if docker context ls 2>/dev/null | grep -q colima; then
    echo "✅ 运行环境: Colima (轻量级容器运行时)"
    COLIMA_STATUS=$(colima status 2>&1 | head -1)
    echo "   状态: $COLIMA_STATUS"
elif docker info 2>/dev/null | grep -q "Docker Desktop"; then
    echo "⚠️  运行环境: Docker Desktop"
    echo "   建议: 改用 Colima 以获得更好的性能"
    echo "   安装: brew install colima && colima start"
else
    echo "✅ 运行环境: Docker Engine"
fi

# 检查 docker compose（支持 V1 和 V2）
if docker compose version > /dev/null 2>&1; then
    # Docker Compose V2
    COMPOSE_VERSION=$(docker compose version --short 2>&1 | head -1)
    echo "✅ Docker Compose V2: $COMPOSE_VERSION"
elif command -v docker-compose > /dev/null 2>&1; then
    # Docker Compose V1
    COMPOSE_VERSION=$(docker-compose version --short 2>&1)
    echo "✅ Docker Compose V1: $COMPOSE_VERSION"
else
    echo "❌ Docker Compose 未安装"
    echo "   安装: brew install docker-compose"
    exit 1
fi
echo ""

# 检查 Go
echo "2️⃣  检查 Go 编译..."
if go build -o /tmp/hugoverse-test ./cmd/hugoverse 2>&1; then
    echo "✅ Go 编译成功"
    rm -f /tmp/hugoverse-test
else
    echo "❌ Go 编译失败"
    exit 1
fi
echo ""

# 检查 Docker 配置
echo "3️⃣  检查 Docker 配置..."
if docker compose config > /dev/null 2>&1; then
    echo "✅ docker-compose.yml 语法正确"
else
    echo "❌ docker-compose.yml 有错误"
    exit 1
fi
echo ""

# 检查必要文件
echo "4️⃣  检查必要文件..."
files=(
    "docker-compose.yml"
    ".env.example"
    "docker/caddy/Dockerfile"
    "docker/caddy/Caddyfile.initial"
    "docker/hugoverse/Dockerfile"
    "docker/hugoverse/docker-entrypoint.sh"
)

all_ok=true
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "❌ $file 缺失"
        all_ok=false
    fi
done
echo ""

if [ "$all_ok" = false ]; then
    echo "❌ 缺少必要文件"
    exit 1
fi

# 总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 所有检查通过！"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📋 下一步："
echo ""
echo "1️⃣  运行完整测试（包含启动服务）："
echo "   ./test-docker-local.sh"
echo ""
echo "2️⃣  或手动操作："
echo "   # 准备配置"
echo "   cp .env.example .env.local"
echo "   # 编辑配置（可选）"
echo "   vim .env.local"
echo "   # 启动服务"
echo "   docker compose --env-file .env.local up -d"
echo "   # 查看日志"
echo "   docker compose --env-file .env.local logs -f"
echo ""
echo "3️⃣  构建生产镜像："
echo "   # 设置版本"
echo "   VERSION=2.0.0"
echo "   # 构建并打标签"
echo "   docker build -t mdfriday/hugoverse:\$VERSION -f docker/hugoverse/Dockerfile ."
echo "   docker build -t mdfriday/hugoverse-caddy:\$VERSION -f docker/caddy/Dockerfile docker/caddy/"
echo "   # 推送到 Docker Hub"
echo "   docker push mdfriday/hugoverse:\$VERSION"
echo "   docker push mdfriday/hugoverse-caddy:\$VERSION"
echo ""
echo "📚 更多信息："
echo "   - 本地开发指南: cat LOCAL-DEVELOPMENT.md"
echo "   - Docker 文档:   cat DOCKER.md"
echo "   - 快速开始:      cat QUICKSTART.md"
echo ""
