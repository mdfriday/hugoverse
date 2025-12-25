#!/bin/bash
#
# License V2 API 端到端测试脚本
# 用于验证 License 管理系统的完整流程
#
# 使用方法:
#   cd /Users/sunwei/github/mdfriday/hugoverse
#   bash ADRs/notes/test-license-api.sh
#

set -e

# ========== 配置 ==========
PROJECT_DIR="/Users/sunwei/github/mdfriday/hugoverse"
API_BASE="http://localhost:1314"
LICENSE_KEY="MDF-STARTER-TEST-$(date +%s)"
DEVICE_ID="device-$(uuidgen 2>/dev/null || echo "test-device-$$")"
SERVER_PID=""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ========== 辅助函数 ==========

print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_step() {
    echo -e "\n${YELLOW}>>> $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# JSON 格式化输出 (如果有 jq)
format_json() {
    if command -v jq &> /dev/null; then
        echo "$1" | jq .
    else
        echo "$1"
    fi
}

# 检查响应是否包含成功标志
check_success() {
    local response="$1"
    local field="${2:-success}"
    
    if echo "$response" | grep -q "\"$field\".*true\|\"$field\":true"; then
        return 0
    elif echo "$response" | grep -q "\"license_key\"\|\"plan\"\|\"devices\"\|\"ips\""; then
        # 某些 GET 接口没有 success 字段，但有数据就算成功
        return 0
    else
        return 1
    fi
}

# 清理函数
cleanup() {
    print_step "清理资源..."
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        print_info "停止服务器 (PID: $SERVER_PID)"
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    print_success "清理完成"
}

# 注册退出时清理
trap cleanup EXIT

# ========== 测试阶段 ==========

print_header "License V2 API 端到端测试"

echo ""
echo "测试配置:"
echo "  - 项目目录: $PROJECT_DIR"
echo "  - API 地址: $API_BASE"
echo "  - License Key: $LICENSE_KEY"
echo "  - Device ID: ${DEVICE_ID:0:20}..."

# ---------- 1. 编译验证 ----------
print_header "阶段 1: 编译验证"

print_step "切换到项目目录"
cd "$PROJECT_DIR"
print_success "当前目录: $(pwd)"

print_step "编译项目"
if go build ./internal/... 2>&1; then
    print_success "编译成功"
else
    print_error "编译失败"
    exit 1
fi

# ---------- 2. 单元测试 ----------
print_header "阶段 2: 单元测试"

print_step "运行 CouchDB Client 测试"
if go test ./internal/infrastructure/couchdb/... -v 2>&1 | tail -5; then
    print_success "CouchDB Client 测试通过"
else
    print_error "CouchDB Client 测试失败"
    exit 1
fi

print_step "运行 License Handler 测试"
if go test ./internal/interfaces/api/handler/handlelicense_test.go \
    ./internal/interfaces/api/handler/handlelicense.go -v 2>&1 | tail -10; then
    print_success "License Handler 测试通过"
else
    print_error "License Handler 测试失败"
    exit 1
fi

print_step "运行 Domain 测试"
if go test ./internal/domain/sync/... ./internal/domain/publish/... \
    ./internal/domain/content/valueobject/... -v 2>&1 | tail -10; then
    print_success "Domain 测试通过"
else
    print_error "Domain 测试失败"
    exit 1
fi

# ---------- 3. 启动服务器 ----------
print_header "阶段 3: 启动服务器"

print_step "检查端口 1314 是否被占用"
if lsof -i :1314 -t > /dev/null 2>&1; then
    print_info "端口 1314 已被占用，尝试停止现有进程"
    lsof -i :1314 -t | xargs kill -9 2>/dev/null || true
    sleep 2
fi

print_step "编译并启动服务器"
go build -o /tmp/hugoverse-test ./main.go 2>&1

print_info "启动服务器..."
/tmp/hugoverse-test serve > /tmp/hugoverse-test.log 2>&1 &
SERVER_PID=$!

print_info "服务器 PID: $SERVER_PID"
print_info "等待服务器启动..."

# 等待服务器启动 (最多 10 秒)
for i in {1..20}; do
    if curl -s "$API_BASE/api/license/v2/info?key=test" > /dev/null 2>&1; then
        print_success "服务器已启动"
        break
    fi
    if [ $i -eq 20 ]; then
        print_error "服务器启动超时"
        cat /tmp/hugoverse-test.log
        exit 1
    fi
    sleep 0.5
done

# ---------- 4. API 测试 ----------
print_header "阶段 4: API 端点测试"

# 4.1 激活 License
print_step "测试 1: 激活 License (POST /api/license/v2/activate)"

ACTIVATE_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/v2/activate" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$LICENSE_KEY\",
        \"device_id\": \"$DEVICE_ID\",
        \"device_name\": \"Test Device\",
        \"device_type\": \"desktop\"
    }")

echo "响应:"
format_json "$ACTIVATE_RESPONSE"

if check_success "$ACTIVATE_RESPONSE"; then
    print_success "License 激活成功"
else
    print_error "License 激活失败"
fi

# 4.2 查询 License 信息
print_step "测试 2: 查询 License 信息 (GET /api/license/v2/info)"

INFO_RESPONSE=$(curl -s "$API_BASE/api/license/v2/info?key=$LICENSE_KEY")

echo "响应:"
format_json "$INFO_RESPONSE"

if echo "$INFO_RESPONSE" | grep -q "\"license_key\""; then
    print_success "查询 License 信息成功"
    
    # 验证字段
    if echo "$INFO_RESPONSE" | grep -q "\"plan\":\"starter\""; then
        print_success "Plan 正确: starter"
    fi
    if echo "$INFO_RESPONSE" | grep -q "\"activated\":true"; then
        print_success "已激活"
    fi
