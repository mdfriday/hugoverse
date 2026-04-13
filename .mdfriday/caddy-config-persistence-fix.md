# Caddy 配置持久化修复

## 问题描述

在容器 restart 后，Caddy 丢失了所有通过 Admin API 动态添加的路由配置（如自定义域名、二级域名、发布站点等），只保留了初始的 503 响应。

## 根本原因

Caddy 的启动命令：

```dockerfile
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
```

导致 Caddy **每次启动都从 `Caddyfile.initial` 重新开始**，完全忽略了 `autosave.json` 中保存的动态配置。

虽然 Caddy 会将运行时配置保存到 `autosave.json`，但在重启时不会自动加载它。

## 解决方案

### 1. 智能启动脚本（`docker/caddy/entrypoint.sh`）

创建启动脚本，实现智能配置加载：

```bash
#!/bin/sh
if [ -f "/config/caddy/autosave.json" ]; then
    # 有保存的配置 → 恢复之前的完整配置
    exec caddy run --resume
else
    # 首次启动 → 使用初始 Caddyfile
    exec caddy run --config "/etc/caddy/Caddyfile" --adapter caddyfile
fi
```

### 2. 更新 Dockerfile

使用 `ENTRYPOINT` 替代 `CMD`：

```dockerfile
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
```

### 3. 增强 auto_init.go

添加 Caddy 配置完整性检查，处理边缘情况（数据库存在但 Caddy 配置丢失）：

```go
// 检查 Caddy 配置是否完整
caddyConfigured, err := isCaddyProperlyConfigured(log)
if !caddyConfigured {
    // 重新初始化 Caddy 路由
    go delayedCaddyAndSiteReinitialize(adminApp, log)
}
```

## 工作流程

### 第一次启动

1. Caddy 启动 → 没有 `autosave.json` → 使用 `Caddyfile.initial`（503 响应）
2. Hugoverse 启动 → 检测到未初始化 → 执行 AUTO_INIT
3. 通过 Admin API 配置 Caddy 路由（localhost、app.localhost、cdb.localhost 等）
4. 配置企业站点
5. 生成 License
6. **Caddy 自动保存配置到 `autosave.json`**

### 容器重启（正常情况）

1. Caddy 启动 → 发现 `autosave.json` → **使用 `--resume` 恢复完整配置**
2. Hugoverse 启动 → 检测到已初始化且有 License → 检查 Caddy 配置完整性 → ✅ 完整 → 跳过配置

**结果：所有动态配置的路由（自定义域名等）都被保留**

### 边缘情况：data/caddy 被删除但数据库保留

1. Caddy 启动 → 没有 `autosave.json` → 使用 `Caddyfile.initial`（503 响应）
2. Hugoverse 启动 → 检测到已初始化且有 License → 检查 Caddy 配置 → ❌ 不完整（只有 503）
3. **自动重新初始化 Caddy 路由和企业站点**
4. 配置恢复完成

## 验证

测试完整流程：

```bash
# 1. 完全清理
docker-compose down -v
rm -rf ./data

# 2. 首次启动
docker-compose up -d

# 3. 检查配置（应该有完整的路由）
docker-compose exec caddy wget -qO- http://localhost:2019/config/ | jq '.apps.http.servers.srv0.routes | length'

# 4. 重启容器
docker-compose restart caddy

# 5. 再次检查配置（应该保留所有路由）
docker-compose exec caddy wget -qO- http://localhost:2019/config/ | jq '.apps.http.servers.srv0.routes | length'
```

## 影响范围

- ✅ 自定义域名持久化
- ✅ 自定义二级域名持久化
- ✅ 发布站点持久化
- ✅ 容器重启后配置保持一致
- ✅ 处理 data 目录清理的边缘情况

## 文件修改

1. **新增**: `docker/caddy/entrypoint.sh` - 智能启动脚本
2. **修改**: `docker/caddy/Dockerfile` - 使用 ENTRYPOINT
3. **修改**: `docker/caddy/Caddyfile.initial` - 更新注释
4. **修改**: `internal/application/auto_init.go` - 添加 Caddy 配置完整性检查

## 技术细节

### Caddy 配置持久化机制

- **`autosave.json`**: Caddy 自动保存当前完整配置
- **`--resume` 标志**: 从 `autosave.json` 恢复配置
- **Admin API**: 动态修改配置会自动触发 autosave

### 为什么需要智能启动脚本？

因为：
1. 如果总是使用 `--config`，会忽略 `autosave.json`
2. 如果总是使用 `--resume`，首次启动会失败（没有 autosave.json）
3. 智能脚本根据 `autosave.json` 是否存在选择正确的启动方式

## 参考

- [Caddy Configuration Persistence](https://caddyserver.com/docs/command-line#caddy-run)
- [Caddy Admin API](https://caddyserver.com/docs/api)
