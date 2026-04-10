# Docker 部署指南

Hugoverse 的 Docker 部署方案提供了开箱即用的自动化部署体验。

## 🚀 快速开始

### 一键安装

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/mdfriday/hugoverse/main/install.sh)
```

或手动安装：

```bash
git clone https://github.com/mdfriday/hugoverse.git
cd hugoverse
bash install.sh
```

## 📦 架构说明

Hugoverse Docker 方案包含三个核心服务：

```
┌─────────────────────────────────────────┐
│           Hugoverse 控制面板            │
│  - License 管理                         │
│  - 自动初始化                           │
│  - API 服务                             │
│  - Master License 在线验证              │
└──────────────┬──────────────────────────┘
               │
       ┌───────┴───────┐
       │               │
┌──────▼─────┐  ┌─────▼──────┐
│   Caddy    │  │  CouchDB   │
│ - 反向代理  │  │ - 同步数据  │
│ - 自动SSL   │  │ - 用户数据库│
│ - 静态站点  │  │            │
└────────────┘  └────────────┘
```

### 服务端口

- **Hugoverse**: 1314 (内部)
- **Caddy**: 80 (HTTP), 443 (HTTPS), 2019 (Admin API)
- **CouchDB**: 5984 (内部)

### 数据持久化

所有数据通过 Docker Volumes 持久化：

```
hugoverse_data      -> Hugoverse 应用数据
couchdb_data        -> CouchDB 数据库
couchdb_config      -> CouchDB 配置
caddy_data          -> Caddy 数据（证书）
caddy_config        -> Caddy 配置
sites               -> 发布的站点
backups             -> 备份文件
```

## 🔧 配置说明

### 核心配置项

配置文件：`.env`

```env
# 【必填】域名和服务器
DOMAIN=your-domain.com
SERVER_IP=1.2.3.4

# 【必填】管理员
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=secure_password

# 【必填】CouchDB
COUCHDB_PASSWORD=another_secure_password

# 【可选】DNSPod（泛域名 SSL）
DNSPOD_ENABLED=true
DNSPOD_ID=your_dnspod_id
DNSPOD_SECRET=your_dnspod_secret

# 【可选】Master License（多用户）
MASTER_LICENSE=
```

### 自动初始化

默认情况下，`AUTO_INIT=true` 会在首次启动时：

1. ✅ 使用环境变量创建管理员账户
2. ✅ 配置 CouchDB 连接
3. ✅ 配置 Caddy 反向代理
4. ✅ 生成 1 个企业 License（免费版）
5. ✅ 配置企业站点路由

如果希望手动配置，设置 `AUTO_INIT=false`，然后访问：

```
http://your-domain.com/admin/init
```

## 💎 Master License 系统

### 免费版 vs 付费版

| 版本 | Sub-Licenses | 价格 |
|------|--------------|------|
| 免费版 | 1 | $0 |
| Starter | 10 | $99/年 |
| Pro | 100 | $499/年 |
| Unlimited | ∞ | $2,999 一次性 |

### 使用 Master License

1. 购买 Master License：https://mdfriday.com/pricing

2. 添加到 `.env`：
   ```env
   MASTER_LICENSE=YOUR_LICENSE_KEY
   ```

3. 重启服务：
   ```bash
   docker-compose restart hugoverse
   ```

4. 系统将在线验证 License，并根据配额生成 Sub-Licenses

### 查看配额

```bash
# 查看日志
docker-compose logs hugoverse | grep "License Quota"

# 输出示例：
# Type: pro
# Max Sub-Licenses: 100
# Used: 1
# Remaining: 99
```

## 🔐 在线验证机制

Master License 通过 API 在线验证：

```
Hugoverse --> https://api.mdfriday.com/v1/master-license/verify
          <-- { valid: true, max_sub_licenses: 100, ... }
```

验证失败或网络错误时，自动降级到免费版（1 License）。

生成 License 后，会上报使用情况：

```
Hugoverse --> https://api.mdfriday.com/v1/master-license/report-usage
          <-- { success: true }
```

## 📋 使用指南

### 启动服务

```bash
docker-compose up -d
```

### 查看日志

```bash
# 所有服务
docker-compose logs -f

# 特定服务
docker-compose logs -f hugoverse
docker-compose logs -f caddy
docker-compose logs -f couchdb
```

### 查看生成的 License

```bash
docker-compose logs hugoverse | grep "License Key"

