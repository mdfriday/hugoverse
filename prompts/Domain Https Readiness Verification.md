# 域名 HTTPS 就绪检测方案（Code Generation Prompt）

> 目标：
> 为平台型系统设计一套**稳定、可解释、与用户真实访问一致**的 HTTPS 就绪检测方案，
> 用于自动化域名接入与证书签发流程（基于 Caddy + ACME）。
> 该方案适用于：单服务器、单 IP、无 CDN、基于 Caddy 自动 HTTPS 的平台系统。

---

## 一、总体设计原则（必须遵守）

1. **用户视角优先**
   所有“是否就绪”的最终判断，必须以**真实用户访问行为**为准，而不是内部状态推断。

2. **ACME 过程不可被模拟**
   不尝试手动探测 `/.well-known/acme-challenge/*`，也不假设 ACME 状态可被提前判断。

3. **前置校验 ≠ 成功保证**
   DNS 检测只能用于排除明显错误配置，不能作为证书一定成功的承诺。

4. **主判据与辅判据分离**

    * 主判据：真实 HTTPS（TLS）连接是否成功
    * 辅判据：Caddy Admin API 提供的证书元信息

两个 API：

```go
s.mux.HandleFunc("/api/license/domain/add", s.wrapContentHandler(s.handler.AddDomainHandler)) - 已实现，第一阶段基于这个进行修改
s.mux.HandleFunc("/api/license/domain/https-status", s.wrapContentHandler(s.handler.DomainSSLStatusHandler)) - 新增，作为 HTTPS 主判据
```

---

## 二、添加自定义域名流程（推荐状态机）

```text
USER_ADDED_DOMAIN
        ↓
CHECK_DNS
        ↓ (通过)
ADD_CADDY_ROUTE
```

---

## 三、DNS 检测（CheckDNS）—— 必须，但只做这一件事

### 目的

* 判断域名是否**已经解析**
* 判断解析结果是否**指向当前服务器 IP**
* 尽早拦截明显错误配置

### 设计边界（非常重要）

* DNS 检测 **不等价于** HTTPS 一定可用
* DNS 检测 **不关心** ACME / 证书状态
* DNS 检测只回答一个问题：

> “从 DNS 角度看，这个域名是否大概率指向了本服务器？”

### 实现要求

* 使用 `net.Resolver.LookupIP`（而不是 `LookupHost`）
* 使用 `net.IP.Equal` 做比较（而不是字符串）
* 当前阶段（单 IP 部署）：

    * 只要求命中一个配置好的 Server IP

### CheckDNS 输出语义

| 字段          | 含义                 |
| ----------- | ------------------ |
| DNSValid    | 域名是否解析且指向本服务器      |
| ResolvedIPs | 实际解析到的 IP 列表       |
| Error       | DNS 级别错误说明（用于用户提示） |

---

## 四、明确 **不再需要** 的检测

### ❌ 不再进行 HTTP 可达性检测（CheckHTTP）

原因：

1. HTTP 可达 ≠ HTTPS 可达
2. HTTP 路径（尤其是 ACME challenge 路径）容易被：

    * CDN
    * 反向代理
    * ISP 劫持
      误导
3. ACME HTTP-01 的真实行为只能由 Caddy 内部完成，外部无法可靠模拟

**结论：**

> 移除所有 `http://domain/.well-known/acme-challenge/*` 相关探测逻辑。

---

## 五、HTTPS 就绪的【主判据】—— 模拟真实 TLS 访问

通过第二个 API 实现。

### 核心思想

> **如果一个真实用户通过 HTTPS 访问域名是成功的，那么 HTTPS 就是“已就绪”的。**

因此，系统必须：

* 主动发起 TLS 连接
* 使用 SNI（ServerName = domain）
* 校验证书是否真实有效

### 主判据：TLS Handshake 检测

#### 判断标准（全部满足才算就绪）

1. TCP + TLS 握手成功（`domain:443`）
2. 服务端返回证书
3. 证书满足：

    * `VerifyHostname(domain)` 通过
    * 当前时间在 `NotBefore ~ NotAfter` 区间内

#### 不关心的事项

* HTTP Status Code（200 / 404 / 403 都不重要）
* 具体业务路由是否存在

### TLS 检测输出语义

| 结果             | 状态        |
| -------------- | --------- |
| 握手成功 + 证书有效    | `ACTIVE`  |
| 握手失败 / timeout | `PENDING` |
| 明确证书错误         | `ERROR`   |

---

## 六、GetCertificateStatus 的正确定位（辅判据）

### 可以做什么

* 查询并展示：

    * 证书颁发者（Issuer）
    * 有效期（NotBefore / NotAfter）
    * 是否是 wildcard 证书
* 用于 Debug / 管理后台展示

### 明确不能做什么

* ❌ 不能作为“HTTPS 是否就绪”的最终判断
* ❌ 不能假设 `subjects` 命中就一定能成功访问
* ❌ 不能把 automation subjects 当作 ACME pending 状态

### 正确使用方式

```text
HTTPS 主判据 = TLS 探测结果
GetCertificateStatus = 补充信息
```
