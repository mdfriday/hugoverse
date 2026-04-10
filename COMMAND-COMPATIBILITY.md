# 启动命令兼容性说明

## 问题

手动启动和 Docker 启动使用不同的命令：

```bash
# 手动启动（用户现有方式）
nohup go/bin/hugoverse serve -env prod &

# Docker 启动（新改的）
hugoverse server
```

这会造成混淆和不一致。

## 解决方案：两种命令都支持

### 1. `serve` 命令（现有命令，完全兼容）

**实现：** 复用 `internal/interfaces/cli/serve.go` 中的现有实现

**用法：**
```bash
hugoverse serve -env prod -port 1314 -https
```

**特点：**
- ✅ 支持命令行参数（flag）
- ✅ 向后兼容，现有部署不受影响
- ✅ 适合手动启动
- ✅ 使用现有的 `serverCmd` 实现

**示例：**
```bash
# 使用默认配置（dev 环境，端口 1314）
hugoverse serve

# 生产环境，自定义端口
hugoverse serve -env prod -port 8080

# 启用 HTTPS
hugoverse serve -env prod -https
```

### 2. `server` 命令（新增命令，Docker 优化）

**实现：** `internal/interfaces/cli/server/command.go`

**用法：**
```bash
hugoverse server
```

**特点：**
- ✅ 纯环境变量配置
- ✅ 符合 12-factor app 原则
- ✅ Docker 最佳实践
- ✅ 简洁，无需参数

**配置方式：**
```bash
export HTTP_PORT=1314
export ENV=prod
export ENABLE_HTTPS=false

hugoverse server
```

## 推荐使用场景

| 场景 | 推荐命令 | 原因 |
|------|---------|------|
| 手动部署 | `serve` | 灵活，支持参数 |
| Docker | `server` | 简洁，环境变量配置 |
| systemd | `serve` | 参数在 service 文件中清晰 |
| 开发调试 | `serve` | 快速修改参数 |

## 现在的配置

### Dockerfile

```dockerfile
# 使用 server 命令（环境变量配置）
CMD ["server"]
```

### docker-compose.yml

```yaml
environment:
  - HTTP_PORT=1314
  - ENV=prod
```

### 手动启动（保持不变）

```bash
# 你现有的启动方式，完全兼容
nohup go/bin/hugoverse serve -env prod &
```

## 实现细节

### serve 命令（适配器模式）

```
serve.go (现有实现)
  └─ serverCmd 结构体
       ├─ NewServeCmd() - 解析 flag 参数
       └─ Run() - 启动服务器

serve_adapter.go (新增)
  └─ ServeCommand 结构体
       ├─ Name() = "serve"
       ├─ Description()
       └─ Run() - 调用 NewServeCmd()
```

**优点：**
- ✅ 不修改现有代码
- ✅ 只是添加适配器层
- ✅ 保持向后兼容

### server 命令（全新实现）

```
server/command.go (新增)
  └─ Command 结构体
       ├─ Name() = "server"
       ├─ Description()
       └─ Run() - 从环境变量读取配置
```

## 测试验证

```bash
# 1. 编译
go build -o hugoverse ./cmd/hugoverse

# 2. 查看可用命令
./hugoverse
# 输出:
#   serve   Start the Hugoverse API server (supports -env, -port, -https flags)
#   server  Start the Hugoverse API server
#   sse     Start a web server for testing SSE implementation

# 3. 测试 serve 命令（你的方式）
./hugoverse serve -env prod -port 1314

# 4. 测试 server 命令（Docker 方式）
HTTP_PORT=1314 ENV=prod ./hugoverse server
```

## 已修改的文件

1. ✅ `internal/interfaces/cli/serve_adapter.go` (新建)
   - 适配器，连接旧命令系统和新接口
   - 调用现有的 `NewServeCmd()`

2. ✅ `internal/interfaces/cli/server/command.go` (已存在)
   - 纯环境变量配置
   - Docker 优化

3. ✅ `internal/interfaces/cli/commands.go`
   - 同时注册 `ServeCommand` 和 `server.Command`

4. ✅ `docker/hugoverse/Dockerfile`
   - 使用 `server` 命令

5. ⭕ `internal/interfaces/cli/serve.go` (未修改)
   - 保持原样，完全兼容

## 总结

✅ **向后兼容**: 你现有的 `hugoverse serve -env prod` 完全不变
✅ **Docker 优化**: Docker 使用 `server` + 环境变量
✅ **适配器模式**: 复用现有代码，不重复实现
✅ **无需修改**: 现有部署不需要任何改动

现在重新构建和测试：

```bash
# 重新构建（代码已更新）
docker-compose --env-file .env.local build hugoverse

# 启动
docker-compose --env-file .env.local up -d

# 验证
docker-compose --env-file .env.local logs hugoverse | grep "Starting"
```
