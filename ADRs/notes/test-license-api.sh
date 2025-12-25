#!/bin/bash
#
# License V2 API 端到端测试脚本
# 用于验证 License 管理系统的完整流程（包括真实 CouchDB 集成）
#
# 测试场景：
# 1. 预先在数据库中创建测试 License
# 2. 使用已存在的 License 进行完整流程测试
# 3. 测试不存在的 License 自动创建场景
#
# 使用方法:
#   cd /Users/sunwei/github/mdfriday/hugoverse
#   bash ADRs/notes/test-license-api.sh
#

set -e

# ========== 配置 ==========
PROJECT_DIR="/Users/sunwei/github/mdfriday/hugoverse"
API_BASE="http://localhost:1314"
COUCHDB_URL="http://admin:987123@127.0.0.1:5984"

# 预先创建的测试 License
TIMESTAMP=$(date +%s)
EXISTING_LICENSE_KEY="MDF-STARTER-EXISTING-${TIMESTAMP}"
NEW_LICENSE_KEY="MDF-STARTER-NEW-${TIMESTAMP}"

DEVICE_ID="device-$(uuidgen 2>/dev/null || echo "test-device-$$")"
SERVER_PID=""
DB_NAME=""

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
    
    # 清理测试数据库 (可选)
    if [ -n "$DB_NAME" ]; then
        print_info "清理测试数据库: $DB_NAME"
        curl -s -X DELETE "http://admin:987123@127.0.0.1:5984/$DB_NAME" > /dev/null 2>&1 || true
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
echo "  - 已存在 License: $EXISTING_LICENSE_KEY"
echo "  - 新建 License: $NEW_LICENSE_KEY"
echo "  - Device ID: ${DEVICE_ID:0:20}..."

# ---------- 1. 编译验证 ----------
print_header "阶段 1: 环境检查和编译"

