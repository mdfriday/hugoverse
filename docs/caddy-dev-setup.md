# Caddy 本地开发环境配置指南

## 🎯 问题说明

在本地开发环境使用 `localhost` 时，Caddy 默认会尝试：
1. 绑定 80/443 端口（需要 sudo 权限）
2. 为 `localhost` 申请 SSL 证书（会失败）
3. 安装本地根证书（需要密码）

这些在开发环境中都不是必需的，且会导致错误。

## ✅ 解决方案

### 自动检测开发环境

代码已修改为自动检测开发环境：
- **域名为 `localhost` 或 `127.0.0.1`** → 开发模式
- **其他域名** → 生产模式

### 开发模式特点

1. **HTTP Only**：只使用 HTTP，不尝试 HTTPS
2. **非特权端口**：使用 8080 端口，无需 sudo
3. **无证书申请**：不尝试申请 SSL 证书
4. **快速启动**：无需等待证书签发

### 生产模式特点

1. **自动 HTTPS**：监听 80 和 443 端口
2. **自动证书**：Let's Encrypt 自动签发证书
3. **需要权限**：需要 sudo 启动

## 🚀 开发环境使用

### 1. 启动 Caddy（开发模式）

```bash
cd /Users/sunwei/github/mdfriday/hugoverse
go run main.go caddy start

# 或使用编译后的二进制
./hugov caddy start
```

**输出：**
```
🚀 Starting Caddy server...
   Admin API: http://127.0.0.1:2019
   Backend: 127.0.0.1:1314
   Domain: localhost
   Mode: Development (HTTP only, no sudo required)
✅ Caddy server started successfully
   Access via: http://localhost:8080
   Note: Development mode uses port 8080 (no sudo required)

💡 Tips:
   - Use 'hugov caddy status' to check server status
   - Use 'hugov caddy add' to add static sites
   - Press Ctrl+C to stop the server
```

### 2. 访问测试

```bash
# 测试核心服务（反向代理到 1314 端口）
curl http://localhost:8080

# 或在浏览器打开
open http://localhost:8080
```

### 3. 添加测试站点

```bash
# 创建测试站点
mkdir -p ~/web/testsite
echo "<h1>Hello from Test Site!</h1>" > ~/web/testsite/index.html

# 添加到 Caddy
go run main.go caddy add -domain site.test -path /tmp/caddy-test-site

# 配置 hosts
echo "127.0.0.1 site.test" | sudo tee -a /etc/hosts

# 测试访问（注意使用 8080 端口）
curl http://testsite.local:8080
```

### 4. 查看状态

```bash
go run main.go caddy status
```

**输出：**
```
🔍 Checking Caddy server status...
✅ Caddy server is running
   Total routes: 2

📋 Configured domains:
   1. localhost
   2. testsite.local

   Admin API: http://127.0.0.1:2019
```

## 🔧 配置对比

### 开发环境配置（localhost）

```json
{
  "admin": {
    "listen": "127.0.0.1:2019"
  },
  "apps": {
    "http": {
      "servers": {
        "main": {
          "listen": [":8080"],
          "routes": [
            {
              "@id": "core-localhost",
              "match": [{"host": ["localhost"]}],
              "handle": [{
                "handler": "reverse_proxy",
                "upstreams": [{"dial": "127.0.0.1:1314"}]
              }]
            }
          ]
        }
      }
    }
  }
}
```

**特点：**
- ✅ 只监听 8080 端口
- ✅ 不需要 sudo
- ✅ 不尝试 HTTPS
- ✅ 启动快速

### 生产环境配置（实际域名）

```json
{
  "admin": {
    "listen": "127.0.0.1:2019"
  },
  "apps": {
    "http": {
      "servers": {
        "main": {
          "listen": [":80", ":443"],
          "routes": [
            {
              "@id": "core-mdfriday-site",
              "match": [{"host": ["mdfriday.site"]}],
              "handle": [{
                "handler": "reverse_proxy",
                "upstreams": [{"dial": "127.0.0.1:1314"}]
              }]
            }
          ]
        }
      }
    }
  }
}
```

**特点：**
- ✅ 监听 80 和 443 端口
- ✅ 自动申请 SSL 证书
- ⚠️ 需要 sudo 权限
- ⚠️ 需要正确的 DNS 配置

## 🧪 完整测试流程

