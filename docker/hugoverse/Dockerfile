# ========== 构建阶段 ==========
FROM golang:1.23-alpine AS builder

WORKDIR /build

# 设置 GOTOOLCHAIN 为 auto，允许自动升级到 Go 1.25
ENV GOTOOLCHAIN=auto

# 安装构建依赖（包括 vips 及其开发库）
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    pkgconf \
    vips-dev \
    glib-dev \
    expat-dev \
    libjpeg-turbo-dev \
    libpng-dev \
    tiff-dev \
    libwebp-dev \
    libheif-dev \
    poppler-dev \
    libexif-dev

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用（启用 CGO 以支持 BBolt 和 vips）
# 不指定 GOARCH，使用当前平台架构
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o hugoverse \
    ./cmd/hugoverse

# 验证构建结果
RUN chmod +x hugoverse && ./hugoverse version || echo "Version check skipped"

# ========== 运行阶段 ==========
FROM alpine:3.20

# 安装运行时依赖（包括 vips 运行库）
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    bash \
    vips \
    glib \
    expat \
    libjpeg-turbo \
    libpng \
    tiff \
    libwebp \
    libheif \
    poppler-glib \
    libexif

# 设置时区为中国
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 hugoverse && \
    adduser -D -u 1000 -G hugoverse hugoverse

# 创建数据目录
RUN mkdir -p /data /backups && \
    chown -R hugoverse:hugoverse /data /backups

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/hugoverse /app/hugoverse

# 从构建上下文复制启动脚本（不是从 builder 阶段）
COPY docker/hugoverse/docker-entrypoint.sh /app/

# 设置权限
RUN chmod +x /app/hugoverse /app/docker-entrypoint.sh && \
    chown -R hugoverse:hugoverse /app

# 设置 Docker 环境标识
ENV DOCKER_CONTAINER=true
ENV HUGOVERSE_DATA_DIR=/data

# 设置标签
LABEL maintainer="MDFriday <support@mdfriday.com>"
LABEL description="Hugoverse - Self-hosted Obsidian Sync & Publish Platform"

# 切换到非 root 用户
USER hugoverse

# 暴露端口
EXPOSE 1314

# 数据卷
VOLUME ["/data", "/backups"]

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=60s \
  CMD curl -f http://localhost:1314/api/health || exit 1

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["server"]
