#!/bin/bash
#
# 验证 Caddy 开发环境修复
# 测试 localhost 是否能正常启动（不需要 sudo，不申请证书）
#

echo "🧪 验证 Caddy 开发环境修复"
echo "=========================="
echo ""

# 检查编译
if [ ! -f "/tmp/hugov" ]; then
    echo "📦 编译 hugov..."
    cd /Users/sunwei/github/mdfriday/hugoverse
    go build -o /tmp/hugov main.go
    echo "✅ 编译完成"
else
    echo "✅ 找到编译好的 hugov"
fi
echo ""

# 显示版本信息
echo "📋 Caddy 版本:"
caddy version 2>&1 | head -1
echo ""

# 说明预期行为
echo "🎯 预期行为（开发模式）:"
echo "   ✅ 使用 HTTP only（无 HTTPS）"
echo "   ✅ 监听 8080 端口（无需 sudo）"
echo "   ✅ 不尝试申请 SSL 证书"
echo "   ✅ 不安装根证书"
echo "   ✅ 快速启动（5-10秒）"
echo ""

# 显示测试步骤
echo "📝 手动测试步骤:"
echo ""
echo "1️⃣  在一个终端启动 Caddy:"
echo "   cd /Users/sunwei/github/mdfriday/hugoverse"
echo "   /tmp/hugov caddy start"
echo ""
echo "   预期输出:"
echo "   ✅ Mode: Development (HTTP only, no sudo required)"
echo "   ✅ Access via: http://localhost:8080"
echo "   ✅ 不会提示输入密码"
echo "   ✅ 不会有 OCSP stapling 警告"
echo ""
echo "2️⃣  在另一个终端测试访问:"
echo "   curl http://localhost:8080"
echo "   # 或浏览器打开: http://localhost:8080"
echo ""
echo "3️⃣  测试 Admin API:"
echo "   /tmp/hugov caddy status"
echo ""
echo "4️⃣  停止服务:"
echo "   在启动的终端按 Ctrl+C"
echo ""

# 检查端口占用
echo "🔍 检查端口占用情况:"
if lsof -i :8080 > /dev/null 2>&1; then
    echo "⚠️  端口 8080 已被占用:"
    lsof -i :8080 | grep LISTEN
    echo "   请先停止占用该端口的进程"
else
    echo "✅ 端口 8080 可用"
fi
echo ""

if lsof -i :2019 > /dev/null 2>&1; then
    echo "⚠️  端口 2019 (Admin API) 已被占用:"
    lsof -i :2019 | grep LISTEN
    echo "   请先停止 Caddy"
else
    echo "✅ 端口 2019 (Admin API) 可用"
fi
echo ""

# 创建测试站点
TEST_DIR="/tmp/caddy-dev-test"
mkdir -p "$TEST_DIR"
cat > "$TEST_DIR/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Caddy Dev Test</title>
    <style>
        body { font-family: Arial; max-width: 600px; margin: 50px auto; padding: 20px; }
        h1 { color: #2196F3; }
        .success { color: #4CAF50; }
    </style>
</head>
<body>
    <h1>✅ Caddy Development Mode Works!</h1>
    <p class="success">
        You are accessing this page through Caddy running on port 8080.
    </p>
    <ul>
        <li>HTTP Only (no HTTPS)</li>
        <li>No sudo required</li>
        <li>No certificate warnings</li>
        <li>Fast startup</li>
    </ul>
    <p>This is a test page from <code>/tmp/caddy-dev-test</code></p>
</body>
</html>
EOF

echo "📁 测试站点已创建: $TEST_DIR"
echo ""

# 显示完整测试脚本
echo "🎬 完整测试脚本（可复制运行）:"
echo ""
cat << 'SCRIPT'
# ========== 复制以下内容到终端 ==========

# 终端 1: 启动 Caddy
cd /Users/sunwei/github/mdfriday/hugoverse
/tmp/hugov caddy start

# 等待启动后，在终端 2 运行:

# 测试核心域名
curl http://localhost:8080

# 添加测试站点
/tmp/hugov caddy add -domain test.local -path /tmp/caddy-dev-test

# 配置 hosts
echo "127.0.0.1 test.local" | sudo tee -a /etc/hosts

# 测试站点（注意使用 8080 端口）
curl http://test.local:8080

# 或在浏览器打开
open http://localhost:8080
open http://test.local:8080

# 查看状态
/tmp/hugov caddy status

# 清理
/tmp/hugov caddy remove -domain test.local
sudo sed -i '' '/test.local/d' /etc/hosts

# 停止（在终端 1 按 Ctrl+C）

# =========================================
SCRIPT
echo ""

echo "✅ 验证准备完成！"
echo ""
echo "💡 关键改进:"
echo "   1. 开发模式自动检测（localhost/127.0.0.1）"
echo "   2. 使用 8080 端口（无需 sudo）"
echo "   3. 禁用自动 HTTPS"
echo "   4. 增加启动重试机制"
echo ""
echo "🎯 现在可以运行: /tmp/hugov caddy start"
echo ""