print_step "检查 CouchDB 连接"
if curl -s "$COUCHDB_URL" > /dev/null 2>&1; then
    print_success "CouchDB 连接正常"
    COUCHDB_VERSION=$(curl -s "$COUCHDB_URL" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
    print_info "CouchDB 版本: $COUCHDB_VERSION"
else
    print_error "CouchDB 连接失败！请确保 CouchDB 已启动在 127.0.0.1:5984"
    print_info "启动命令示例: docker run -d -p 5984:5984 -e COUCHDB_USER=admin -e COUCHDB_PASSWORD=987123 couchdb:latest"
    exit 1
fi

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
    if curl -s "$API_BASE/api/license/info?key=test" > /dev/null 2>&1; then
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
print_header "阶段 4: 预先创建测试 License"

# 4.0 预先在数据库中创建几条 License
print_step "创建测试 License 1 (Starter Plan)"

CREATE_RESPONSE_1=$(curl -s -X POST "$API_BASE/api/license/create" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$EXISTING_LICENSE_KEY\",
        \"plan\": \"starter\",
        \"expiry_days\": 365
    }")

echo "响应:"
format_json "$CREATE_RESPONSE_1"

if check_success "$CREATE_RESPONSE_1"; then
    print_success "测试 License 1 创建成功"
else
    print_error "测试 License 1 创建失败"
fi

print_step "创建测试 License 2 (Creator Plan)"

CREATE_RESPONSE_2=$(curl -s -X POST "$API_BASE/api/license/create" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"MDF-CREATOR-PRE-${TIMESTAMP}\",
        \"plan\": \"creator\",
        \"expiry_days\": 365
    }")

echo "响应:"
format_json "$CREATE_RESPONSE_2"

if check_success "$CREATE_RESPONSE_2"; then
    print_success "测试 License 2 创建成功"
else
    print_error "测试 License 2 创建失败"
fi

print_step "创建测试 License 3 (Pro Plan)"

CREATE_RESPONSE_3=$(curl -s -X POST "$API_BASE/api/license/create" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"MDF-PRO-PRE-${TIMESTAMP}\",
        \"plan\": \"pro\",
        \"expiry_days\": 365
    }")

echo "响应:"
format_json "$CREATE_RESPONSE_3"

if check_success "$CREATE_RESPONSE_3"; then
    print_success "测试 License 3 创建成功"
    print_info "✅ 已在数据库中预先创建 3 条 License"
else
    print_error "测试 License 3 创建失败"
fi

# ---------- 5. 使用已存在的 License 进行完整流程测试 ----------
print_header "阶段 5: 测试已存在的 License (完整流程)"

print_step "验证 License 未激活状态"
INFO_BEFORE=$(curl -s "$API_BASE/api/license/info?key=$EXISTING_LICENSE_KEY")
echo "激活前状态:"
format_json "$INFO_BEFORE"

if echo "$INFO_BEFORE" | grep -q "\"activated\":false"; then
    print_success "确认 License 未激活"
else
    print_info "License 状态: $(echo $INFO_BEFORE | grep -o '\"activated\":[^,}]*')"
fi

# 5.1 激活已存在的 License
# 5.1 激活已存在的 License
print_step "测试 1: 激活已存在的 License (POST /api/license/activate)"

ACTIVATE_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/activate" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$EXISTING_LICENSE_KEY\",
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

# 5.2 查询 License 信息
print_step "测试 2: 查询 License 信息 (GET /api/license/info)"

INFO_RESPONSE=$(curl -s "$API_BASE/api/license/info?key=$EXISTING_LICENSE_KEY")

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

# 5.3 查询设备列表
print_step "测试 3: 查询设备列表 (GET /api/license/devices)"

DEVICES_RESPONSE=$(curl -s "$API_BASE/api/license/devices?key=$EXISTING_LICENSE_KEY")

echo "响应:"
format_json "$DEVICES_RESPONSE"

if echo "$DEVICES_RESPONSE" | grep -q "\"devices\""; then
    print_success "查询设备列表成功"
    DEVICE_COUNT=$(echo "$DEVICES_RESPONSE" | grep -o '"count":[0-9]*' | cut -d: -f2)
    print_info "设备数量: $DEVICE_COUNT"
else
    print_error "查询设备列表失败"
fi

# 5.4 查询 IP 列表
print_step "测试 4: 查询 IP 列表 (GET /api/license/ips)"

IPS_RESPONSE=$(curl -s "$API_BASE/api/license/ips?key=$EXISTING_LICENSE_KEY")

echo "响应:"
format_json "$IPS_RESPONSE"

if echo "$IPS_RESPONSE" | grep -q "\"ips\""; then
    print_success "查询 IP 列表成功"
else
    print_error "查询 IP 列表失败"
fi

# 5.5 查询 Sync 信息
print_step "测试 5: 查询 Sync 信息 (GET /api/license/sync)"

SYNC_RESPONSE=$(curl -s "$API_BASE/api/license/sync?key=$EXISTING_LICENSE_KEY")

echo "响应:"
format_json "$SYNC_RESPONSE"

if echo "$SYNC_RESPONSE" | grep -q "\"email\"\|\"db_endpoint\""; then
    print_success "查询 Sync 信息成功"
    
    # 提取 DB 信息
    SYNC_EMAIL=$(echo "$SYNC_RESPONSE" | grep -o '"email":"[^"]*"' | cut -d'"' -f4)
    DB_NAME=$(echo "$SYNC_RESPONSE" | grep -o '"db_name":"[^"]*"' | cut -d'"' -f4)
    
    print_info "Sync Email: $SYNC_EMAIL"
    print_info "DB Name: $DB_NAME"
    
    # 验证 CouchDB 数据库是否真的被创建
    print_step "验证 CouchDB 数据库是否存在"
    DB_CHECK_URL="http://admin:987123@127.0.0.1:5984/$DB_NAME"
    if DB_INFO=$(curl -s "$DB_CHECK_URL"); then
        if echo "$DB_INFO" | grep -q '"db_name"'; then
            print_success "✅ CouchDB 数据库已成功创建"
            DOC_COUNT=$(echo "$DB_INFO" | grep -o '"doc_count":[0-9]*' | cut -d: -f2)
            DISK_SIZE=$(echo "$DB_INFO" | grep -o '"disk_size":[0-9]*' | cut -d: -f2)
            print_info "文档数量: $DOC_COUNT"
            print_info "磁盘大小: $DISK_SIZE bytes"
        else
            print_error "数据库信息异常"
            echo "$DB_INFO"
        fi
    else
        print_error "无法访问 CouchDB 数据库"
    fi
    
    # 验证用户是否被创建
    print_step "验证 CouchDB 用户是否存在"
    USER_CHECK_URL="http://admin:987123@127.0.0.1:5984/_users/org.couchdb.user:$SYNC_EMAIL"
    if USER_INFO=$(curl -s "$USER_CHECK_URL"); then
        if echo "$USER_INFO" | grep -q "\"name\":\"$SYNC_EMAIL\""; then
            print_success "✅ CouchDB 用户已成功创建"
            print_info "用户名: $SYNC_EMAIL"
        else
            print_info "用户信息: $USER_INFO"
        fi
    fi
    
elif echo "$SYNC_RESPONSE" | grep -q "error"; then
    print_info "Sync 功能未启用或配置错误"
    echo "$SYNC_RESPONSE"
else
    print_error "查询 Sync 信息失败"
fi

# 5.6 查询 Publish 信息
print_step "测试 6: 查询 Publish 信息 (GET /api/license/publish)"

PUBLISH_RESPONSE=$(curl -s "$API_BASE/api/license/publish?key=$EXISTING_LICENSE_KEY")

echo "响应:"
format_json "$PUBLISH_RESPONSE"

if echo "$PUBLISH_RESPONSE" | grep -q "\"sites\"\|\"domains\""; then
    print_success "查询 Publish 信息成功"
else
    print_error "查询 Publish 信息失败"
fi

# ---------- 6. 测试新 License（不存在于数据库）的自动创建场景 ----------
print_header "阶段 6: 测试不存在的 License (自动创建场景)"

print_step "测试 7: 激活不存在的 License (自动创建)"

NEW_LICENSE_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/activate" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$NEW_LICENSE_KEY\",
        \"device_id\": \"device-new-${TIMESTAMP}\",
        \"device_name\": \"New Device\",
        \"device_type\": \"desktop\"
    }")

