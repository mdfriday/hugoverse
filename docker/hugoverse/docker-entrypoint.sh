#!/bin/bash
set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Hugoverse Docker Container Starting"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📦 Version: ${VERSION:-latest}"
echo "🐳 Environment: Docker"
echo "📁 Data Directory: ${HUGOVERSE_DATA_DIR:-/data}"
echo "🕒 Timezone: $(date +%Z) ($(date +%Y-%m-%d\ %H:%M:%S))"
echo ""

# ========== 函数：等待服务就绪 ==========
wait_for_service() {
    local name=$1
    local url=$2
    local max_attempts=30
    local attempt=1
    
    echo "⏳ Waiting for $name..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "$url" > /dev/null 2>&1; then
            echo "✅ $name is ready"
            return 0
        fi
        
        if [ $((attempt % 5)) -eq 0 ]; then
            echo "   Still waiting for $name (attempt $attempt/$max_attempts)..."
        fi
        
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "❌ Timeout waiting for $name after $max_attempts attempts"
    return 1
}

# ========== 等待依赖服务 ==========
echo "🔍 Checking dependencies..."
echo ""

# 等待 CouchDB
if [ -n "$COUCHDB_URL" ]; then
    wait_for_service "CouchDB" "$COUCHDB_URL/_up" || exit 1
    
    # 初始化 CouchDB 系统数据库
    echo "🔧 Initializing CouchDB system databases..."
    COUCH_USER=${COUCHDB_USER:-admin}
    COUCH_PASS=${COUCHDB_PASSWORD:-test123456}
    
    # 设置单节点集群
    curl -X POST "${COUCHDB_URL}/_cluster_setup" \
        -H "Content-Type: application/json" \
        -u "${COUCH_USER}:${COUCH_PASS}" \
        -d '{"action":"finish_cluster"}' \
        > /dev/null 2>&1 || echo "   Cluster setup already done"
    
    # 创建系统数据库
    for db in _users _replicator _global_changes; do
        if ! curl -s "${COUCHDB_URL}/${db}" -u "${COUCH_USER}:${COUCH_PASS}" | grep -q "db_name"; then
            curl -X PUT "${COUCHDB_URL}/${db}" -u "${COUCH_USER}:${COUCH_PASS}" > /dev/null 2>&1 || true
            echo "   ✅ Created $db"
        fi
    done
    
    echo "✅ CouchDB initialized"
else
    echo "⚠️  COUCHDB_URL not set, skipping CouchDB check"
fi

# 等待 Caddy Admin API
if [ -n "$CADDY_ADMIN_API" ]; then
    wait_for_service "Caddy Admin API" "$CADDY_ADMIN_API/config/" || exit 1
else
    echo "⚠️  CADDY_ADMIN_API not set, skipping Caddy check"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⚙️  Configuration Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 显示配置
if [ "$AUTO_INIT" = "true" ]; then
    echo "🔧 Mode: Auto-initialization"
    echo "   Domain: ${DOMAIN:-not set}"
    echo "   Admin: ${ADMIN_EMAIL:-not set}"
    echo "   CouchDB: ${COUCHDB_URL:-not set}"
    echo "   Caddy: ${CADDY_ADMIN_API:-not set}"
    echo ""
    
    # Master License 信息
    if [ -n "$MASTER_LICENSE" ]; then
        echo "🔑 Master License: Provided (will verify online)"
    else
        echo "🆓 Master License: Not provided (FREE mode - 1 license)"
    fi
    echo ""
    
    # DNSPod 配置
    if [ "$DNSPOD_ENABLED" = "true" ] && [ -n "$DNSPOD_ID" ]; then
        echo "🌐 DNSPod: ✅ Enabled (wildcard SSL for *.${DOMAIN})"
    else
        echo "🌐 DNSPod: ❌ Disabled (HTTP-01 validation only)"
    fi
    echo ""
    
    # 企业功能
    if [ "$AUTO_GENERATE_ENTERPRISE_LICENSE" = "true" ]; then
        echo "🏢 Enterprise License: ✅ Auto-generate"
        echo "   Plan: ${ENTERPRISE_LICENSE_PLAN:-enterprise}"
        echo "   Count: ${ENTERPRISE_LICENSE_COUNT:-1}"
    fi
    
    if [ "$AUTO_CONFIGURE_ENTERPRISE_SITE" = "true" ]; then
        echo "📁 Enterprise Site: ✅ Auto-configure"
        echo "   Domain: ${ENTERPRISE_SITE_DOMAIN:-$DOMAIN}"
    fi
    
    echo ""
    echo "📍 After initialization, access:"
    echo "   Admin Panel: http://${DOMAIN}/admin"
    if [ "$DNSPOD_ENABLED" = "true" ]; then
        echo "   CouchDB: https://cdb.${DOMAIN}"
    else
        echo "   CouchDB: http://cdb.${DOMAIN}"
    fi
else
    echo "🛠️  Mode: Manual configuration"
    echo ""
    echo "📍 Please visit to complete setup:"
    echo "   http://${DOMAIN:-localhost}/admin/init"
fi

echo ""
echo "💡 Need more licenses?"
echo "   Visit: https://mdfriday.com/pricing"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✨ Starting Hugoverse Service"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 启动 Hugoverse
exec /app/hugoverse "$@"
