# Caddy CLI 使用指南

本文档说明如何使用 `hugov caddy` 命令管理 Caddy 服务器，包括自动配置 CouchDB 反向代理。

## 🚀 快速开始

### 开发环境（本地测试）

```bash
# 使用默认配置启动（localhost:8080）
hugov caddy start

# 访问地址：
# - 核心服务: http://localhost:8080
# - CouchDB:  http://cdb.localhost:8080
```

### 生产环境

```bash
# 启动生产服务器（需要 DNS 配置）
hugov caddy start -domain mdfriday.site

# 访问地址：
# - 核心服务: https://mdfriday.site
# - CouchDB:  https://cdb.mdfriday.site
```

---

## 📋 命令详解

### 1. `start` - 启动服务器

启动 Caddy 服务器，自动配置两个反向代理路由：
- `{domain}` → 默认后端服务（Hugo/App）
- `cdb.{domain}` → CouchDB 服务

**参数：**

```bash
hugov caddy start [options]

Options:
  -domain string    核心域名 (默认: localhost)
  -backend string   默认后端服务地址 (默认: 127.0.0.1:1314)
  -couchdb string   CouchDB 服务地址 (默认: 127.0.0.1:5984)
  -admin string     Caddy Admin API 地址 (默认: http://127.0.0.1:2019)
  -config string    配置文件路径 (默认: /tmp/caddy-config.json)
  -pid string       PID 文件路径 (默认: /tmp/caddy.pid)
  -log string       日志文件路径 (默认: /tmp/caddy.log)
```

**示例：**

```bash
# 开发环境 - 使用默认配置
hugov caddy start

# 开发环境 - 自定义端口
hugov caddy start -domain localhost -backend 127.0.0.1:8080 -couchdb 127.0.0.1:5984

# 生产环境 - 使用自定义域名
hugov caddy start -domain mdfriday.site

# 生产环境 - 完整配置
hugov caddy start \
  -domain mdfriday.site \
  -backend 127.0.0.1:1314 \
  -couchdb 127.0.0.1:5984 \
  -config /etc/caddy/config.json \
  -pid /var/run/caddy.pid \
  -log /var/log/caddy.log
```

**输出示例：**

```
🚀 Starting Caddy server in background...
   Admin API: http://127.0.0.1:2019
   Backend: 127.0.0.1:1314
   CouchDB: 127.0.0.1:5984
   Domain: localhost
   Config: /tmp/caddy-config.json
   PID File: /tmp/caddy.pid
   Log File: /tmp/caddy.log
   Mode: Development (HTTP only, no sudo required)
✅ Caddy server started successfully in background
   PID: 12345

📡 Access URLs:
   Core Service:  http://localhost:8080
   CouchDB:       http://cdb.localhost:8080

   Note: Development mode uses port 8080 (no sudo required)

💡 Tips:
   - Use 'hugov caddy status' to check server status
   - Use 'hugov caddy add' to add static sites
   - Use 'hugov caddy stop' to stop the server
   - Use 'hugov caddy export' to export current config
   - Logs: tail -f /tmp/caddy.log
```

---

### 2. `status` - 检查状态

查看 Caddy 服务器运行状态和配置的路由。

```bash
hugov caddy status

# 使用自定义 Admin API 地址
hugov caddy status -admin http://127.0.0.1:2019
```

**输出示例：**

```
🔍 Checking Caddy server status...
✅ Caddy server is running
   Total routes: 2

📋 Configured domains:
   1. localhost
   2. cdb.localhost

   Admin API: http://127.0.0.1:2019
   View full config: curl http://127.0.0.1:2019/config/ | jq
```

---

### 3. `stop` - 停止服务器

优雅停止 Caddy 服务器。

```bash
hugov caddy stop

# 使用自定义 PID 文件
hugov caddy stop -pid /var/run/caddy.pid
```

---

### 4. `add` - 添加静态站点

动态添加新的静态站点路由。

```bash
hugov caddy add -domain example.com -path /web/sites/example-com
```

**参数：**
- `-domain`: 域名（必需）
- `-path`: 静态文件路径（必需）
- `-admin`: Admin API 地址（可选）

**示例：**

```bash
# 添加静态站点
hugov caddy add -domain blog.example.com -path /var/www/blog

# 添加本地测试站点
hugov caddy add -domain test.localhost -path ./dist
```

---

### 5. `remove` - 移除静态站点

移除已添加的静态站点路由。

```bash
hugov caddy remove -domain example.com
```

---

### 6. `cert` - 查看证书状态

查询域名的 SSL 证书状态（仅生产环境）。

```bash
hugov caddy cert -domain mdfriday.site
hugov caddy cert -domain cdb.mdfriday.site
```

**输出示例：**

```
📜 Checking SSL certificate for: mdfriday.site

✅ Status: Issued

📋 Certificate details:
   Domain: mdfriday.site
   Issuer: Let's Encrypt
   Valid from: 2025-01-04 10:00:00
   Valid until: 2025-04-04 10:00:00
   Days remaining: 89
```

