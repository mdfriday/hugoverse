#!/bin/sh
set -e

# Caddy 配置文件路径
AUTOSAVE_FILE="/config/caddy/autosave.json"
INITIAL_CONFIG="/etc/caddy/Caddyfile"

echo "🚀 Starting Caddy..."

# 检查是否存在保存的配置
if [ -f "$AUTOSAVE_FILE" ]; then
    echo "📂 Found autosave.json, resuming from saved configuration..."
    echo "   This will load all dynamically configured routes (custom domains, etc.)"
    exec caddy run --resume
else
    echo "📄 No autosave.json found, loading initial configuration from Caddyfile..."
    echo "   Hugoverse will configure routes via Admin API after startup"
    exec caddy run --config "$INITIAL_CONFIG" --adapter caddyfile
fi
