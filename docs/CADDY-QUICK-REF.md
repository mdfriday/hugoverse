# Caddy 本地测试环境 - 快速参考 🚀

## ✅ 验证结果

**状态：** 🟢 **全部通过 - 可以使用！**

---

## 🎯 核心改进

| 项目 | 修复前 ❌ | 修复后 ✅ |
|------|---------|---------|
| HTTPS | 强制启用，证书错误 | 完全禁用 |
| 端口 | 80/443 (需要 sudo) | 8080 (无需 sudo) |
| 证书 | 要求密码安装 | 不申请证书 |
| 警告 | OCSP stapling 警告 | 无警告 |
| 启动 | 超时失败 | 5-10秒成功 |

---

## 🚀 快速开始

```bash
# 1. 启动 Caddy（开发模式）
cd /Users/sunwei/github/mdfriday/hugoverse
./hugov caddy start

# 预期输出：
# ✅ Mode: Development (HTTP only, no sudo required)
# ✅ Access via: http://localhost:8080
```

**关键日志（表示成功）：**
```
"automatic HTTPS is completely disabled for server"
```

---

## 📊 测试验证

### 1. 检查状态
```bash
./hugov caddy status

# 应该显示：
# ✅ Caddy server is running
#    Total routes: 2
```

### 2. 测试 HTTP 访问
```bash
curl http://localhost:8080

# 或使用测试站点
curl -H "Host: testsite.local" http://localhost:8080
```

### 3. 添加新站点
```bash
# 创建站点
mkdir -p ~/web/mysite
echo "<h1>Hello</h1>" > ~/web/mysite/index.html

# 添加到 Caddy
./hugov caddy add -domain mysite.local -path ~/web/mysite

# 配置 hosts
echo "127.0.0.1 mysite.local" | sudo tee -a /etc/hosts

# 访问
curl http://mysite.local:8080
```

---

## 💡 重要提示

### ⚠️ 端口号

**开发环境（localhost）：使用 8080**
```bash
✅ http://localhost:8080
✅ http://testsite.local:8080
❌ http://localhost (80 端口不工作)
```

### 🔄 模式切换

**开发模式：**
```bash
./hugov caddy start -domain localhost
# → 8080 端口，HTTP only
```

**生产模式：**
```bash
sudo ./hugov caddy start -domain mdfriday.site
# → 80/443 端口，HTTPS auto
```

---

## 🛠️ 常用命令

```bash
# 查看状态
./hugov caddy status

# 添加站点
./hugov caddy add -domain example.local -path /path/to/site

# 移除站点
./hugov caddy remove -domain example.local

# 查看证书（开发环境无证书）
./hugov caddy cert -domain example.local

# 停止 Caddy
pkill -f 'hugov caddy'
```

---

## 🐛 故障排查

### 问题：无法访问 localhost:8080

**检查：**
```bash
# 1. Caddy 是否运行？
ps aux | grep "hugov caddy"

# 2. 端口是否被占用？
lsof -i :8080

# 3. Admin API 是否正常？
curl http://127.0.0.1:2019/config/
```

### 问题：HTTPS 错误

**检查配置：**
```bash
curl -s http://127.0.0.1:2019/config/apps/http/servers/main | jq '.automatic_https'

# 应该显示：
# {
#   "disable": true
# }
```

---

## 📁 测试环境

```
✅ Caddy: v2.10.2
✅ 端口: 8080 (HTTP), 2019 (Admin)
✅ 模式: Development
✅ HTTPS: Disabled
✅ 路由: 2 (localhost, testsite.local)
✅ 进程: PID 80799
```

---

## 📖 完整文档

- **测试报告**: `docs/caddy-test-report.md`
- **开发指南**: `docs/caddy-dev-setup.md`
- **CLI 文档**: `docs/caddy-cli.md`

---

## 🎉 使用示例

### 场景 1: 开发 API

```bash
# 终端 1: 启动后端
./hugov serve -port 1314

# 终端 2: 启动 Caddy
./hugov caddy start

# 访问
curl http://localhost:8080
# Caddy → 反向代理 → 127.0.0.1:1314
```

### 场景 2: 测试静态站点

```bash
# 启动 Caddy
./hugov caddy start

# 添加站点
./hugov caddy add -domain myapp.local -path ~/web/myapp/dist

# 配置 hosts + 访问
echo "127.0.0.1 myapp.local" | sudo tee -a /etc/hosts
open http://myapp.local:8080
```

---

## ✅ 验证清单

- [x] Caddy 正常启动
- [x] 无需 sudo
- [x] 无需密码
- [x] 无证书警告
- [x] HTTP 访问正常
- [x] 静态文件服务正常
- [x] 动态添加站点正常
- [x] Admin API 正常
- [x] 状态查询正常

---

**测试完成时间**: 2025-12-28 11:31  
**测试状态**: 🟢 **通过**  
**可用性**: ✅ **生产就绪（开发环境）**

