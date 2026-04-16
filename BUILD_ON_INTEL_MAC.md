# 在 Intel Mac 上构建 AMD64 镜像

## 为什么 Intel Mac 是最佳选择？

| 设备 | 架构 | 跨平台需求 | 构建速度 | 推荐度 |
|------|------|-----------|---------|--------|
| Apple Silicon Mac | ARM64 | ✅ 需要 | 慢（10-15分钟） | ⭐⭐ |
| Intel Mac | **AMD64** | ❌ **不需要** | **快（3-5分钟）** | ⭐⭐⭐⭐⭐ |
| 云服务器 | AMD64 | ❌ 不需要 | 快（3-5分钟） | ⭐⭐⭐⭐⭐ |

## Intel Mac 的优势

✅ **原生架构**：直接构建 AMD64 镜像，与服务器架构一致  
✅ **构建速度快**：原生构建，比 Apple Silicon 跨平台快 3-5 倍  
✅ **无需 buildx**：不需要跨平台工具和 QEMU 模拟  
✅ **本地环境**：无需 SSH 到服务器，在本地即可完成  
✅ **Docker Desktop**：可以使用完整的 Docker Desktop（无 Colima 问题）

## 操作步骤

### 1. 安装 Docker Desktop（推荐）

```bash
# 下载并安装 Docker Desktop for Mac (Intel)
# https://www.docker.com/products/docker-desktop/

# 验证安装
docker version
docker info
```

### 2. 克隆代码

```bash
# 克隆仓库
git clone https://github.com/mdfriday/hugoverse.git
cd hugoverse
```

### 3. 验证架构

```bash
# 确认是 AMD64/x86_64 架构
uname -m
# 应该显示: x86_64

# 确认 Docker 架构
docker version --format '{{.Server.Arch}}'
# 应该显示: amd64
```

### 4. 构建镜像（原生 AMD64）

```bash
# 拉取 CouchDB
make aliyun-pull-couchdb

# 构建所有镜像（原生 AMD64，速度快！）
make docker-build-all VERSION=26.4.2

# 验证架构（应该都是 amd64）
docker inspect mdfriday/caddy:26.4.2 --format '{{.Architecture}}'
docker inspect mdfriday/hugoverse:26.4.2 --format '{{.Architecture}}'
docker inspect couchdb:3.3 --format '{{.Architecture}}'
```

### 5. 推送到阿里云

```bash
# 登录阿里云镜像仓库
make aliyun-login

# 推送所有镜像
make aliyun-push-couchdb
make aliyun-push-all VERSION=26.4.2
```

### 6. 验证推送

```bash
# 拉取并验证
docker pull registry.cn-hangzhou.aliyuncs.com/mdfriday/caddy:26.4.2

# 检查架构
docker inspect registry.cn-hangzhou.aliyuncs.com/mdfriday/caddy:26.4.2 \
  --format '{{.Architecture}}'
# 应该显示: amd64
```

## 一键脚本

在 Intel Mac 上直接使用项目提供的脚本：

```bash
# 构建并推送（自动完成所有步骤）
./build-and-push.sh 26.4.2
```

脚本会：
1. ✅ 拉取 CouchDB 3.3
2. ✅ 构建 Caddy 和 Hugoverse（原生 AMD64）
3. ✅ 验证架构
4. ✅ 推送到阿里云
5. ✅ 验证推送结果

## 注意事项

### 1. 无需 PLATFORM 参数

Intel Mac 原生就是 AMD64，不需要指定平台：

```bash
# ✅ 正确（原生构建）
make docker-build-all VERSION=26.4.2

# ⚠️ 不需要（多此一举）
make docker-build-all VERSION=26.4.2 PLATFORM=linux/amd64
```

### 2. 无需 buildx

Intel Mac 原生构建，不需要 buildx：

```bash
# Makefile 会自动检测并使用 docker build
# 速度快，无需 QEMU 模拟
```

### 3. Docker Desktop vs Colima

在 Intel Mac 上，推荐使用 Docker Desktop：

| 工具 | 优势 | 劣势 |
|------|------|------|
| **Docker Desktop** | 稳定、GUI、无配置问题 | 大公司需要付费 |
| Colima | 免费开源 | 可能有配置问题 |

如果是个人使用或小团队，Docker Desktop 更省心。

## 完整流程

```bash
# 在 Intel Mac 上执行

# 1. 验证架构
uname -m  # 应该显示 x86_64

# 2. 克隆代码
git clone https://github.com/mdfriday/hugoverse.git
cd hugoverse

# 3. 一键构建并推送
./build-and-push.sh 26.4.2

# 完成！🎉
```

## 与其他方案对比

### 构建速度

| 设备 | 时间 | 说明 |
|------|------|------|
| Apple Silicon Mac | 10-15 分钟 | 需要 QEMU 模拟 AMD64 |
| **Intel Mac** | **3-5 分钟** | ✅ **原生构建** |
| 云服务器 | 3-5 分钟 | 原生构建 |

### 便利性

| 设备 | SSH 需求 | 网络要求 | 本地开发 |
|------|----------|---------|---------|
| Apple Silicon Mac | ❌ | ❌ | ✅ |
| **Intel Mac** | ❌ | ❌ | ✅ |
| 云服务器 | ✅ 需要 | ✅ 稳定连接 | ❌ |

### 推荐场景

| 场景 | 推荐设备 | 原因 |
|------|---------|------|
| **快速构建发布** | **Intel Mac** | 本地 + 原生 + 快速 |
| 自动化 CI/CD | 云服务器 / GitHub Actions | 自动化 |
| 日常开发测试 | 任意 Mac | 不需要推送 |

## 总结

**Intel Mac 是最佳本地构建环境**：

✅ 原生 AMD64 架构  
✅ 构建速度快（3-5 分钟）  
✅ 无需跨平台工具  
✅ 本地操作，无需 SSH  
✅ 可用 Docker Desktop  

### 快速开始

```bash
cd ~/github/mdfriday/hugoverse
./build-and-push.sh 26.4.2
```

简单高效！🚀
