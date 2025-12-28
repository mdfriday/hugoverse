# Caddy 命令行工具使用指南

## 📋 命令概览

`hugov caddy` 命令提供了完整的 Caddy 服务器管理功能。

```bash
hugov caddy [subcommand]
```

### 子命令列表

| 命令 | 功能 | 示例 |
|------|------|------|
| `start` | 启动 Caddy 服务器 | `hugov caddy start` |
| `stop` | 停止 Caddy 服务器 | `hugov caddy stop` |
| `status` | 查看服务器状态 | `hugov caddy status` |
| `add` | 添加静态站点 | `hugov caddy add -domain example.com -path /path` |
| `remove` | 移除静态站点 | `hugov caddy remove -domain example.com` |
| `cert` | 查看 SSL 证书状态 | `hugov caddy cert -domain example.com` |

## 🚀 使用示例

### 1. 启动 Caddy 服务器

```bash
# 使用默认配置启动（开发环境，使用 localhost）
hugov caddy start

# 自定义配置启动
hugov caddy start -domain localhost -backend 127.0.0.1:1314

# 生产环境（使用实际域名）
hugov caddy start -domain mdfriday.site -backend 127.0.0.1:1314
```

**参数说明：**
- `-admin`: Admin API 地址（默认：`http://127.0.0.1:2019`）
- `-backend`: 后端服务地址（默认：`127.0.0.1:1314`）
- `-domain`: 核心域名（默认：`localhost`，开发环境推荐）
- `-config`: 配置文件路径（可选）

**启动后输出：**
```
🚀 Starting Caddy server...
   Admin API: http://127.0.0.1:2019
   Backend: 127.0.0.1:1314
   Domain: localhost
✅ Caddy server started successfully
   Access via: http://localhost

💡 Tips:
   - Use 'hugov caddy status' to check server status
   - Use 'hugov caddy add' to add static sites
   - Press Ctrl+C to stop the server
```

**注意事项：**
- 开发环境使用 `localhost`，不会尝试申请 SSL 证书
- 生产环境使用实际域名（如 `mdfriday.site`），会自动申请 Let's Encrypt 证书
- 绑定 80/443 端口需要 root 权限：`sudo hugov caddy start`

### 2. 查看服务器状态

```bash
hugov caddy status
```

**输出示例：**
```
🔍 Checking Caddy server status...
✅ Caddy server is running
   Total routes: 1

📋 Configured domains:
   1. localhost

   Admin API: http://127.0.0.1:2019
   View full config: curl http://127.0.0.1:2019/config/ | jq
```

### 3. 添加静态站点

```bash
# 添加本地测试站点
hugov caddy add -domain mysite.local -path /Users/sunwei/web/mysite

# 添加生产站点（会自动申请 SSL）
hugov caddy add -domain blog.example.com -path /web/sites/blog
```

**参数说明：**
- `-domain`: 域名（必需）
- `-path`: 静态文件目录（必需）
- `-admin`: Admin API 地址（可选）

**输出示例：**
```
📁 Adding static site: mysite.local -> /Users/sunwei/web/mysite
✅ Static site added successfully
   Domain: mysite.local
   Path: /Users/sunwei/web/mysite
```

**使用技巧：**
- 开发环境可以使用 `.local` 域名，配合 `/etc/hosts`
- 确保站点目录存在且有 `index.html`
- 生产环境确保 DNS 已正确配置

### 4. 查看 SSL 证书状态

```bash
hugov caddy cert -domain example.com
```

**输出示例（证书已签发）：**
```
📜 Checking SSL certificate for: example.com

✅ Status: Issued

📋 Certificate details:
   Domain: example.com
   Issuer: Let's Encrypt
   Valid from: 2024-01-01 00:00:00
   Valid until: 2024-04-01 00:00:00
   Days remaining: 89
```

**输出示例（证书申请中）：**
```
📜 Checking SSL certificate for: example.com

⏳ Status: Pending (being issued)

💡 Certificate is being issued. This usually takes a few seconds.
   Check again in a moment.
```

### 5. 移除静态站点

```bash
hugov caddy remove -domain mysite.local
```

**输出示例：**
```
🗑️  Removing static site: mysite.local
✅ Static site removed successfully
```

### 6. 停止 Caddy 服务器

在运行 `hugov caddy start` 的终端按 `Ctrl+C`，服务器会优雅关闭。

```
^C

🛑 Stopping Caddy server...
✅ Caddy server stopped
```

## 📖 完整工作流示例

### 开发环境完整流程

```bash
# 1. 启动 Caddy（使用 localhost）
hugov caddy start -domain localhost &

# 2. 检查状态
hugov caddy status

# 3. 创建测试站点目录
mkdir -p ~/web/testsite
echo "<h1>Hello World</h1>" > ~/web/testsite/index.html

# 4. 添加测试站点（使用 .local 域名）
hugov caddy add -domain testsite.local -path ~/web/testsite

# 5. 配置 hosts（macOS/Linux）
echo "127.0.0.1 testsite.local" | sudo tee -a /etc/hosts

# 6. 测试访问
curl http://testsite.local
# 输出: <h1>Hello World</h1>

# 7. 查看所有路由
hugov caddy status

# 8. 移除站点
hugov caddy remove -domain testsite.local

# 9. 停止服务器（在启动的终端按 Ctrl+C）
```

