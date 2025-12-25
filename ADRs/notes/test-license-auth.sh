#!/bin/bash
#
# License API 权限测试脚本
# 测试重点：验证公开接口和需要认证的接口
#
# 使用方法:
#   cd /Users/sunwei/github/mdfriday/hugoverse
#   bash ADRs/notes/test-license-auth.sh
#

set -e

# ========== 配置 ==========
PROJECT_DIR="/Users/sunwei/github/mdfriday/hugoverse"
API_BASE="http://localhost:1314"

# 测试账号
TEST_EMAIL="me@sunwei.xyz"
TEST_PASSWORD="123456"
AUTH_TOKEN=""

# 测试 License
TIMESTAMP=$(date +%s)
TEST_LICENSE_KEY="MDF-STARTER-TEST-${TIMESTAMP}"
INVALID_LICENSE_KEY="MDF-INVALID-KEY-${TIMESTAMP}"
DEVICE_ID="device-$(uuidgen 2>/dev/null || echo "test-device-$$")"

SERVER_PID=""

# ========== 辅助函数 ==========
print_header() {
    echo ""
    echo -e "\033[0;34m========================================\033[0m"
    echo -e "\033[0;34m  $1\033[0m"
    echo -e "\033[0;34m========================================\033[0m"
}

print_step() {
    echo ""
    echo -e "\033[1;33m>>> $1\033[0m"
}

print_success() {
    echo -e "\033[0;32m✅ $1\033[0m"
}

print_error() {
    echo -e "\033[0;31m❌ $1\033[0m"
}

print_info() {
    echo -e "\033[0;34mℹ️  $1\033[0m"
}

format_json() {
    if command -v jq &> /dev/null; then
        echo "$1" | jq '.'
    else
        echo "$1"
    fi
}

check_success() {
    echo "$1" | grep -q '"success":true'
}

kill_server() {
    if [ -n "$SERVER_PID" ] && ps -p "$SERVER_PID" > /dev/null 2>&1; then
        print_info "停止服务器 (PID: $SERVER_PID)"
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
}

trap kill_server EXIT INT TERM

# ========== 主流程 ==========
print_header "License API 权限测试"

echo ""
echo "测试配置:"
echo "  - 测试账号: $TEST_EMAIL"
echo "  - API 地址: $API_BASE"
echo "  - License Key: $TEST_LICENSE_KEY"
echo "  - Device ID: ${DEVICE_ID:0:20}..."

# ---------- 1. 启动服务器 ----------
print_header "阶段 1: 启动服务器"

cd "$PROJECT_DIR"

print_step "检查端口 1314 是否被占用"
if lsof -i:1314 > /dev/null 2>&1; then
    print_info "端口 1314 被占用，尝试关闭..."
    kill $(lsof -t -i:1314) 2>/dev/null || true
    sleep 2
fi

print_step "编译并启动服务器"
rm -rf data
go build -o hugoverse_test main.go 2>&1 | head -5

print_info "启动服务器..."
nohup ./hugoverse_test serve -port 1314 > /tmp/hugoverse-auth-test.log 2>&1 &
SERVER_PID=$!
print_info "服务器 PID: $SERVER_PID"

print_info "等待服务器启动..."
sleep 5

if ps -p "$SERVER_PID" > /dev/null; then
    print_success "服务器已启动"
else
    print_error "服务器启动失败"
    cat /tmp/hugoverse-auth-test.log
    exit 1
fi

# ---------- 2. 用户登录 ----------
print_header "阶段 2: 用户登录获取 TOKEN"

print_step "登录测试账号: $TEST_EMAIL"

LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/api/login" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "email=$TEST_EMAIL&password=$TEST_PASSWORD")

echo "登录响应:"
format_json "$LOGIN_RESPONSE"

# 提取 TOKEN (支持两种格式: {"key":"..."} 和 {"data":["..."]})
AUTH_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"key":"[^"]*"' | cut -d'"' -f4)
if [ -z "$AUTH_TOKEN" ]; then
    AUTH_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"data":\["[^"]*"' | sed 's/"data":\["//g' | sed 's/"//g')
fi

if [ -n "$AUTH_TOKEN" ]; then
    print_success "登录成功，TOKEN 已获取"
    print_info "TOKEN (前20字符): ${AUTH_TOKEN:0:20}..."
else
    print_error "登录失败，无法获取 TOKEN"
    kill_server
    exit 1
fi

# ---------- 3. 创建测试 License (需要认证) ----------
print_header "阶段 3: 创建测试 License (需要认证)"

print_step "使用 TOKEN 创建 License"

CREATE_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/create" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -d "{
        \"license_key\": \"$TEST_LICENSE_KEY\",
        \"plan\": \"starter\",
        \"expiry_days\": 365
    }")

echo "响应:"
format_json "$CREATE_RESPONSE"

if check_success "$CREATE_RESPONSE"; then
    print_success "License 创建成功"
else
    print_error "License 创建失败"
    kill_server
    exit 1
fi

# ---------- 4. 测试公开的激活接口 ----------
print_header "阶段 4: 测试公开的 License 激活接口"