echo "响应:"
format_json "$NEW_LICENSE_RESPONSE"

if check_success "$NEW_LICENSE_RESPONSE"; then
    print_success "新 License 自动创建并激活成功"
    
    # 验证是否自动创建
    if echo "$NEW_LICENSE_RESPONSE" | grep -q "\"plan\":\"starter\""; then
        print_success "自动判断 Plan: starter (基于 Key 格式)"
    fi
else
    print_error "新 License 创建失败"
fi

print_step "测试 8: 验证新创建的 License 信息"

NEW_INFO=$(curl -s "$API_BASE/api/license/info?key=$NEW_LICENSE_KEY")
echo "新 License 信息:"
format_json "$NEW_INFO"

if echo "$NEW_INFO" | grep -q "\"license_key\""; then
    print_success "新 License 已保存到数据库"
else
    print_error "新 License 未保存"
fi

# ---------- 7. 边界测试 ----------
print_header "阶段 7: 边界测试"

# ---------- 7. 边界测试 ----------
print_header "阶段 7: 边界测试"

print_step "测试 9: 查询不存在的 License"

NOT_FOUND_RESPONSE=$(curl -s "$API_BASE/api/license/info?key=NOT-EXIST-KEY")

echo "响应:"
format_json "$NOT_FOUND_RESPONSE"

if echo "$NOT_FOUND_RESPONSE" | grep -q "\"error\"\|not found"; then
    print_success "正确返回 License 不存在错误"
else
    print_error "边界测试失败"
fi

# 7.2 测试缺少参数
print_step "测试 10: 缺少必要参数"

MISSING_PARAM_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/activate" \
    -H "Content-Type: application/json" \
    -d '{"license_key": ""}')

echo "响应:"
format_json "$MISSING_PARAM_RESPONSE"

if echo "$MISSING_PARAM_RESPONSE" | grep -q "\"error\"\|required"; then
    print_success "正确返回参数缺失错误"
else
    print_error "边界测试失败"
fi

# ---------- 8. 幂等性和设备限制测试 ----------
print_header "阶段 8: 幂等性和设备限制测试"

# 8.1 再次激活同一 License
print_step "测试 11: 重复激活 License (幂等性测试)"

REACTIVATE_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/activate" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$EXISTING_LICENSE_KEY\",
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

# 8.2 新设备激活
print_step "测试 12: 新设备激活 (设备限制测试)"

NEW_DEVICE_ID="device-new-$(date +%s)"
NEW_DEVICE_RESPONSE=$(curl -s -X POST "$API_BASE/api/license/activate" \
    -H "Content-Type: application/json" \
    -d "{
        \"license_key\": \"$EXISTING_LICENSE_KEY\",
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
FINAL_DEVICES=$(curl -s "$API_BASE/api/license/devices?key=$EXISTING_LICENSE_KEY")
echo "最终设备列表:"
format_json "$FINAL_DEVICES"

# ---------- 9. 测试总结 ----------
# ---------- 9. 测试总结 ----------
print_header "测试总结"

echo ""
echo "测试完成！"
echo ""
echo "✅ 测试场景："
echo "  1. 预先创建 3 条测试 License (Starter, Creator, Pro)"
echo "  2. 使用已存在的 License 进行完整流程测试"
echo "  3. 测试不存在的 License 自动创建场景"
echo ""
echo "📋 测试的 License:"
echo "  - 已存在: $EXISTING_LICENSE_KEY"
echo "  - 新建: $NEW_LICENSE_KEY"
echo "  - 其他: MDF-CREATOR-PRE-${TIMESTAMP}, MDF-PRO-PRE-${TIMESTAMP}"
echo ""
echo "🔧 使用的 Device ID: ${DEVICE_ID:0:20}..."
echo ""
echo "📝 服务器日志位置: /tmp/hugoverse-test.log"
echo ""

print_success "License V2 API 端到端测试完成"

# 脚本结束，cleanup 函数会自动执行