# 输出：
# License Key: MDF-XXXX-XXXX-XXXX
# Email: mdf.XXXX.XXXX@mdfriday.com
# Password: xxxxxxxxxx
```

将这些凭证分享给用户，在 Obsidian Friday 插件中激活。

### 手动生成 License

```bash
# 进入容器
docker-compose exec hugoverse sh

# 生成 License（会验证 Master License 配额）
/app/hugoverse license generate \
  -email admin@your-domain.com \
  -password your_password \
  -plan enterprise \
  -count 5
```

如果超出配额，会看到：

```
❌ License Quota Exceeded
   Current: 100 / 100 licenses
   Requested: 5
   Available: 0

💡 To generate more licenses:
   1. Visit: https://mdfriday.com/pricing
   2. Purchase a Master License
   3. Set environment: export MASTER_LICENSE=YOUR_KEY
```

### 重启服务

```bash
docker-compose restart
```

### 停止服务

```bash
docker-compose down
```

### 更新

```bash
# 拉取最新镜像
docker-compose pull

# 重新启动
docker-compose up -d
```

## 🐛 故障排查

### 健康检查

```bash
curl http://localhost:1314/api/health

# 响应：
# {
#   "status": "healthy",
#   "docker": true,
#   "initialized": true,
#   "version": "latest"
# }
```

### License 配额问题

问题：
```
❌ License Quota Exceeded
```

解决方案：
1. 检查当前 Master License 类型和配额
2. 购买更高级别的 Master License
3. 更新 `.env` 中的 `MASTER_LICENSE`
4. 重启 Hugoverse

### Master License 验证失败

问题：
```
⚠️  Master License verification failed: connection timeout
   Falling back to FREE mode (1 license)
```

解决方案：
1. 检查服务器网络是否可以访问 `api.mdfriday.com`
2. 检查防火墙规则
3. 验证 Master License 是否正确

临时方案：系统会自动降级到免费版（1 License），不影响基本使用。

### CouchDB 连接失败

问题：
```
❌ Timeout waiting for CouchDB
```

解决方案：
```bash
# 检查 CouchDB 状态
docker-compose ps couchdb

# 查看 CouchDB 日志
docker-compose logs couchdb

# 重启 CouchDB
docker-compose restart couchdb
```

### Caddy SSL 证书问题

使用 DNSPod（推荐）：
```env
DNSPOD_ENABLED=true
DNSPOD_ID=your_id
DNSPOD_SECRET=your_secret
```

不使用 DNSPod：
- 确保 80 端口可访问
- 域名正确指向服务器
- 防火墙允许 HTTP 流量

## 🔒 安全建议

1. **使用强密码**：管理员和 CouchDB 密码至少 12 位
2. **启用 DNSPod**：获取泛域名 SSL 证书
3. **定期备份**：自动备份已启用，retention = 7 天
4. **保护 Master License**：不要提交到 Git，使用环境变量
5. **定期更新**：及时更新到最新版本

## 📊 监控和维护

### 查看资源使用

```bash
# Docker 资源
docker stats

# 磁盘使用
docker system df

# 特定服务
docker stats hugoverse-app hugoverse-caddy hugoverse-couchdb
```

### 备份

自动备份：
- 频率：每天
- 位置：`backups` volume
- 保留：7 天（可配置）

手动备份：
```bash
# 备份所有 volumes
docker run --rm \
  -v hugoverse_couchdb_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/backup-$(date +%Y%m%d).tar.gz /data
```

### 恢复

```bash
# 停止服务
docker-compose down

# 恢复 volume
docker run --rm \
  -v hugoverse_couchdb_data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/backup-YYYYMMDD.tar.gz -C /

# 启动服务
docker-compose up -d
```

## 🆘 获取帮助

- **GitHub Issues**: https://github.com/mdfriday/hugoverse/issues
- **Email**: support@mdfriday.com
- **文档**: https://docs.mdfriday.com

## 🎯 下一步

1. ✅ 完成安装
2. ✅ 配置 DNS
3. ✅ 获取生成的 License
4. ✅ 在 Obsidian Friday 插件中激活
5. 🎉 开始使用！

需要更多 License？访问：https://mdfriday.com/pricing