### 生产环境完整流程

```bash
# 1. 确保域名 DNS 已配置指向服务器

# 2. 启动 Caddy（需要 root 权限）
sudo hugov caddy start -domain mdfriday.site -backend 127.0.0.1:1314

# 3. 添加用户站点
sudo hugov caddy add \
  -domain blog.example.com \
  -path /web/sites/user123/blog

# 4. 等待证书签发（通常几秒钟）
sleep 10

# 5. 检查证书状态
hugov caddy cert -domain blog.example.com

# 6. 访问站点
curl https://blog.example.com
```

## 🔧 开发环境配置

### 本地 hosts 配置

编辑 `/etc/hosts`（需要 root 权限）：

```bash
sudo vim /etc/hosts
```

添加：
```
127.0.0.1 testsite.local
127.0.0.1 blog.local
127.0.0.1 docs.local
```

### 测试站点结构

```bash
mkdir -p ~/web/sites/testsite
cat > ~/web/sites/testsite/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Test Site</title>
</head>
<body>
    <h1>Hello from Caddy!</h1>
    <p>This is a test site served by Caddy.</p>
</body>
</html>
EOF
```

## 🐛 故障排查

### 问题 1: 端口被占用

**错误：**
```
❌ Failed to start Caddy: listen tcp :80: bind: address already in use
```

**解决：**
```bash
# 查看占用 80 端口的进程
sudo lsof -i :80

# 停止其他 web 服务器
sudo systemctl stop apache2
sudo systemctl stop nginx
```

### 问题 2: 权限不足

**错误：**
```
❌ Failed to start Caddy: listen tcp :80: bind: permission denied
```

**解决：**
```bash
# 使用 sudo 启动
sudo hugov caddy start
```

### 问题 3: 证书申请失败

**原因：**
- DNS 未正确配置
- 域名无法从公网访问
- 防火墙阻止了 80/443 端口

**检查：**
```bash
# 检查 DNS 解析
dig example.com

# 检查端口是否开放
sudo netstat -tulpn | grep :80

# 查看 Caddy 日志
curl http://127.0.0.1:2019/config/apps/tls/automation/policies
```

### 问题 4: 站点 404

**检查清单：**
1. 站点目录是否存在
2. 目录是否有 `index.html`
3. 文件权限是否正确
4. 域名是否已添加到 Caddy

```bash
# 检查站点目录
ls -la /path/to/site

# 查看 Caddy 路由
hugov caddy status
```

## 💡 高级用法

### 与后端服务集成

```bash
# 1. 启动后端 API 服务
hugov serve -port 1314 &

# 2. 启动 Caddy，代理到后端
hugov caddy start -domain localhost -backend 127.0.0.1:1314

# 3. 添加静态前端站点
hugov caddy add -domain app.local -path /web/frontend/dist
```

### 批量添加站点

```bash
#!/bin/bash
sites=(
  "site1.local:/web/sites/site1"
  "site2.local:/web/sites/site2"
  "site3.local:/web/sites/site3"
)

for site in "${sites[@]}"; do
  domain="${site%%:*}"
  path="${site##*:}"
  hugov caddy add -domain "$domain" -path "$path"
done
```

### 监控证书过期

```bash
#!/bin/bash
# check-certs.sh
domains=("example.com" "blog.example.com" "docs.example.com")

for domain in "${domains[@]}"; do
  echo "Checking $domain..."
  hugov caddy cert -domain "$domain"
  echo ""
done
```

## 📚 相关命令

### 直接使用 Caddy Admin API

```bash
# 查看完整配置
curl http://127.0.0.1:2019/config/ | jq

# 查看所有路由
curl http://127.0.0.1:2019/config/apps/http/servers/main/routes | jq

# 查看证书
curl http://127.0.0.1:2019/config/apps/tls/certificates | jq

# 删除路由
curl -X DELETE http://127.0.0.1:2019/id/site-example-com
```

### 查看 Caddy 进程

```bash
# 查看进程
ps aux | grep caddy

# 查看端口占用
sudo lsof -i :80
sudo lsof -i :443
sudo lsof -i :2019

# 停止 Caddy
pkill caddy
```

## 🎯 最佳实践

1. **开发环境**：使用 `localhost` 或 `.local` 域名
2. **生产环境**：使用实际域名，自动启用 HTTPS
3. **权限管理**：生产环境使用 sudo，开发环境可以使用高端口
4. **日志查看**：使用 Admin API 监控服务状态
5. **备份配置**：定期备份 Caddy 配置文件

## 📖 参考资源

- [Caddy 官方文档](https://caddyserver.com/docs/)
- [Caddy Admin API](https://caddyserver.com/docs/api)
- [Let's Encrypt 文档](https://letsencrypt.org/docs/)

