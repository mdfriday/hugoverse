# Caddy 配置持久化方案（最终版）

## 问题重新分析

### 关键认识

之前的分析忽略了一个**关键场景**：

#### 运行时动态配置

用户在系统运行期间会通过 API 动态添加配置：

1. **自定义二级域名**
   - 用户在管理后台添加：`user123.example.com`
   - 通过 API → Caddy Admin API 添加路由
   - 配置保存到 `autosave.json`

2. **自定义域名**
   - 用户绑定：`custom-domain.com`
   - 通过 API → Caddy Admin API 添加路由 + TLS
   - 配置保存到 `autosave.json`

3. **其他动态路由**
   - 发布站点、预览站点等
   - 都通过 API 动态添加

**关键问题：如果每次重启都重新初始化 Caddy，会丢失所有运行时动态添加的配置！**

### 为什么不能重新初始化 Caddy？

```
用户操作：
  ├─> 添加自定义域名 custom.com
  ├─> 添加二级域名 sub.example.com
  └─> 发布多个站点

重启容器（如果重新初始化）：
  └─> ❌ 所有自定义域名丢失
  └─> ❌ 所有二级域名丢失
  └─> ❌ 所有发布站点路由丢失
  └─> 用户需要重新配置所有内容
```

## 正确的解决方案

### 方案：启用 Caddy 配置持久化

修改 `docker/caddy/Caddyfile.initial`：

```caddyfile
{
    admin 0.0.0.0:2019
    auto_https off
    
    # 启用配置持久化 ← 关键修改
    persist_config on
    
    log {
        output stdout
        format console
        level INFO
    }
}

# 默认路由（首次启动时使用）
# 如果 autosave.json 存在，会自动覆盖此配置
:80 {
    respond "🚀 Hugoverse is initializing... Please wait." 503
}
```

### Caddy 配置加载逻辑

启用 `persist_config on` 后：

1. **首次启动**：
   ```
   Caddy 启动
     ↓
   加载 Caddyfile.initial (503 响应)
     ↓
   Hugoverse 通过 Admin API 配置路由
     ↓
   Caddy 自动保存到 autosave.json
   ```

2. **重启**：
   ```
   Caddy 启动
     ↓
   检测到 autosave.json 存在
     ↓
   自动加载 autosave.json（包含所有动态添加的配置）
     ↓
   ✅ 所有路由恢复（包括自定义域名、二级域名等）
   ```

3. **运行时**：
   ```
   用户添加自定义域名
     ↓
   Hugoverse → Caddy Admin API
     ↓
   Caddy 自动更新 autosave.json
     ↓
   下次重启时自动恢复
   ```

## 配置文件位置

```
data/
└── caddy/
    ├── config/
    │   └── caddy/
    │       └── autosave.json  ← Caddy 配置持久化文件
    └── data/
        └── caddy/
            ├── certificates/  ← TLS 证书
            └── locks/
```

### autosave.json 包含的配置

```json
{
  "apps": {
    "http": {
      "servers": {
        "main": {
          "routes": [
            // 初始路由（Caddy + CouchDB + 企业站点）
            {"host": ["cdb.localhost"], ...},
            {"host": ["app.localhost"], ...},
            {"host": ["localhost"], ...},
            
            // 运行时动态添加的路由
            {"host": ["user123.example.com"], ...},     // 二级域名
            {"host": ["custom-domain.com"], ...},       // 自定义域名
            {"host": ["site1.example.com"], ...},       // 发布站点
            // ... 更多动态路由
          ]
        }
      }
    },
    "tls": {
      // TLS 配置（包括自定义域名的证书策略）
    }
  }
}
```

## 修改说明

### 1. Caddyfile.initial

**修改前：**
```caddyfile
persist_config off  # 不持久化
```

**修改后：**
```caddyfile
persist_config on   # 启用持久化
```

### 2. Auto-init 逻辑

**无需修改重启逻辑**，因为 Caddy 会自动处理：

```go
if hasLicense {
    // 跳过企业功能配置（包括 Caddy 初始化）
    // Caddy 会自动从 autosave.json 加载配置
    log.Println("   Skipping enterprise features configuration on restart")
    log.Println("   ℹ️  Caddy will auto-load saved configuration (including custom domains)")
}
```

## 工作流程

### 首次启动流程

```
1. Docker 启动 Caddy
   └─> 读取 Caddyfile.initial
   └─> 显示 503 (等待初始化)
   └─> persist_config on 生效

2. Hugoverse 启动
   └─> AutoInitialize
   └─> 检测：系统未初始化
   └─> 创建管理员

3. 企业功能配置
   └─> initializeCaddyRoutes
       └─> 通过 Admin API 配置路由
           ├─> app.localhost → hugoverse:1314
           ├─> cdb.localhost → couchdb:5984
           └─> localhost → /srv/enterprise
   
4. Caddy 自动保存
   └─> 写入 autosave.json
   └─> 包含所有路由配置

5. 用户添加自定义域名
   └─> 管理后台 → API
   └─> Hugoverse → Caddy Admin API
   └─> Caddy 自动更新 autosave.json
```

### 重启流程

```
1. Docker 重启 Caddy
   └─> 检测到 autosave.json 存在
   └─> 自动加载 autosave.json
   └─> ✅ 所有路由立即可用（包括自定义域名）

2. Hugoverse 重启
   └─> AutoInitialize
   └─> 检测：系统已初始化 + 有 License
   └─> 跳过企业功能配置
   └─> ✅ 不干扰 Caddy（Caddy 已自动恢复）

3. 应用正常运行
   └─> 所有路由正常工作
   └─> 自定义域名正常访问
   └─> 二级域名正常访问
```

