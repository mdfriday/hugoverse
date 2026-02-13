# GitHub Release Webhook 集成

## 概述

本功能实现了 GitHub Webhook 集成，用于自动下载 `mdfriday/obsidian-friday-plugin` 仓库的最新 release 版本。

## 功能特性

- ✅ 验证 GitHub Webhook 签名 (HMAC-SHA256)
- ✅ 只处理 `published` 事件
- ✅ 自动下载最新版本的 zipball
- ✅ 保存为固定文件名 `friday-latest.zip`
- ✅ 原子性文件写入（先写入临时文件，完成后再重命名）
- ✅ 配置化管理（通过 Admin 配置）

## 配置管理

### Admin 配置结构

GitHub 相关配置已集成到 Admin Domain，遵循 DDD 架构：

```
internal/domain/admin/
├── entity/
│   └── github.go           # GitHub 实体（提供配置访问接口）
├── valueobject/
│   └── config.go           # Config 值对象（存储配置数据）
├── factory/
│   └── admin.go            # Admin 工厂（初始化 GitHub 实体）
└── type.go                 # GitHub 接口定义
```

### 配置字段

| 字段 | 类型 | 说明 | 默认值 |
|-----|-----|-----|--------|
| `github_hook_secret` | string | GitHub Webhook Secret | 无 |
| `github_token` | string | GitHub Personal Access Token | 无 |
| `github_target_repo` | string | 目标仓库全名 | `mdfriday/obsidian-friday-plugin` |

### 配置方法

#### 方式 1: 通过管理后台配置

1. 访问管理后台配置页面
2. 找到 **GitHub 配置** 部分
3. 填写以下字段：
   - **GitHub Webhook Secret**: 在 GitHub 配置的 webhook secret
   - **GitHub Personal Access Token**: GitHub 个人访问令牌（用于下载私有仓库）
   - **GitHub Target Repository**: 目标仓库（默认 `mdfriday/obsidian-friday-plugin`）
4. 保存配置

#### 方式 2: 通过配置文件

编辑 Admin 配置 JSON 文件：

```json
{
  "github_hook_secret": "your-webhook-secret-here",
  "github_token": "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "github_target_repo": "mdfriday/obsidian-friday-plugin"
}
```

## GitHub 配置

### 1. 生成 Personal Access Token

1. 访问 GitHub → **Settings** → **Developer settings** → **Personal access tokens** → **Tokens (classic)**
2. 点击 **Generate new token (classic)**
3. 设置：
   - **Note**: `MDFriday Release Webhook`
   - **Expiration**: 按需选择
   - **Scopes**: 勾选 `repo` (Full control of private repositories)
4. 生成并复制 token

### 2. 配置 Webhook

进入 `mdfriday/obsidian-friday-plugin` 仓库的设置：

1. **Settings** → **Webhooks** → **Add webhook**

2. **配置参数**:
   - **Payload URL**: `https://your-domain.com/api/github-hook`
   - **Content type**: `application/x-www-form-urlencoded`
   - **Secret**: 自定义 secret（需与 Admin 配置中的 `github_hook_secret` 一致）
   - **SSL verification**: Enable SSL verification (推荐)
   - **Which events**: 选择 **Let me select individual events**，只勾选 **Releases**
   - **Active**: ✅ 勾选

3. 点击 **Add webhook**

### 3. 验证配置

GitHub 会发送一个 `ping` 事件来验证配置。查看 Webhook 的 **Recent Deliveries** 标签，确认：
- Status: 200 OK
- Response body: 显示成功消息

## 工作流程

```
GitHub Release Published
        ↓
发送 Webhook 到 /api/github-hook
        ↓
从 Admin 配置读取 HookSecret, Token, TargetRepository
        ↓
验证签名 (X-Hub-Signature-256 vs HookSecret)
        ↓
解析 payload (application/x-www-form-urlencoded)
        ↓
检查事件类型 (只处理 published)
        ↓
检查仓库名称 (与 TargetRepository 比较)
        ↓
使用 Token 下载 zipball (从 release.zipball_url)
        ↓
保存到 UploadDir/friday-latest.zip
        ↓
返回 200 OK
```

## 安全性

### 签名验证

- 使用 HMAC-SHA256 验证 webhook 签名
- 使用常量时间比较防止时序攻击
- 签名不匹配时返回 401 Unauthorized

### 配置化管理

