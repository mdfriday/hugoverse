# Caddy Infrastructure Client

Caddy 基础设施客户端，用于管理 Caddy 服务器和动态配置。

## 功能特性

1. **启动 Caddy 服务器** - 使用默认配置启动 Caddy
2. **动态添加静态站点** - 通过 Admin API 热加载自定义域名
3. **查询 SSL 证书状态** - 检查域名的证书签发状态

## 安装

确保已安装 Caddy：

```bash
# macOS
brew install caddy

# Linux
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy
```

## 使用示例

### 1. 启动 Caddy 服务器

```go
package main

import (
    "fmt"
    "log"
    "github.com/mdfriday/hugoverse/internal/infrastructure/caddy"
)

func main() {
    // 创建客户端配置
    config := &caddy.Config{
        AdminAPI:       "http://127.0.0.1:2019",
        BinaryPath:     "caddy",                // Caddy 二进制路径
        DefaultBackend: "127.0.0.1:1314",       // 默认后端服务
        CoreDomain:     "mdfriday.site",        // 核心域名
    }

    // 创建客户端
    client := caddy.NewClient(config)

    // 启动服务器（使用默认配置）
    if err := client.StartServer(); err != nil {
        log.Fatalf("Failed to start Caddy: %v", err)
    }

    fmt.Println("✅ Caddy server started successfully")
    
    // 保持运行...
    select {}
}
```

默认配置包含：
- Admin API: `127.0.0.1:2019`
- HTTP/HTTPS 监听: `:80`, `:443`
- 核心域名反向代理: `mdfriday.site` -> `127.0.0.1:1314`

### 2. 动态添加自定义域名静态站点

```go
// 添加静态站点
err := client.AddStaticSite("example.com", "/web/sites/example-com")
if err != nil {
    log.Fatalf("Failed to add static site: %v", err)
}

fmt.Println("✅ Static site added: example.com")
```

这会向 Caddy Admin API 发送请求，实时添加路由：

```json
{
  "@id": "site-example-com",
  "match": [
    { "host": ["example.com"] }
  ],
  "handle": [
    {
      "handler": "file_server",
      "root": "/web/sites/example-com"
    }
  ]
}
```

### 3. 查询 SSL 证书状态

```go
// 查询证书状态
certInfo, err := client.GetCertificateStatus("example.com")
if err != nil {
    log.Fatalf("Failed to get certificate status: %v", err)
}

fmt.Printf("Domain: %s\n", certInfo.Domain)
fmt.Printf("Status: %s\n", certInfo.Status)

if certInfo.Status == "issued" {
    fmt.Printf("Issuer: %s\n", certInfo.Issuer)
    fmt.Printf("Valid from: %s\n", certInfo.NotBefore)
    fmt.Printf("Valid until: %s\n", certInfo.NotAfter)
}
```

证书状态可能的值：
- `issued` - 证书已签发
- `pending` - 正在申请中
- `not_found` - 未找到证书
- `failed` - 申请失败

### 4. 完整示例

```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/mdfriday/hugoverse/internal/infrastructure/caddy"
)

func main() {
    // 1. 创建客户端
    config := &caddy.Config{
        AdminAPI:       "http://127.0.0.1:2019",
        DefaultBackend: "127.0.0.1:1314",
        CoreDomain:     "mdfriday.site",
    }
    client := caddy.NewClient(config)

    // 2. 启动服务器
    if err := client.StartServer(); err != nil {
        log.Fatalf("Failed to start Caddy: %v", err)
    }
    defer client.Stop()
    
    fmt.Println("✅ Caddy started")
    
    // 等待服务器完全启动
    time.Sleep(2 * time.Second)

    // 3. 添加自定义域名静态站点
    sites := map[string]string{
        "blog.example.com": "/web/sites/blog",
        "docs.example.com": "/web/sites/docs",
    }

    for domain, path := range sites {
        if err := client.AddStaticSite(domain, path); err != nil {
            log.Printf("Failed to add site %s: %v", domain, err)
            continue
        }
        fmt.Printf("✅ Added site: %s -> %s\n", domain, path)
    }

    // 4. 查询证书状态
    time.Sleep(5 * time.Second) // 等待证书申请

    for domain := range sites {
        certInfo, err := client.GetCertificateStatus(domain)
        if err != nil {
            log.Printf("Failed to check cert for %s: %v", domain, err)
            continue
        }
        
        fmt.Printf("📜 Certificate for %s: %s\n", domain, certInfo.Status)
    }

    // 5. 获取当前配置
    config, err := client.GetConfig()
    if err != nil {
        log.Printf("Failed to get config: %v", err)
    } else {
        fmt.Printf("📋 Current routes: %d\n", 
            len(config["apps"].(map[string]interface{})["http"].(map[string]interface{})["servers"].(map[string]interface{})["main"].(map[string]interface{})["routes"].([]interface{})))
    }

    // 6. 移除站点
    if err := client.RemoveStaticSite("blog.example.com"); err != nil {
        log.Printf("Failed to remove site: %v", err)
    } else {
        fmt.Println("✅ Removed site: blog.example.com")
    }

    // 保持运行
    fmt.Println("\n🚀 Caddy is running. Press Ctrl+C to stop.")
    select {}
}
```