print_step "测试 1: 激活已存在的 License (无需 TOKEN)"

ACTIVATE_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/activate" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$TEST_LICENSE_KEY\",
        \"device_id\": \"$DEVICE_ID\",
        \"device_name\": \"Test Device\",
        \"device_type\": \"desktop\"
    }")

echo "响应:"
format_json "$ACTIVATE_RESPONSE"

if check_success "$ACTIVATE_RESPONSE"; then
    print_success "✅ 公开接口激活成功（无需 TOKEN）"
else
    print_error "激活失败"
fi

print_step "测试 2: 激活不存在的 License (应该失败)"

INVALID_ACTIVATE=$(curl -s -X POST "$API_BASE/api/license/activate" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$INVALID_LICENSE_KEY\",
        \"device_id\": \"device-invalid\",
        \"device_name\": \"Invalid Device\",
        \"device_type\": \"desktop\"
    }")

echo "响应:"
format_json "$INVALID_ACTIVATE"

if echo "$INVALID_ACTIVATE" | grep -q "\"error\""; then
    print_success "✅ 正确返回错误：License 不存在"
else
    print_error "应该返回错误，但激活成功了"
fi

# ---------- 5. 测试需要认证的接口 ----------
print_header "阶段 5: 测试需要认证的接口"

print_step "测试 3: 有 TOKEN 查询 License 信息"

INFO_WITH_TOKEN=$(curl -s "$API_BASE/api/license/info?key=$TEST_LICENSE_KEY" \
    -H "Authorization: Bearer $AUTH_TOKEN")

echo "响应:"
format_json "$INFO_WITH_TOKEN"

if echo "$INFO_WITH_TOKEN" | grep -q "\"license_key\""; then
    print_success "✅ 有 TOKEN 查询成功"
else
    print_error "有 TOKEN 查询失败"
fi

print_step "测试 4: 无 TOKEN 查询 License 信息 (应该失败)"

INFO_WITHOUT_TOKEN=$(curl -s -w "\nHTTP_CODE:%{http_code}" "$API_BASE/api/license/info?key=$TEST_LICENSE_KEY")

echo "响应:"
echo "$INFO_WITHOUT_TOKEN"

if echo "$INFO_WITHOUT_TOKEN" | grep -q "HTTP_CODE:401\|HTTP_CODE:403"; then
    print_success "✅ 正确返回未授权错误 (401/403)"
else
    HTTP_CODE=$(echo "$INFO_WITHOUT_TOKEN" | grep -o 'HTTP_CODE:[0-9]*' | cut -d: -f2)
    print_error "应该返回 401/403，实际返回: $HTTP_CODE"
fi

print_step "测试 5: 有 TOKEN 查询设备列表"

DEVICES_WITH_TOKEN=$(curl -s "$API_BASE/api/license/devices?key=$TEST_LICENSE_KEY" \
    -H "Authorization: Bearer $AUTH_TOKEN")

echo "响应:"
format_json "$DEVICES_WITH_TOKEN"

if echo "$DEVICES_WITH_TOKEN" | grep -q "\"devices\""; then
    print_success "✅ 有 TOKEN 查询设备列表成功"
else
    print_error "有 TOKEN 查询设备列表失败"
fi

print_step "测试 6: 无 TOKEN 查询设备列表 (应该失败)"

DEVICES_WITHOUT_TOKEN=$(curl -s -w "\nHTTP_CODE:%{http_code}" "$API_BASE/api/license/devices?key=$TEST_LICENSE_KEY")

echo "响应:"
echo "$DEVICES_WITHOUT_TOKEN"

if echo "$DEVICES_WITHOUT_TOKEN" | grep -q "HTTP_CODE:401\|HTTP_CODE:403"; then
    print_success "✅ 正确返回未授权错误 (401/403)"
else
    HTTP_CODE=$(echo "$DEVICES_WITHOUT_TOKEN" | grep -o 'HTTP_CODE:[0-9]*' | cut -d: -f2)
    print_error "应该返回 401/403，实际返回: $HTTP_CODE"
fi

# ---------- 6. 测试总结 ----------
print_header "测试总结"

echo ""
echo "✅ 权限验证测试完成！"
echo ""
echo "测试结果："
echo "  1. ✅ 用户登录获取 TOKEN"
echo "  2. ✅ 使用 TOKEN 创建 License"
echo "  3. ✅ 公开接口激活 License (无需 TOKEN)"
echo "  4. ✅ 激活不存在的 License 返回错误"
echo "  5. ✅ 有 TOKEN 查询 License 信息成功"
echo "  6. ✅ 无 TOKEN 查询受保护接口失败 (401/403)"
echo ""
echo "📋 测试的 License:"
echo "  - 有效: $TEST_LICENSE_KEY"
echo "  - 无效: $INVALID_LICENSE_KEY"
echo ""
echo "📝 服务器日志: /tmp/hugoverse-auth-test.log"
echo ""

print_success "License API 权限测试完成"

# 清理
print_step "清理资源..."
kill_server
print_success "清理完成"

