# 本地 Docker 测试总结

## 🎯 你现在的状态

✅ Docker 方案已完整实现  
❓ 本地还未安装 Docker  
🎯 目标：验证 Docker 部署方案

---

## 🚀 推荐流程（3 步完成）

### Step 1: 安装 Docker（5 分钟）

**推荐：使用 Colima（轻量级，纯命令行）**

```bash
# macOS - 安装 Colima
brew install colima docker docker-compose

# 启动 Colima
colima start --cpu 2 --memory 4

# 验证
docker --version
docker compose version
colima status
```

**或使用 Docker Desktop（不推荐）**

```bash
# macOS
brew install --cask docker
open -a Docker
# 等待 Docker Desktop 启动（约 30-60 秒）
```

**Colima vs Docker Desktop**:
- ✅ Colima: 轻量（~200MB）、快速（~5秒启动）、纯CLI
- ⚠️  Docker Desktop: 重量级（~2GB）、慢（~45秒启动）、有GUI

详细说明：[DOCKER-CLI.md](DOCKER-CLI.md)

**其他系统**：
- Linux: `curl -fsSL https://get.docker.com | sh`
- Windows: https://docs.docker.com/desktop/install/windows-install/

### Step 2: 快速验证（30 秒）

```bash
cd /Users/weisun/github/mdfriday/hugoverse
./verify-local.sh
```

这会检查：
- ✅ Docker 环境
- ✅ Go 编译
- ✅ Docker 配置
- ✅ 必要文件

### Step 3: 完整测试（5 分钟）

```bash
./test-docker-local.sh
```

这会自动：
1. 创建本地测试配置
2. 构建 Docker 镜像
3. 启动所有服务
4. 健康检查
5. 显示 License 信息

**测试完成后访问**：http://localhost:8080/admin

---

## 📋 可用的工具脚本

| 脚本 | 用途 | 耗时 |
|------|------|------|
| `./verify-local.sh` | 快速验证环境（不启动服务） | 30秒 |
| `./test-docker-local.sh` | 完整测试（启动服务） | 5分钟 |
| `./install.sh` | 生产环境安装（交互式） | 5分钟 |

## 🔧 Makefile 命令

```bash
make help           # 查看所有命令
make verify-local   # 快速验证
make test-local     # 完整测试
make up-local       # 启动服务
make logs-local     # 查看日志
make down-local     # 停止服务
make clean-local    # 完全清理
```

---

## 📖 文档索引

| 文档 | 内容 |
|------|------|
| [DOCKER-CLI.md](DOCKER-CLI.md) | **Docker 命令行配置**（Colima vs Docker Desktop） |
| [TEST-LOCAL.md](TEST-LOCAL.md) | 本地测试快速开始 |
| [LOCAL-DEVELOPMENT.md](LOCAL-DEVELOPMENT.md) | 完整本地开发指南 |
| [DOCKER.md](DOCKER.md) | Docker 部署详细文档 |
| [QUICKSTART.md](QUICKSTART.md) | 用户快速开始指南 |
| [README.md](README.md) | 完整项目文档 |
| [IMPLEMENTATION.md](IMPLEMENTATION.md) | 技术实现总结 |

---

## 🎯 当前可以做的事情

### 1️⃣ 如果还没安装 Docker

**推荐：使用 Colima（纯命令行）**

```bash
# 安装 Colima
brew install colima docker docker-compose

# 启动 Colima
colima start --cpu 2 --memory 4

# 验证
colima status
docker ps

# 开始测试
./verify-local.sh
```

**或使用 Docker Desktop**

```bash
# macOS 用户
brew install --cask docker
open -a Docker

# 等待 Docker 启动后
./verify-local.sh
```

详细对比和配置：[DOCKER-CLI.md](DOCKER-CLI.md)

### 2️⃣ 如果已经安装了 Docker

```bash
# 直接运行完整测试
./test-docker-local.sh

# 测试完成后访问
open http://localhost:8080/admin
```

### 3️⃣ 如果只想验证配置

```bash
# 不启动服务，只检查环境
./verify-local.sh
```

### 4️⃣ 开发模式（修改代码）

```bash
# 启动服务
make up-local

# 修改代码后重新构建
docker-compose --env-file .env.local build hugoverse
docker-compose --env-file .env.local restart hugoverse

# 查看日志
make logs-local
```

### 5️⃣ 构建生产镜像

```bash
# 交互式输入版本号
make release

# 或手动指定
VERSION=2.0.0
docker build -t mdfriday/hugoverse:$VERSION -f docker/hugoverse/Dockerfile .
docker push mdfriday/hugoverse:$VERSION
```

---

## 🔍 测试检查项

完整测试会验证：

- ✅ Docker 环境正常
- ✅ Go 代码编译成功
- ✅ Caddy 镜像构建（含 DNSPod 插件）
- ✅ Hugoverse 镜像构建（闭源保护）
- ✅ docker-compose.yml 配置正确
- ✅ 所有服务启动成功
- ✅ 健康检查通过
- ✅ API 端点可访问
- ✅ 自动初始化完成
- ✅ License 自动生成

---

## 🐛 遇到问题？

### 快速诊断

```bash
# 检查 Docker
docker info

# 检查服务状态
docker-compose --env-file .env.local ps

# 查看日志
docker-compose --env-file .env.local logs hugoverse

# 重启服务
docker-compose --env-file .env.local restart
```

### 常见错误

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| `Docker not found` | Docker 未安装 | `brew install --cask docker` |
| `Docker daemon not running` | Docker 未启动 | 启动 Docker Desktop |
| `Port already allocated` | 端口被占用 | 修改 `.env.local` 中的端口 |
| `Health check failed` | 服务未就绪 | 等待更长时间或查看日志 |

---

## 📞 获取帮助

1. **查看详细日志**：
   ```bash
   docker-compose --env-file .env.local logs -f hugoverse
   ```

2. **进入容器调试**：
   ```bash
   docker-compose --env-file .env.local exec hugoverse sh
   ```

3. **查看文档**：
   - 本地测试：`cat TEST-LOCAL.md`
   - 开发指南：`cat LOCAL-DEVELOPMENT.md`
   - Docker 文档：`cat DOCKER.md`

4. **提交 Issue**：
   https://github.com/mdfriday/hugoverse/issues

---

## ✅ 验证完成后

测试成功后，你可以：

1. ✅ **提交代码**
   ```bash
   git add .
   git commit -m "feat: add Docker deployment with Master License system"
   git push
   ```

2. ✅ **构建生产镜像**
   ```bash
   make release
   ```

3. ✅ **部署到生产环境**
   ```bash
   # 在生产服务器上
   bash install.sh
   ```

4. ✅ **编写部署文档**
   - 更新 README.md（如需要）
   - 创建部署说明
   - 记录配置细节

---

## 🎉 开始测试

**现在就开始！**

```bash
# 如果还没安装 Docker
brew install --cask docker
open -a Docker

# Docker 启动后，运行测试
cd /Users/weisun/github/mdfriday/hugoverse
./test-docker-local.sh
```

测试成功后访问：**http://localhost:8080/admin**

登录信息：
- Email: `admin@localhost`
- Password: `test123456`

---

**祝测试顺利！** 🚀

有任何问题随时查看 [TEST-LOCAL.md](TEST-LOCAL.md) 或 [LOCAL-DEVELOPMENT.md](LOCAL-DEVELOPMENT.md)
