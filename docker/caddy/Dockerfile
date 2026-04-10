# ========== 构建阶段：编译带 DNSPod 插件的 Caddy ==========
FROM caddy:2.10-builder AS builder

# 设置 GOTOOLCHAIN 为 auto，允许自动下载更高版本的 Go
ENV GOTOOLCHAIN=auto

# 使用 xcaddy 构建带 Tencent Cloud DNS 插件的 Caddy
# 使用最新版本，Go 会自动升级到所需版本
RUN xcaddy build \
    --with github.com/caddy-dns/tencentcloud

# ========== 运行阶段 ==========
FROM caddy:2.10-alpine

# 从构建阶段复制自定义 Caddy
COPY --from=builder /usr/bin/caddy /usr/bin/caddy

# 验证 DNSPod 插件已安装
RUN caddy list-modules | grep dns.providers.tencentcloud || \
    (echo "ERROR: tencentcloud DNS module not found" && exit 1)

# 创建必要的目录
RUN mkdir -p /data/caddy /config/caddy /srv

# 设置标签
LABEL maintainer="MDFriday <support@mdfriday.com>"
LABEL description="Caddy with Tencent Cloud DNS plugin for Hugoverse"

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD caddy validate --config /etc/caddy/Caddyfile || exit 1

EXPOSE 80 443 2019

# 使用 Caddyfile 启动
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