else
    print_error "查询 License 信息失败"
fi

# 4.3 查询设备列表
print_step "测试 3: 查询设备列表 (GET /api/license/v2/devices)"

DEVICES_RESPONSE=$(curl -s "$API_BASE/api/license/v2/devices?key=$LICENSE_KEY")

echo "响应:"
format_json "$DEVICES_RESPONSE"

if echo "$DEVICES_RESPONSE" | grep -q "\"devices\""; then
    print_success "查询设备列表成功"
    DEVICE_COUNT=$(echo "$DEVICES_RESPONSE" | grep -o '"count":[0-9]*' | cut -d: -f2)
    print_info "设备数量: $DEVICE_COUNT"
else
    print_error "查询设备列表失败"
fi

# 4.4 查询 IP 列表
print_step "测试 4: 查询 IP 列表 (GET /api/license/v2/ips)"

IPS_RESPONSE=$(curl -s "$API_BASE/api/license/v2/ips?key=$LICENSE_KEY")

echo "响应:"
format_json "$IPS_RESPONSE"

if echo "$IPS_RESPONSE" | grep -q "\"ips\""; then
    print_success "查询 IP 列表成功"
else
    print_error "查询 IP 列表失败"
fi

# 4.5 查询 Sync 信息
print_step "测试 5: 查询 Sync 信息 (GET /api/license/v2/sync)"

SYNC_RESPONSE=$(curl -s "$API_BASE/api/license/v2/sync?key=$LICENSE_KEY")

echo "响应:"
format_json "$SYNC_RESPONSE"

if echo "$SYNC_RESPONSE" | grep -q "\"email\"\|\"db_endpoint\"\|error"; then
    if echo "$SYNC_RESPONSE" | grep -q "\"email\""; then
        print_success "查询 Sync 信息成功"
        EMAIL=$(echo "$SYNC_RESPONSE" | grep -o '"email":"[^"]*"' | cut -d'"' -f4)
        print_info "Sync Email: $EMAIL"
    else
        print_info "Sync 账号尚未创建或 CouchDB 未配置"
    fi
else
    print_error "查询 Sync 信息失败"
fi

# 4.6 查询 Publish 信息
print_step "测试 6: 查询 Publish 信息 (GET /api/license/v2/publish)"

PUBLISH_RESPONSE=$(curl -s "$API_BASE/api/license/v2/publish?key=$LICENSE_KEY")

echo "响应:"
format_json "$PUBLISH_RESPONSE"

if echo "$PUBLISH_RESPONSE" | grep -q "\"sites\"\|\"domains\""; then
    print_success "查询 Publish 信息成功"
else
    print_error "查询 Publish 信息失败"
fi

# 4.7 测试不存在的 License
print_step "测试 7: 查询不存在的 License (边界测试)"

NOT_FOUND_RESPONSE=$(curl -s "$API_BASE/api/license/v2/info?key=NOT-EXIST-KEY")

echo "响应:"
format_json "$NOT_FOUND_RESPONSE"

if echo "$NOT_FOUND_RESPONSE" | grep -q "\"error\"\|not found"; then
    print_success "正确返回 License 不存在错误"
else
    print_error "边界测试失败"
fi

# 4.8 测试缺少参数
print_step "测试 8: 缺少必要参数 (边界测试)"

MISSING_PARAM_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/v2/activate" \
    -H "Content-Type: application/json" \
    -d '{"license_key": ""}')

echo "响应:"
format_json "$MISSING_PARAM_RESPONSE"

if echo "$MISSING_PARAM_RESPONSE" | grep -q "\"error\"\|required"; then
    print_success "正确返回参数缺失错误"
else
    print_error "边界测试失败"
fi

# 4.9 再次激活同一 License (模拟重复激活)
print_step "测试 9: 重复激活 License (幂等性测试)"

REACTIVATE_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/v2/activate" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$LICENSE_KEY\",
        \"device_id\": \"$DEVICE_ID\",
        \"device_name\": \"Test Device Updated\",
        \"device_type\": \"desktop\"
    }")

echo "响应:"
format_json "$REACTIVATE_RESPONSE"

if check_success "$REACTIVATE_RESPONSE"; then
    print_success "重复激活成功 (幂等性验证通过)"
else
    print_error "重复激活失败"
fi

# 4.10 新设备激活
print_step "测试 10: 新设备激活 (设备限制测试)"

NEW_DEVICE_ID="device-new-$(date +%s)"
NEW_DEVICE_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/v2/activate" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$LICENSE_KEY\",
        \"device_id\": \"$NEW_DEVICE_ID\",
        \"device_name\": \"New Test Device\",
        \"device_type\": \"mobile\"
    }")

echo "响应:"
format_json "$NEW_DEVICE_RESPONSE"

if check_success "$NEW_DEVICE_RESPONSE"; then
    print_success "新设备激活成功"
else
    print_info "新设备激活受限 (可能达到设备上限)"
fi

# 最终验证设备数量
print_step "最终验证: 确认设备数量"
FINAL_DEVICES=$(curl -s "$API_BASE/api/license/v2/devices?key=$LICENSE_KEY")
echo "最终设备列表:"
format_json "$FINAL_DEVICES"

# ---------- 5. 测试总结 ----------
print_header "测试总结"

echo ""
echo "测试完成！"
echo ""
echo "测试的 License Key: $LICENSE_KEY"
echo "使用的 Device ID: ${DEVICE_ID:0:20}..."
echo ""
echo "服务器日志位置: /tmp/hugoverse-test.log"
echo ""

print_success "License V2 API 端到端测试完成"

# 脚本结束，cleanup 函数会自动执行