```bash
#!/bin/bash

# 1. 启动后端服务（在终端 1）
cd /Users/sunwei/github/mdfriday/hugoverse
go run main.go serve -port 1314

# 2. 启动 Caddy（在终端 2）
go run main.go caddy start
# 输出: Access via: http://localhost:8080

# 3. 测试反向代理（在终端 3）
curl http://localhost:8080
# 应该返回后端 API 的响应

# 4. 创建测试站点
mkdir -p ~/web/mysite
cat > ~/web/mysite/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head><title>My Site</title></head>
<body>
    <h1>Hello from My Site!</h1>
    <p>Served by Caddy on port 8080</p>
</body>
</html>
EOF

# 5. 添加站点
go run main.go caddy add -domain mysite.local -path ~/web/mysite

# 6. 配置 hosts
echo "127.0.0.1 mysite.local" | sudo tee -a /etc/hosts

# 7. 测试访问（使用 8080 端口）
curl http://mysite.local:8080
# 输出: Hello from My Site!

# 8. 在浏览器中打开
open http://localhost:8080
open http://mysite.local:8080

# 9. 查看状态
go run main.go caddy status

# 10. 清理（可选）
go run main.go caddy remove -domain mysite.local
sudo sed -i '' '/mysite.local/d' /etc/hosts
```

## 🐛 故障排查

### 问题 1: 端口 8080 被占用

**错误：**
```
failed to start Caddy: listen tcp :8080: bind: address already in use
```

**解决：**
```bash
# 查看占用进程
lsof -i :8080

# 停止占用进程
kill <PID>

# 或使用其他端口（需修改配置）
```

### 问题 2: Admin API 超时

**错误：**
```
Caddy started but Admin API not responding after 5 retries
```

**原因：**
- Caddy 进程启动失败
- 防火墙阻止了 2019 端口

**解决：**
```bash
# 检查 Caddy 进程
ps aux | grep caddy

# 检查 Admin API
curl http://127.0.0.1:2019/config/

# 查看 Caddy 日志
# 日志会输出到终端
```

### 问题 3: 无法访问 localhost:8080

**检查清单：**

1. **Caddy 是否正常运行？**
```bash
go run main.go caddy status
```

2. **后端服务是否运行？**
```bash
curl http://127.0.0.1:1314
```

3. **路由是否正确？**
```bash
curl http://127.0.0.1:2019/config/apps/http/servers/main/routes | jq
```

4. **端口是否正确？**
开发环境使用 8080，生产环境使用 80/443

### 问题 4: 添加的站点无法访问

**检查：**

1. **域名是否添加到 hosts？**
```bash
cat /etc/hosts | grep mysite.local
```

2. **站点目录是否存在？**
```bash
ls -la ~/web/mysite/index.html
```

3. **是否使用了 8080 端口？**
```bash
curl http://mysite.local:8080  # ✅ 正确
curl http://mysite.local        # ❌ 错误（80 端口）
```

## 💡 开发技巧

### 1. 同时运行多个服务

```bash
# 终端 1: 后端 API
cd /Users/sunwei/github/mdfriday/hugoverse
go run main.go serve -port 1314

# 终端 2: Caddy 反向代理
go run main.go caddy start

# 终端 3: 前端开发服务器（如果有）
cd frontend
npm run dev
```

### 2. 快速重启

```bash
# 停止 Caddy (Ctrl+C)
# 重新启动
go run main.go caddy start
```

### 3. 调试配置

```bash
# 查看完整配置
curl http://127.0.0.1:2019/config/ | jq

# 查看路由
curl http://127.0.0.1:2019/config/apps/http/servers/main/routes | jq

# 手动添加路由
curl -X POST http://127.0.0.1:2019/config/apps/http/servers/main/routes \
  -H "Content-Type: application/json" \
  -d '{
    "@id": "test-site",
    "match": [{"host": ["test.local"]}],
    "handle": [{
      "handler": "file_server",
      "root": "/tmp/test"
    }]
  }'
```

## 🎯 生产环境部署

当准备部署到生产环境时：

```bash
# 使用实际域名
sudo hugov caddy start -domain mdfriday.site -backend 127.0.0.1:1314

# 特点：
# - 监听 80 和 443 端口
# - 自动申请 Let's Encrypt 证书
# - 自动 HTTP -> HTTPS 重定向
# - 需要 sudo 权限
# - 需要正确的 DNS 配置
```

## 📚 参考

- 开发模式：HTTP only，8080 端口，无需 sudo
- 生产模式：HTTPS auto，80/443 端口，需要 sudo
- 所有 `.local` 域名都需要配置 `/etc/hosts`
- Admin API 始终在 127.0.0.1:2019