- Secret 和 Token 存储在 Admin 配置中
- 支持通过管理后台安全更新
- 未配置时返回友好错误提示

### 请求过滤

- 只处理 POST 请求
- 只处理 `published` 事件
- 只处理目标仓库的请求

## API 端点

### POST /api/github-hook

**Headers**:
```
Content-Type: application/x-www-form-urlencoded
X-Hub-Signature-256: sha256=<hex-signature>
X-GitHub-Event: release
```

**Request Body**:
```
payload=<url-encoded-json>
```

**Response**:
- `200 OK`: Webhook 处理成功
- `401 Unauthorized`: 签名验证失败
- `400 Bad Request`: 请求格式错误
- `500 Internal Server Error`: 下载失败或配置错误

## 测试

### 配置测试环境

```bash
# 1. 启动服务器
./hugov serve

# 2. 配置 Admin（可选，如果已配置可跳过）
# 访问管理后台或编辑配置文件

# 3. 确保配置了以下值：
# - github_hook_secret: "your-test-secret"
# - github_token: "ghp_xxxxxxxxxxxx"
# - github_target_repo: "mdfriday/obsidian-friday-plugin"
```

### 运行测试脚本

**注意**: 测试脚本需要更新以使用配置的 secret：

```bash
# 编辑 examples/test_github_webhook.sh
# 将 SECRET 变量改为你的配置值
SECRET="your-test-secret"  # 与 Admin 配置中的 github_hook_secret 一致

# 运行测试
./examples/test_github_webhook.sh
```

### 检查下载结果

```bash
# 查看文件
ls -lh ~/.mdfriday/upload/friday-latest.zip

# 验证 zip 文件
unzip -t ~/.mdfriday/upload/friday-latest.zip
```

## 代码架构

### Domain 层（DDD 架构）

```
internal/domain/admin/
├── type.go                 # GitHub 接口定义
│   └── GitHub interface {
│       HookSecret() string
│       GithubToken() string
│       TargetRepository() string
│   }
├── entity/
│   ├── admin.go            # Admin 聚合根（包含 *GitHub）
│   └── github.go           # GitHub 实体（实现 GitHub 接口）
├── valueobject/
│   └── config.go           # Config 值对象（存储配置数据）
└── factory/
    └── admin.go            # Admin 工厂（初始化 GitHub 实体）
```

### Infrastructure 层

```
internal/infrastructure/github/
├── webhook.go              # Webhook 签名验证和 payload 解析
├── webhook_test.go         # 单元测试
└── release.go              # Release 下载逻辑
```

### Interface 层

```
internal/interfaces/api/handler/
└── handlegithub.go         # GitHub Webhook 处理器
```

## 注意事项

1. **配置管理**: 所有敏感信息（Secret、Token）通过 Admin 配置管理，不硬编码
2. **Token 权限**: GitHub Token 需要 `repo` 权限（用于访问私有仓库）
3. **文件覆盖**: 每次下载会覆盖旧的 `friday-latest.zip`
4. **下载超时**: 默认超时 5 分钟，适用于大文件
5. **磁盘空间**: 确保 UploadDir 有足够的空间
6. **日志记录**: 所有操作都会记录日志，便于排查问题

## 故障排查

### Webhook 未触发

- 检查 GitHub Webhook 配置是否正确
- 查看 GitHub 的 Recent Deliveries 日志
- 确认服务器可以从外网访问

### 签名验证失败

- 确认 Admin 配置中的 `github_hook_secret` 与 GitHub Webhook 配置的 Secret 一致
- 检查 Content-Type 是否为 `application/x-www-form-urlencoded`
- 查看服务器日志中的签名信息

### 配置未生效

- 检查 Admin 配置是否正确保存
- 重启服务器使配置生效
- 查看日志确认配置加载情况

### 下载失败

- 检查网络连接
- 确认 `github_token` 配置正确且有效
- 确认 Token 有 `repo` 权限
- 确认 UploadDir 目录有写权限
- 查看服务器日志中的详细错误信息
- 验证 GitHub API 是否可访问

## 未来改进

- [ ] 支持配置多个仓库的 webhook
- [ ] 添加下载进度日志
- [ ] 添加文件版本管理（保留历史版本）
- [ ] 实现 webhook 重试机制
- [ ] 添加更详细的指标和监控
- [ ] 支持 webhook 签名算法配置