## API 方法

### StartServer()
启动 Caddy 服务器，使用默认配置。

### AddStaticSite(domain, sitePath string)
添加静态站点路由，支持热加载。

**参数：**
- `domain`: 域名（如 `example.com`）
- `sitePath`: 静态文件目录路径（如 `/web/sites/example-com`）

### GetCertificateStatus(domain string) (*CertificateInfo, error)
查询域名的 SSL 证书状态。

**返回：**
```go
type CertificateInfo struct {
    Domain     string    // 域名
    Status     string    // 状态: issued, pending, not_found, failed
    NotBefore  time.Time // 生效时间
    NotAfter   time.Time // 过期时间
    Issuer     string    // 签发者
    Error      string    // 错误信息（如果有）
}
```

### RemoveStaticSite(domain string)
移除静态站点路由。

### Stop()
停止 Caddy 服务器（优雅关闭）。

### Ping()
检查 Caddy Admin API 是否可访问。

### GetConfig() (map[string]interface{}, error)
获取当前完整的 Caddy 配置。

## 直接使用 curl 命令

### 添加静态站点
```bash
curl -X POST \
  http://127.0.0.1:2019/config/apps/http/servers/main/routes \
  -H "Content-Type: application/json" \
  -d '{
    "@id": "site-example-com",
    "match": [
      { "host": ["example.com"] }
    ],
    "handle": [
      {
        "handler": "file_server",
        "root": "/web/sites/example-com"
      }
    ]
  }'
```

### 查看当前配置
```bash
curl http://127.0.0.1:2019/config/ | jq
```

### 查看所有路由
```bash
curl http://127.0.0.1:2019/config/apps/http/servers/main/routes | jq
```

### 删除路由
```bash
curl -X DELETE http://127.0.0.1:2019/id/site-example-com
```

### 查看证书
```bash
curl http://127.0.0.1:2019/config/apps/tls/certificates | jq
```

## 配置文件

如果需要持久化配置，可以指定配置文件路径：

```go
config := &caddy.Config{
    ConfigPath: "/etc/caddy/config.json",
    // ... 其他配置
}
```

配置文件格式（JSON）：
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
              "match": [
                {
                  "host": ["mdfriday.site"]
                }
              ],
              "handle": [
                {
                  "handler": "reverse_proxy",
                  "upstreams": [
                    {
                      "dial": "127.0.0.1:1314"
                    }
                  ]
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

## 注意事项

1. **权限要求**：绑定 80/443 端口需要 root 权限或 CAP_NET_BIND_SERVICE 能力
2. **自动 HTTPS**：Caddy 会自动为所有域名申请 Let's Encrypt 证书
3. **热重载**：通过 Admin API 的修改无需重启服务器
4. **并发安全**：Client 是并发安全的

## 测试

运行单元测试：
```bash
go test ./internal/infrastructure/caddy
```

运行集成测试（需要 Caddy 二进制）：
```bash
go test -v ./internal/infrastructure/caddy
```

跳过集成测试：
```bash
go test -short ./internal/infrastructure/caddy
```

## 参考链接

- [Caddy 官方文档](https://caddyserver.com/docs/)
- [Caddy Admin API](https://caddyserver.com/docs/api)
- [Caddy JSON 配置](https://caddyserver.com/docs/json/)