## 优势

### 1. 保留动态配置

- ✅ 自定义域名在重启后自动恢复
- ✅ 二级域名在重启后自动恢复
- ✅ 所有发布站点路由自动恢复
- ✅ TLS 证书配置自动恢复

### 2. 配置一致性

- ✅ 运行时配置 = 重启后配置
- ✅ 无需手动同步
- ✅ 无需担心配置丢失

### 3. 用户体验

- ✅ 重启后无需重新配置域名
- ✅ 服务立即可用
- ✅ 零停机时间（配置层面）

### 4. 开发体验

- ✅ 无需额外的配置恢复逻辑
- ✅ Caddy 自动处理持久化
- ✅ 代码更简洁

## 测试验证

### 1. 首次启动测试

```bash
docker-compose --env-file .env.local down -v
rm -rf ./data/caddy ./data/hugoverse
docker-compose --env-file .env.local up -d
docker-compose --env-file .env.local logs -f
```

**验证：**
- ✅ 可以访问 `http://app.localhost:8080/admin`
- ✅ `autosave.json` 已创建
- ✅ 包含初始路由配置

### 2. 添加自定义域名测试

```bash
# 在管理后台添加自定义二级域名
# 例如：test.example.com

# 查看 autosave.json
cat ./data/caddy/config/caddy/autosave.json | jq '.apps.http.servers.main.routes[] | select(.match[0].host[] | contains("test"))'
```

**验证：**
- ✅ 新路由已添加到 `autosave.json`
- ✅ 可以访问自定义域名

### 3. 重启后恢复测试（关键）

```bash
# 重启容器
docker-compose --env-file .env.local restart

# 立即测试
curl -i http://app.localhost:8080/admin
curl -i http://test.example.com  # 之前添加的自定义域名
```

**预期结果：**
- ✅ `app.localhost` 立即可用（无 503）
- ✅ 自定义域名 `test.example.com` 立即可用
- ✅ 无需等待 Hugoverse 初始化
- ✅ 所有路由配置完整恢复

### 4. 日志验证

```bash
docker-compose --env-file .env.local logs hugoverse | grep -A5 "already initialized"
```

**预期日志：**
```
✅ System already initialized
ℹ️  Enterprise features already configured (found 1 license(s))
   Skipping enterprise features configuration on restart
   ℹ️  Caddy will auto-load saved configuration (including custom domains)
   💡 To force reconfigure: set FORCE_RECONFIGURE_ENTERPRISE=true
```

## 与之前方案的对比

| 特性 | persist_config off<br>（重新初始化） | persist_config on<br>（持久化）✅ |
|------|--------------------------------------|----------------------------------|
| **自定义域名** | ❌ 重启后丢失 | ✅ 自动恢复 |
| **二级域名** | ❌ 重启后丢失 | ✅ 自动恢复 |
| **发布站点** | ❌ 重启后丢失 | ✅ 自动恢复 |
| **启动时间** | ⚠️ 需等待重新配置 | ✅ 立即可用 |
| **配置一致性** | ❌ 可能不一致 | ✅ 完全一致 |
| **代码复杂度** | ⚠️ 需重新初始化逻辑 | ✅ 简单，Caddy 自动处理 |
| **用户体验** | ❌ 需重新配置 | ✅ 无感知 |

## 注意事项

### 1. 数据卷配置

确保 `docker-compose.yml` 正确映射 Caddy 数据目录：

```yaml
caddy:
  volumes:
    - ${CADDY_HOST_DATA_DIR:-./data/caddy}/data:/data
    - ${CADDY_HOST_DATA_DIR:-./data/caddy}/config:/config  # ← autosave.json 位置
    - ./docker/caddy/Caddyfile.initial:/etc/caddy/Caddyfile:ro
```

### 2. 配置文件权限

`autosave.json` 由 Caddy 自动管理：
- ✅ 自动创建
- ✅ 自动更新
- ✅ 自动加载
- ⚠️ 不要手动编辑（会被覆盖）

### 3. 备份建议

建议备份 `autosave.json`：
```bash
# 备份
cp ./data/caddy/config/caddy/autosave.json ./backups/caddy-config-$(date +%Y%m%d).json

# 恢复（如果需要）
cp ./backups/caddy-config-20260413.json ./data/caddy/config/caddy/autosave.json
docker-compose restart caddy
```

### 4. 清理重建

如果需要完全重新配置：

```bash
# 停止服务
docker-compose --env-file .env.local down

# 删除 Caddy 配置
rm -f ./data/caddy/config/caddy/autosave.json

# 重新启动
docker-compose --env-file .env.local up -d
```

## 总结

### 问题本质

之前的方案（`persist_config off` + 重新初始化）只考虑了初始配置，忽略了**运行时动态添加的配置**，导致重启后用户的自定义域名等配置丢失。

### 正确方案

启用 `persist_config on`，让 Caddy 自动管理配置持久化：
- ✅ 初始配置通过 Admin API 添加
- ✅ 运行时配置通过 Admin API 添加
- ✅ 所有配置自动保存到 `autosave.json`
- ✅ 重启时自动从 `autosave.json` 加载
- ✅ 配置完整一致，用户无感知

### 修改的文件

1. `docker/caddy/Caddyfile.initial` - 启用 `persist_config on`
2. `internal/application/auto_init.go` - 更新日志说明

### 核心价值

- 🎯 **配置持久化**：运行时配置重启后自动恢复
- 🚀 **快速启动**：Caddy 立即加载配置，无需等待
- 💪 **用户体验**：自定义域名等配置永久保留
- 🔧 **简化代码**：无需额外的配置恢复逻辑
