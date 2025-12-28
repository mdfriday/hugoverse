# Caddy Client 快速参考

## 📦 文件结构

```
internal/infrastructure/caddy/
├── client.go          # Caddy 客户端核心实现
├── client_test.go     # 单元测试
└── README.md          # 详细文档

examples/
├── caddy-demo/        # 基础演示
│   └── main.go
└── caddy-integration/ # 集成示例（发布服务）
    └── main.go
```

## 🚀 快速开始

### 1. 创建客户端

```go
import "github.com/mdfriday/hugoverse/internal/infrastructure/caddy"

config := &caddy.Config{
    AdminAPI:       "http://127.0.0.1:2019",
    BinaryPath:     "caddy",
    DefaultBackend: "127.0.0.1:1314",
    CoreDomain:     "mdfriday.site",
}

client := caddy.NewClient(config)
```

### 2. 启动服务器

```go
if err := client.StartServer(); err != nil {
    log.Fatal(err)
}
defer client.Stop()
```

### 3. 添加静态站点

```go
err := client.AddStaticSite("example.com", "/web/sites/example-com")
```

### 4. 查询证书状态

```go
certInfo, err := client.GetCertificateStatus("example.com")
fmt.Printf("Status: %s\n", certInfo.Status)
```

## 🎯 三大核心功能

### 功能 1: 启动 Caddy 服务器

**默认配置：**
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
              "handle": [
                {
                  "handler": "reverse_proxy",
                  "upstreams": [{"dial": "127.0.0.1:1314"}]
                }
              ]
            }
          ]
        }
      }
    }
  }
}
```

**代码：**
```go
client.StartServer()
```

### 功能 2: 动态添加自定义域名

**等效 curl 命令：**
```bash
curl -X POST \
  http://127.0.0.1:2019/config/apps/http/servers/main/routes \
  -H "Content-Type: application/json" \
  -d '{
    "@id": "site-example-com",
    "match": [{"host": ["example.com"]}],
    "handle": [
      {
        "handler": "file_server",
        "root": "/web/sites/example-com"
      }
    ]
  }'
```

**代码：**
```go
client.AddStaticSite("example.com", "/web/sites/example-com")
```

**特点：**
- ✅ 热加载，无需重启
- ✅ 自动申请 SSL 证书
- ✅ 自动 HTTP -> HTTPS 重定向

### 功能 3: 查询 SSL 证书状态

**代码：**
```go
certInfo, err := client.GetCertificateStatus("example.com")

// 证书状态
switch certInfo.Status {
case "issued":     // 已签发
case "pending":    // 申请中
case "not_found":  // 未找到
case "failed":     // 失败
}

// 证书详情（如果已签发）
if certInfo.Status == "issued" {
    fmt.Println(certInfo.Issuer)    // 签发者
    fmt.Println(certInfo.NotBefore) // 生效时间
    fmt.Println(certInfo.NotAfter)  // 过期时间
}
```

## 📊 API 方法列表

| 方法 | 功能 | 返回值 |
|------|------|--------|
| `StartServer()` | 启动 Caddy | error |
| `Stop()` | 停止 Caddy | error |
| `Ping()` | 检查连接 | error |
| `AddStaticSite(domain, path)` | 添加站点 | error |
| `RemoveStaticSite(domain)` | 移除站点 | error |
| `GetCertificateStatus(domain)` | 查询证书 | (*CertificateInfo, error) |
| `GetConfig()` | 获取配置 | (map[string]interface{}, error) |

## 🔧 常用 curl 命令

### 查看所有路由
```bash
curl http://127.0.0.1:2019/config/apps/http/servers/main/routes | jq
```

### 查看完整配置
```bash
curl http://127.0.0.1:2019/config/ | jq
```

### 删除路由
```bash
curl -X DELETE http://127.0.0.1:2019/id/site-example-com
```

### 查看证书
```bash
curl http://127.0.0.1:2019/config/apps/tls/certificates | jq
```

### 重新加载配置
```bash
curl -X POST http://127.0.0.1:2019/load -d @config.json
```

## 💡 使用场景

### 场景 1: License 激活后自动配置

```go
func OnLicenseActivated(userID, licenseKey string) error {
    // 创建站点目录
    sitePath := fmt.Sprintf("/web/sites/%s", userID)
    os.MkdirAll(sitePath, 0755)
    
    // 生成默认子域名
    domain := fmt.Sprintf("%s.mdfriday.site", userID)
    
    // 添加到 Caddy
    return caddyClient.AddStaticSite(domain, sitePath)
}
```

### 场景 2: 用户发布站点

```go
func PublishUserSite(userID, customDomain, siteName string) error {
    sitePath := fmt.Sprintf("/web/sites/%s/%s", userID, siteName)
    
    domain := customDomain
    if domain == "" {
        domain = fmt.Sprintf("%s.mdfriday.site", userID)
    }
    
    return caddyClient.AddStaticSite(domain, sitePath)
}
```

### 场景 3: 查看站点 SSL 状态

```go
func GetSiteStatus(domain string) (string, error) {
    certInfo, err := caddyClient.GetCertificateStatus(domain)
    if err != nil {
        return "", err
    }
    
    return certInfo.Status, nil
}
```

## ⚠️ 注意事项

1. **权限**：绑定 80/443 端口需要 root 权限
2. **DNS**：域名需要正确解析到服务器 IP
3. **防火墙**：确保 80/443 端口开放
4. **证书申请**：Let's Encrypt 有频率限制
5. **并发安全**：Client 是并发安全的

## 🧪 测试

```bash
# 单元测试
go test ./internal/infrastructure/caddy/

# 集成测试（需要 Caddy 二进制）
go test -v ./internal/infrastructure/caddy/

# 运行示例
go run ./examples/caddy-demo/main.go
```

## 📚 参考资源

- [Caddy 官方文档](https://caddyserver.com/docs/)
- [Admin API 文档](https://caddyserver.com/docs/api)
- [JSON 配置参考](https://caddyserver.com/docs/json/)
- [Let's Encrypt 文档](https://letsencrypt.org/docs/)

