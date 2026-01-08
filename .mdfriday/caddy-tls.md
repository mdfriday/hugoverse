## 一、需求总结（What & Why）

### 1. 业务目标

你在构建一个 **支持用户自定义域名的静态站点 SaaS 平台**，核心目标是：

* 平台主域名：`mdfriday.com`
* 平台子域名（平台分配）：

    * `abc.mdfriday.com`
    * 使用 **Wildcard 证书**
* 用户自定义域名：

    * `hello.com`, `foo.com`, `bar.com`
    * 用户在自己的 DNS 服务商（如腾讯云 DNSPod）配置 A/AAAA 记录指向你的服务器
    * 访问时必须是 **公网可信 HTTPS（Let’s Encrypt）**

### 2. 技术约束

* 使用 **Caddy** 作为 Web Server 与 ACME 客户端
* Caddy 由 **Go 程序启动**，并通过 **Admin API 动态加载 JSON 配置**
* 不自行实现 ACME 协议
* 尽可能 **保证证书一次签发成功**，避免触发 Let’s Encrypt 限额
* 支持：

    * 动态注册站点
    * 多域名指向同一静态目录
    * 大规模用户自定义域名

---

## 二、核心设计原则

### 原则 1：证书签发完全交给 Caddy

* 所有证书：

    * 申请
    * 验证
    * 签发
    * 自动续期
* **全部由 Caddy 内置 ACME 完成**
* 你的系统不接触私钥、不实现 ACME 协议

你只做一件事：

> **确保 Caddy 发起 ACME 时，一定能成功**

---

### 原则 2：主域名与用户域名使用不同 ACME 策略

| 类型      | 域名                               | ACME Challenge |
| ------- | -------------------------------- | -------------- |
| 平台域名    | `mdfriday.com`, `*.mdfriday.com` | DNS-01（DNSPod） |
| 用户自定义域名 | `hello.com`                      | HTTP-01        |

原因：

* Wildcard 必须用 DNS-01
* 用户自定义域名你不掌控 DNS
* HTTP-01 对用户来说最简单，只需指向 IP

---

## 三、推荐的整体解决方案（How）

### 1️⃣ 固定配置（启动时预置）

在 Caddy 启动时，通过 Go 构建 JSON，**一次性预置**：

* `mdfriday.com`
* `*.mdfriday.com`

使用 DNSPod 的 DNS-01 challenge：

```json
{
  "subjects": ["mdfriday.com", "*.mdfriday.com"],
  "issuers": [
    {
      "module": "acme",
      "challenges": {
        "dns": {
          "provider": {
            "name": "dnspod",
            "api_token": "{env.DNSPOD_API_TOKEN}"
          }
        }
      }
    }
  ]
}
```

这是**稳定、不频繁变更**的证书策略。

---

### 2️⃣ 用户自定义域名的动态流程

#### Step 1：用户提交域名

例如：

```text
hello.com
```

#### Step 2：前置准备检查（你来做）

在触发 Caddy 之前，主动检查：

##### DNS 指向检查（必须）

* `net.LookupHost("hello.com")`
* 解析 IP 是否包含你的服务器公网 IP（或 LB IP）

##### HTTP 可达性检查（强烈推荐）

```text
http://hello.com/.well-known/acme-challenge/test
```

确保：

* 端口 80 可访问
* 请求能命中你的服务器

> 目的：避免 Let’s Encrypt HTTP-01 验证失败

---

#### Step 3：动态注册 Caddy 配置

通过 Caddy Admin API 动态注入：

1. **HTTP route**

    * `hello.com`
    * 指向用户静态目录 `/data/sites/u123`

2. **TLS automation policy**

    * 使用 ACME
    * HTTP-01 challenge

```json
{
  "subjects": ["hello.com"],
  "issuers": [
    {
      "module": "acme",
      "challenges": {
        "http": {}
      }
    }
  ]
}
```

---

#### Step 4：证书签发（Caddy 自动完成）

* Caddy 监听到新域名
* 自动触发 ACME
* Let’s Encrypt 访问 HTTP-01
* 验证通过 → 证书签发 → HTTPS 生效

你不参与任何证书细节。

---

## 四、证书分组与限额策略（关键）

### Let’s Encrypt 关键限制

| 限制         | 数值       |
| ---------- | -------- |
| 单证书 SAN 上限 | 100 个域名  |
| 同一组域名重复签发  | 5 次 / 周  |
| 每注册域名证书    | 50 张 / 周 |

---

### 推荐分组策略（非常重要）

* **10~20 个自定义域名为一组**
* 每组一个 ACME policy
* 新增域名 → 新建 policy
* 不要频繁修改已有 policy 的 subjects

#### 示例：

```text
group-1: 10 domains → cert-1
group-2: 10 domains → cert-2
```

优势：

* 极大降低重复签发风险
* HTTP-01 失败不会影响大量域名
* 易于扩展到数百/数千用户

---

## 五、职责边界总结

| 事项             | 责任方                |
| -------------- | ------------------ |
| DNS 是否指向正确     | 用户                 |
| DNS / HTTP 预检测 | 你                  |
| 路由映射           | 你（Caddy Admin API） |
| TLS policy 管理  | 你                  |
| ACME 流程        | Caddy              |
| 证书签发 / 续期      | Caddy              |
| HTTPS 安全性      | Let’s Encrypt      |

---

## 六、最终结论（一句话版）

> **你构建的是一个“证书准备系统”，而不是证书系统本身。**
>
> 你通过 DNS + HTTP 预检测，确保 Caddy 在正确的时间、用正确的策略，一次性成功完成 Let’s Encrypt HTTP-01 验证与证书签发。