---

### 7. `export` - 导出配置

导出当前 Caddy 配置到文件。

```bash
hugov caddy export -output /tmp/caddy-backup.json
```

**用途：**
- 备份当前配置
- 调试配置问题
- 迁移到其他服务器

---

## 🌐 DNS 配置要求

### 生产环境 DNS 设置

如果使用自定义域名（如 `mdfriday.site`），需要配置以下 DNS 记录：

```
# 方案一：分别配置
A    mdfriday.site        → 服务器IP
A    cdb.mdfriday.site    → 服务器IP

# 方案二：使用通配符（推荐）
A    *.mdfriday.site      → 服务器IP
```

**验证 DNS 配置：**

```bash
# 检查主域名
dig mdfriday.site +short

# 检查 CouchDB 子域名
dig cdb.mdfriday.site +short
```

---

## 🔧 开发环境 vs 生产环境

| 特性            | 开发环境 (localhost)    | 生产环境 (自定义域名)           |
| ------------- | -------------------- | ----------------------- |
| 监听端口          | `:8080`              | `:80`, `:443`           |
| HTTPS         | ❌ 禁用                | ✅ 自动启用                 |
| SSL 证书        | ❌ 不需要               | ✅ Let's Encrypt 自动申请   |
| 需要 sudo       | ❌ 不需要               | ✅ 需要（绑定 80/443 端口）     |
| 访问核心服务        | `http://localhost:8080` | `https://mdfriday.site` |
| 访问 CouchDB    | `http://cdb.localhost:8080` | `https://cdb.mdfriday.site` |

---

## 🐛 常见问题

### Q1: 启动失败 "permission denied"

**原因**：生产环境需要 root 权限绑定 80/443 端口。

**解决方案：**

```bash
# 方案一：使用 sudo（推荐）
sudo hugov caddy start -domain mdfriday.site

# 方案二：给 caddy 二进制添加 CAP_NET_BIND_SERVICE 能力
sudo setcap 'cap_net_bind_service=+ep' $(which caddy)
hugov caddy start -domain mdfriday.site
```

### Q2: 无法访问 CouchDB 子域名

**检查清单：**

1. DNS 是否正确配置
   ```bash
   dig cdb.mdfriday.site +short
   ```

2. Caddy 是否运行
   ```bash
   hugov caddy status
   ```

3. CouchDB 是否运行
   ```bash
   curl http://127.0.0.1:5984
   ```

4. 防火墙是否开放端口
   ```bash
   # 检查 80/443 端口
   sudo netstat -tlnp | grep -E ':(80|443)'
   ```

### Q3: SSL 证书申请失败

**常见原因：**
- 域名 DNS 未正确解析到服务器
- 服务器防火墙阻止了 80/443 端口
- Let's Encrypt 达到速率限制

**检查方法：**

```bash
# 1. 检查 DNS 解析
dig mdfriday.site +short

# 2. 检查端口是否可访问（从外部）
curl -I http://mdfriday.site

# 3. 查看 Caddy 日志
tail -f /tmp/caddy.log

# 4. 查看证书状态
hugov caddy cert -domain mdfriday.site
```

---

## 📚 相关文档

- [Caddy Documentation](https://caddyserver.com/docs/)
- [CouchDB Documentation](https://docs.couchdb.org/)
- [Let's Encrypt Rate Limits](https://letsencrypt.org/docs/rate-limits/)

---

## 🎯 实战示例

### 完整部署流程

```bash
# 1. 启动 CouchDB
docker run -d -p 5984:5984 --name couchdb \
  -e COUCHDB_USER=admin \
  -e COUCHDB_PASSWORD=password \
  couchdb:latest

# 2. 启动你的应用服务器（Hugo/App）
./your-app-server &  # 监听 127.0.0.1:1314

# 3. 启动 Caddy（自动配置反向代理）
hugov caddy start -domain mdfriday.site

# 4. 验证服务
curl https://mdfriday.site
curl https://cdb.mdfriday.site

# 5. 查看状态
hugov caddy status

# 6. 查看证书
hugov caddy cert -domain mdfriday.site
hugov caddy cert -domain cdb.mdfriday.site
```

### 添加额外的静态站点

```bash
# 构建静态站点
hugo build -o /web/sites/blog

# 添加到 Caddy
hugov caddy add -domain blog.mdfriday.site -path /web/sites/blog

# 验证
curl https://blog.mdfriday.site
```

---

## ✅ 最佳实践

1. **开发环境**：使用 `localhost`，避免配置 DNS 和 SSL
2. **生产环境**：配置好 DNS 后再启动 Caddy，让 SSL 证书自动申请
3. **日志监控**：定期查看 Caddy 日志 `tail -f /tmp/caddy.log`
4. **备份配置**：定期导出配置 `hugov caddy export -output backup.json`
5. **证书监控**：检查证书到期时间 `hugov caddy cert -domain xxx`

---

**更新日期：** 2025-01-04
**版本：** v1.0

