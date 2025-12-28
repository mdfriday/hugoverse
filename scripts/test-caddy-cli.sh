#!/bin/bash
#
# Caddy CLI 快速测试脚本
# 测试所有 caddy 命令的基本功能
#

set -e

echo "🧪 Caddy CLI 测试脚本"
echo "===================="
echo ""

# 检查 hugov 是否存在
if [ ! -f "/tmp/hugov" ]; then
    echo "❌ /tmp/hugov 不存在，请先编译：go build -o /tmp/hugov main.go"
    exit 1
fi

HUGOV="/tmp/hugov"

echo "✅ 找到 hugov: $HUGOV"
echo ""

# 测试 1: 查看帮助
echo "📋 测试 1: 查看 caddy 帮助"
$HUGOV caddy 2>&1 | head -10
echo ""

# 测试 2: 查看主命令帮助（确认 caddy 命令已添加）
echo "📋 测试 2: 查看主命令帮助"
$HUGOV 2>&1 | grep -A 2 "caddy"
echo ""

# 测试 3: 测试 status 命令（Caddy 未运行时）
echo "📋 测试 3: 检查 status（Caddy 未运行）"
$HUGOV caddy status 2>&1 | head -5 || true
echo ""

# 测试 4: 准备测试站点
echo "📋 测试 4: 准备测试站点"
TEST_SITE_DIR="/tmp/caddy-test-site"
mkdir -p "$TEST_SITE_DIR"
cat > "$TEST_SITE_DIR/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Caddy Test Site</title>
</head>
<body>
    <h1>Hello from Caddy Test Site!</h1>
    <p>This is a test page.</p>
</body>
</html>
EOF
echo "✅ 测试站点已创建: $TEST_SITE_DIR"
ls -lh "$TEST_SITE_DIR/index.html"
echo ""

# 说明
echo "🎯 后续手动测试步骤："
echo ""
echo "1️⃣  启动 Caddy（在新终端）："
echo "   cd /Users/sunwei/github/mdfriday/hugoverse"
echo "   go run main.go caddy start -domain localhost -backend 127.0.0.1:1314"
echo ""
echo "2️⃣  查看状态（在另一个终端）："
echo "   go run main.go caddy status"
echo ""
echo "3️⃣  添加测试站点："
echo "   go run main.go caddy add -domain testsite.local -path $TEST_SITE_DIR"
echo ""
echo "4️⃣  配置 hosts："
echo "   echo '127.0.0.1 testsite.local' | sudo tee -a /etc/hosts"
echo ""
echo "5️⃣  测试访问："
echo "   curl http://testsite.local"
echo "   # 或在浏览器打开: http://testsite.local"
echo ""
echo "6️⃣  查看证书状态（可选）："
echo "   go run main.go caddy cert -domain testsite.local"
echo ""
echo "7️⃣  移除站点："
echo "   go run main.go caddy remove -domain testsite.local"
echo ""
echo "8️⃣  停止 Caddy："
echo "   在启动 Caddy 的终端按 Ctrl+C"
echo ""

# 检查 Caddy 是否已安装
echo "🔍 检查 Caddy 安装..."
if command -v caddy &> /dev/null; then
    CADDY_VERSION=$(caddy version 2>&1 | head -1)
    echo "✅ Caddy 已安装: $CADDY_VERSION"
else
    echo "⚠️  Caddy 未安装"
    echo "   安装方法（macOS）: brew install caddy"
    echo "   安装方法（Linux）: 参考 https://caddyserver.com/docs/install"
fi
echo ""

echo "✅ 基础测试完成！"
echo ""
echo "💡 提示："
echo "   - 测试站点目录: $TEST_SITE_DIR"
echo "   - 开发环境使用 localhost 域名"
echo "   - 绑定 80/443 端口需要 sudo 权限"
echo ""

