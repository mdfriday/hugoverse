# ========== 构建阶段：编译带 DNSPod / AliDNS 插件的 Caddy ==========
FROM caddy:2.10-builder AS builder

# 设置 GOTOOLCHAIN 为 auto，允许自动下载更高版本的 Go
ENV GOTOOLCHAIN=auto

# 使用 xcaddy 构建带腾讯云 DNS (DNSPod) 与阿里云 DNS (AliDNS) 插件的 Caddy
# 两个插件同时编入是零成本的：只有被 TLS policy 激活的 provider 才会真正用到
RUN xcaddy build \
    --with github.com/caddy-dns/tencentcloud \
    --with github.com/caddy-dns/alidns

# ========== 运行阶段 ==========
FROM caddy:2.10-alpine

# 从构建阶段复制自定义 Caddy
COPY --from=builder /usr/bin/caddy /usr/bin/caddy

# 复制初始 Caddyfile
COPY Caddyfile.initial /etc/caddy/Caddyfile

# 验证 DNSPod / AliDNS 插件均已安装
RUN caddy list-modules | grep dns.providers.tencentcloud || \
    (echo "ERROR: tencentcloud DNS module not found" && exit 1)
RUN caddy list-modules | grep dns.providers.alidns || \
    (echo "ERROR: alidns DNS module not found" && exit 1)

# 创建必要的目录
RUN mkdir -p /data/caddy /config/caddy /srv

# 复制启动脚本
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 设置标签
LABEL maintainer="MDFriday <support@mdfriday.com>"
LABEL description="Caddy with Tencent Cloud DNS (DNSPod) and Alibaba Cloud DNS (AliDNS) plugins for Hugoverse"

# 健康检查（使用 Admin API 检查，因为可能没有 Caddyfile）
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:2019/config/ || exit 1

EXPOSE 80 443 2019

# 使用智能启动脚本
ENTRYPOINT ["/entrypoint.sh"]
