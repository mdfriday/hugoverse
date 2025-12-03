# 📝 **《MDFriday License 体系需求文档》**

版本：v2.0 - 最新实现版本
作者：程序员闪电侠
适用系统：后端（Golang），前端（TypeScript / Browser / Electron）
更新日期：2025-12-03

---

# 1. 背景与目标

MDFriday 提供免费版、包年版、永久版三种授权模式。

* **免费版**：可使用部分主题，30天试用期，构建输出包含 mdfriday.com branding。
* **包年版**：包含部分服务器端服务（如发布站点、统计、提供额外主题），1年有效期。
* **永久版**：不使用后端服务，仅解锁额外主题与去除 branding，必须做到**本地可验证，无需联网**，永不过期。

**重要更新**：系统现在支持**多设备激活**，每个 license 默认支持最多 **3台设备** 同时使用。

为支持永久版，系统需要：

* 可验证 license 的合法性
* 可安全解锁主题（防止直接复制）
* 不依赖后端实时验证
* 仅增加破解成本，不追求绝对安全

因此需要设计一个 **可离线验证的 License + 内容加密体系**。

---

# 2. 整体架构设计

核心思路：

* 使用 **License Key** 格式：`MDF-XXXX-XXXX-XXXX`，不绑定用户ID，支持多设备激活
* **多级加密体系**：使用不同的 KEK 私钥加密不同等级的内容
  * **KEK_BASIC**: 加密基础内容，lifetime + yearly 都能访问
  * **KEK_PREMIUM**: 加密高级内容，只有 yearly 能访问
* 每个付费内容根据访问等级使用相应的 KEK 加密
* license 根据计划类型包含不同的 resourceKeys：
  * **Lifetime**: 只包含 basic resourceKey
  * **Yearly**: 包含 basic + premium resourceKeys
* license payload 使用 **ECDSA 私钥签名**
* 前端通过 API 激活 license key，获得设备绑定的完整 license
* 前端仅需要两个公钥：
  * RSA 公钥：用于解密 resourceKeys → 获取对应的 CEK
  * ECDSA 公钥：用于验证 license 是否伪造

**安全优势**：
* 后端私钥永不泄露，只有后端能加密内容
* 前端只能解密，无法加密新内容
* 密码学层面的访问控制，无法绕过
* 不同等级内容使用不同密钥，完全隔离

---

# 3. 专业术语定义

| 名称                          | 说明                            |
| --------------------------- | ----------------------------- |
| License Key                 | MDF-XXXX-XXXX-XXXX 格式的许可证密钥    |
| CEK（Content Encryption Key） | 用 AES-GCM 加密主题内容的密钥           |
| KEK_BASIC                   | 用于加密基础内容的 RSA 密钥对            |
| KEK_PREMIUM                 | 用于加密高级内容的 RSA 密钥对            |
| resourceKeys                | 包含多个等级的加密 CEK 的映射表          |
| Content Level               | 内容访问等级：basic（基础）、premium（高级） |
| License Detail              | 包含 license key 详细信息的 JSON 文件  |
| License Registry            | 包含生成的 license keys 数组的注册文件   |
| Device ID                   | 设备唯一标识符，用于设备绑定              |
| License Payload             | 包含设备信息和 resourceKeys 的 JSON 对象 |
| Signature                   | 后端对 payload 的 ECDSA 签名，用于防伪   |
| Public Keys                 | 前端持有两个公钥：RSA（解密）, ECDSA（验签）   |

---

# 4. License 类型定义

### 4.1 免费版（30天试用）

* 需要激活 license key（MDF-XXXX-XXXX-XXXX）
* 30天有效期，过期后需要升级
* 构建输出包含 mdfriday branding
* 支持最多3台设备激活

### 4.2 永久版（离线认证）

* 使用 license key 激活后完全离线工作
* 激活后本地存储 `.mdf.license` 文件
* 验签通过后解锁**基础内容**
* 永不过期（9999-01-01）
* 支持最多3台设备激活
* **访问权限**：只能解密 basic 级别的内容

### 4.3 包年版（混合认证）

* 初次激活需要联网验证 license key
* 激活后可离线使用，直到过期
* 1年有效期，过期后需要续费
* 支持最多3台设备激活
* **访问权限**：可以解密 basic + premium 级别的所有内容

---

# 5. 数据结构定义

## 5.1 License Detail Schema（详细信息文件）

### Lifetime License Detail
```json
{
  "licenseKey": "MDF-ABCD-EFGH-JKLM",
  "plan": "lifetime",
  "issueDate": "2025-12-03",
  "expiryDate": "9999-01-01",
  "maxActivations": 3,
  "currentActivations": 1,
  "deviceIds": ["device-123"],
  "resourceKeys": {
    "basic": "base64-encoded-encrypted-basic-cek"
  },
  "version": 1
}
```

### Yearly License Detail
```json
{
  "licenseKey": "MDF-NOPQ-RSTU-VWXY",
  "plan": "yearly",
  "issueDate": "2025-12-03",
  "expiryDate": "2026-12-03",
  "maxActivations": 3,
  "currentActivations": 1,
  "deviceIds": ["device-456"],
  "resourceKeys": {
    "basic": "base64-encoded-encrypted-basic-cek",
    "premium": "base64-encoded-encrypted-premium-cek"
  },
  "version": 1
}
```

## 5.2 License Registry Schema（注册文件）

```json
{
  "generatedAt": "2025-12-03",
  "plan": "lifetime",
  "count": 5,
  "licenseKeys": [
    "MDF-ABCD-EFGH-JKLM",
    "MDF-NOPQ-RSTU-VWXY"
  ]
}
```

## 5.3 License Payload Schema（激活后的载荷）

### Lifetime License Payload
```json
{
  "licenseKey": "MDF-ABCD-EFGH-JKLM",
  "deviceId": "device-123",
  "plan": "lifetime",
  "exp": null,
  "resourceKeys": {
    "basic": "base64-encoded-encrypted-basic-cek"
  },
  "issueAt": "2025-12-03T14:30:00Z",
  "version": 1
}
```

### Yearly License Payload
```json
{
  "licenseKey": "MDF-NOPQ-RSTU-VWXY",
  "deviceId": "device-456",
  "plan": "yearly",
  "exp": "2026-12-03T14:30:00Z",
  "resourceKeys": {
    "basic": "base64-encoded-encrypted-basic-cek",
    "premium": "base64-encoded-encrypted-premium-cek"
  },
  "issueAt": "2025-12-03T14:30:00Z",
  "version": 1
}
```

说明：

| 字段                | 说明                                    |
| ----------------- | ------------------------------------- |
| licenseKey        | License Key (MDF-XXXX-XXXX-XXXX)     |
| deviceId          | 设备唯一标识符                               |
| plan              | 授权类型 free/yearly/lifetime             |
| issueDate         | 签发日期 (YYYY-MM-DD)                    |
| expiryDate        | 过期日期 (YYYY-MM-DD, lifetime为9999-01-01) |
| maxActivations    | 最大激活设备数量（默认3）                        |
| currentActivations | 当前已激活设备数量                             |
| deviceIds         | 已激活的设备ID列表                            |
| resourceKeys      | 分级加密的 CEK 映射表                        |
| resourceKeys.basic | 基础内容的加密 CEK（所有付费 license 都有）        |
| resourceKeys.premium | 高级内容的加密 CEK（只有 yearly license 有）   |
| version           | license schema 版本                     |

---

# 6. 多级加密架构详解

## 6.1 加密等级定义

系统支持两个内容访问等级：

| 等级      | 说明           | 可访问的 License 类型 | KEK 类型      |
| ------- | ------------ | --------------- | ----------- |
| basic   | 基础内容，所有付费用户可访问 | lifetime + yearly | KEK_BASIC   |
| premium | 高级内容，仅高级用户可访问  | yearly only     | KEK_PREMIUM |

## 6.2 License 权限分配

不同类型的 license 包含不同的 resourceKeys：

### Lifetime License
```json
{
  "resourceKeys": {
    "basic": "encrypted_basic_cek"
  }
}
```
- ✅ 可以解密 basic 级别内容
- ❌ 无法解密 premium 级别内容

### Yearly License
```json
{
  "resourceKeys": {
    "basic": "encrypted_basic_cek",
    "premium": "encrypted_premium_cek"
  }
}
```
- ✅ 可以解密 basic 级别内容
- ✅ 可以解密 premium 级别内容

## 6.3 内容加密流程

1. **后端加密内容**：
   ```bash
   # 基础内容 - 所有付费用户可访问
   hugov license encrypt-level -input theme.json -level basic
   # 输出: theme.json.basic.enc
   
   # 高级内容 - 只有年费用户可访问
   hugov license encrypt-level -input premium_theme.json -level premium
   # 输出: premium_theme.json.premium.enc
   ```

2. **加密格式**：
   ```
   [level_len][level_string][cek_len][encrypted_cek][encrypted_content]
   ```

3. **解密验证**：
   - 系统检查 license 是否包含对应等级的 resourceKey
   - 如果没有权限，直接拒绝解密
   - 密码学层面强制执行访问控制

---

# 7. End-to-End 加密设计流程

## 6.1 密钥体系

后端维护五套密钥：

| 类型              | 算法            | 用途                | 前端可看到？    |
| --------------- | ------------- | ----------------- | --------- |
| ECDSA 密钥对       | ECDSA P-256   | 签名 license        | 公钥可见      |
| RSA 密钥对         | RSA-OAEP 2048 | 包装 license 中的 CEK | 公钥可见      |
| KEK_BASIC 密钥对   | RSA-OAEP 2048 | 加密基础内容            | 私钥后端专用    |
| KEK_PREMIUM 密钥对 | RSA-OAEP 2048 | 加密高级内容            | 私钥后端专用    |
| AES CEK         | AES-GCM       | 加密具体内容            | 仅加密后的结果可见 |

---

# 7. License 生成与激活流程（后端 Golang）

## 7.1 License Key 生成流程

1. 管理员使用 CLI 命令生成指定数量的 license keys
2. 根据 license plan 生成对应的 resourceKeys：
   - **Lifetime**: 生成 basic CEK，用 RSA 包装 → basic resourceKey
   - **Yearly**: 生成 basic + premium CEK，分别用 RSA 包装 → basic + premium resourceKeys
3. 创建 license detail 文件（包含 resourceKeys、激活信息、有效期等）
4. 生成 registry 文件（包含所有 license keys 列表）
5. 保存到磁盘：`MDF-XXXX-XXXX-XXXX.json` 和 `licenses_[plan]_[date].json`

## 7.2 License 激活流程

1. 前端调用 `/api/license/activate` API
2. 后端验证 license key 格式和有效性
3. 检查设备激活数量限制（最多3台）
4. 如果设备已激活，直接返回 license
5. 如果是新设备且未达到限制，添加到设备列表
6. 创建包含设备绑定和对应 resourceKeys 的 license payload
7. 使用 ECDSA 私钥签名 payload
8. 返回完整的激活响应（license + 公钥 + 详情）

## 7.3 CLI 命令

```bash
# 生成密钥对（包含 KEK_BASIC 和 KEK_PREMIUM）
hugov license keygen

# 生成 license keys
hugov license generate -plan lifetime -count 5 -output-dir ./licenses
hugov license generate -plan yearly -count 3 -output-dir ./licenses

# 加密内容（分级加密）
hugov license encrypt-level -input ./basic_theme.json -level basic
hugov license encrypt-level -input ./premium_theme.json -level premium

# 测试激活（开发用）
hugov license activate -key MDF-HCWU-SE9K-3HHJ -device-id test-device

# 验证 license
hugov license verify -license ./licenses/MDF-HCWU-SE9K-3HHJ_lifetime.mdf.license

# 解密内容
hugov license decrypt -encrypted ./basic_theme.json.basic.enc -license ./activated.mdf.license
```

---

# 8. 前端激活与验证流程（TypeScript）

## 8.1 License Key 激活流程

```typescript
// 1. 用户输入 license key 和生成设备ID
const licenseKey = "MDF-ABCD-EFGH-JKLM";
const deviceId = generateDeviceId(); // 前端生成唯一设备标识

// 2. 调用激活 API
const response = await fetch('/api/license/activate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ licenseKey, deviceId })
});

const result = await response.json();

if (result.success) {
  // 3. 保存激活后的 license 和公钥
  localStorage.setItem('mdf_license', JSON.stringify(result.license));
  localStorage.setItem('mdf_public_keys', JSON.stringify(result.publicKeys));
  
  // 4. 验证并解锁功能
  await verifyAndUnlock(result.license, result.publicKeys);
}
```

## 8.2 License 验证步骤

```typescript
async function verifyAndUnlock(license, publicKeys) {
  // 1. 验证 signature 是否有效（ECDSA）
  const isValidSignature = await verifyECDSASignature(
    license.payload, 
    license.signature, 
    publicKeys.ecdsaPublicKey
  );
  
  if (!isValidSignature) {
    throw new Error('Invalid license signature');
  }
  
  // 2. 解析 payload
  const payload = JSON.parse(atob(license.payload));
  
  // 3. 验证有效期
  if (payload.exp && new Date(payload.exp) < new Date()) {
    throw new Error('License has expired');
  }
  
  // 4. 根据内容等级选择合适的 resourceKey 解密
  const availableKeys = payload.resourceKeys;
  
  // 5. 解密不同等级的内容
  const basicThemes = availableKeys.basic ? 
    await decryptThemes(availableKeys.basic, publicKeys.rsaPublicKey, 'basic') : [];
  const premiumThemes = availableKeys.premium ? 
    await decryptThemes(availableKeys.premium, publicKeys.rsaPublicKey, 'premium') : [];
  
  // 6. 加载解锁主题
  loadUnlockedThemes([...basicThemes, ...premiumThemes], payload.plan);
}
```

## 8.3 前端需要的公钥

前端从激活 API 或公钥端点获取：

```typescript
// 从激活响应获取
const publicKeys = activationResponse.publicKeys;

// 或从专用端点获取
const publicKeys = await fetch('/api/license/public-keys').then(r => r.json());

// 公钥格式
interface PublicKeys {
  ecdsaPublicKey: string; // PEM 格式，用于验证签名
  rsaPublicKey: string;   // PEM 格式，用于解密 resourceKey
}
```

---

# 9. 文件格式与存储约定

## 9.1 后端存储结构

```
licenses/                                    # License 存储目录
├── licenses_lifetime_2025-12-03.json      # Registry 文件
├── licenses_yearly_2025-12-03.json        # Registry 文件  
├── MDF-ABCD-EFGH-JKLM.json               # License 详情文件
├── MDF-NOPQ-RSTU-VWXY.json               # License 详情文件
└── ...

~/.mdfriday/keys/                           # 密钥存储目录
├── ecdsa_private.pem                       # ECDSA 私钥
├── ecdsa_public.pem                        # ECDSA 公钥
├── rsa_private.pem                         # RSA 私钥（用于包装 license 中的 CEK）
├── rsa_public.pem                          # RSA 公钥
├── basic_kek_private.pem                   # KEK_BASIC 私钥（加密基础内容）
├── basic_kek_public.pem                    # KEK_BASIC 公钥
├── premium_kek_private.pem                 # KEK_PREMIUM 私钥（加密高级内容）
└── premium_kek_public.pem                  # KEK_PREMIUM 公钥
```

## 9.2 前端存储结构

```
localStorage:
├── mdf_license          # 激活后的 license 对象
├── mdf_public_keys      # 公钥对象
└── mdf_device_id        # 设备唯一标识

本地文件（可选）:
└── ~/Library/MDFriday/
    └── activated_license.mdf               # 激活后的 license 备份
```

## 9.3 主题内容加密文件结构

```
themes/
├── basic_theme_1/
│   ├── theme.json.basic.enc                # 基础级别加密的主题配置
│   ├── template.html.basic.enc             # 基础级别加密的模板文件
│   ├── style.css.basic.enc                 # 基础级别加密的样式文件
│   ├── preview.jpg                         # 预览图（未加密）
│   └── meta.json                           # 元数据（未加密）
├── premium_theme_1/
│   ├── theme.json.premium.enc              # 高级级别加密的主题配置
│   ├── template.html.premium.enc           # 高级级别加密的模板文件
│   ├── style.css.premium.enc               # 高级级别加密的样式文件
│   ├── preview.jpg                         # 预览图（未加密）
│   └── meta.json                           # 元数据（未加密）
└── premium_theme_2/
    └── ...
```

其中：
- `*.basic.enc` 为基础级别加密内容，lifetime + yearly license 都能解密
- `*.premium.enc` 为高级级别加密内容，只有 yearly license 能解密

---

# 10. 安全性与目标

明确：
**不追求绝对安全，只追求提升破解成本，使普通用户无法轻易绕过。**

对抗方式：

| 攻击类型         | 方案                                       |
| ------------ | ---------------------------------------- |
| 修改前端代码绕过验证   | 使用签名 + 分级加密内容双重保护                        |
| 分享付费主题       | 主题内容按等级加密，需要对应等级的 license 才能解密         |
| 分享 license   | 设备绑定限制，最多3台设备，超出后无法激活                  |
| 设备 ID 伪造     | 设备 ID 包含在签名中，伪造会导致签名验证失败               |
| lifetime 访问高级内容 | 密码学层面隔离，lifetime license 无法获得 premium KEK |
| 离线破解         | 需要同时伪造 license + 破解多级加密 + 绕过设备绑定，成本极高   |
| 批量注册免费试用     | 可通过后端监控异常激活模式，实施 IP 或其他限制              |

---

# 11. API 接口规范（Golang 后端）

## 11.1 激活 License（前端使用，无需认证）

```http
POST /api/license/activate
Content-Type: application/json

{
  "licenseKey": "MDF-ABCD-EFGH-JKLM",
  "deviceId": "unique-device-identifier"
}
```

成功响应：
```json
{
  "success": true,
  "license": {
    "payload": "base64-encoded-license-data",
    "signature": "ecdsa-signature"
  },
  "publicKeys": {
    "ecdsaPublicKey": "-----BEGIN PUBLIC KEY-----...",
    "rsaPublicKey": "-----BEGIN PUBLIC KEY-----..."
  },
  "detail": {
    "licenseKey": "MDF-ABCD-EFGH-JKLM",
    "plan": "lifetime",
    "currentActivations": 1,
    "maxActivations": 3,
    "expiryDate": "9999-01-01"
  }
}
```

失败响应：
```json
{
  "success": false,
  "errorMsg": "Maximum activations reached (3/3)"
}
```

## 11.2 获取公钥（前端初始化时使用，无需认证）

```http
GET /api/license/public-keys
```

返回：
```json
{
  "ecdsaPublicKey": "-----BEGIN PUBLIC KEY-----...",
  "rsaPublicKey": "-----BEGIN PUBLIC KEY-----..."
}
```

## 11.3 验证 License Key 格式（前端使用，无需认证）

```http
POST /api/license/validate
Content-Type: application/json

{
  "licenseKey": "MDF-ABCD-EFGH-JKLM"
}
```

返回：
```json
{
  "valid": true,
  "licenseKey": "MDF-ABCD-EFGH-JKLM",
  "plan": "lifetime",
  "expiryDate": "9999-01-01",
  "maxActivations": 3,
  "currentActivations": 1,
  "expired": false
}
```

## 11.4 CLI 管理命令（后端管理员使用）

```bash
# 生成密钥对（包含多级 KEK）
hugov license keygen

# 批量生成 license keys
hugov license generate -plan lifetime -count 100 -output-dir ./licenses
hugov license generate -plan yearly -count 50 -output-dir ./licenses

# 加密不同等级的内容
hugov license encrypt-level -input ./basic_content.json -level basic
hugov license encrypt-level -input ./premium_content.json -level premium

# 验证 license 文件
hugov license verify -license ./MDF-ABCD-EFGH-JKLM_lifetime.mdf.license

# 测试激活流程
hugov license activate -key MDF-ABCD-EFGH-JKLM -device-id test-device

# 解密内容
hugov license decrypt -encrypted ./basic_content.json.basic.enc -license ./activated.mdf.license
```

---

# 12. 前端（TypeScript）实现模块

## 12.1 核心模块

| 模块                                    | 说明                        | 状态   |
| ------------------------------------- | ------------------------- | ---- |
| `generateDeviceId()`                  | 生成设备唯一标识符                 | 需实现  |
| `activateLicense(key, deviceId)`      | 调用激活 API                 | 需实现  |
| `verifyLicense(payload, signature)`   | 验证 ECDSA 签名              | 需实现  |
| `unwrapCEK(resourceKey, rsaPublicKey)` | 用 RSA-OAEP 公钥解出 CEK      | 需实现  |
| `decryptContent(cek, encContent)`     | AES-GCM 解密主题内容           | 需实现  |
| `saveLicense(license)`                | 保存激活后的 license           | 需实现  |
| `loadLicense()`                       | 加载本地保存的 license          | 需实现  |
| `isLicenseValid(payload)`             | 校验 plan、exp、设备绑定         | 需实现  |
| `getUnlockedFeatures(plan)`           | 根据计划类型获取解锁功能             | 需实现  |

## 12.2 设备标识生成

```typescript
function generateDeviceId(): string {
  // 组合多个设备特征生成唯一标识
  const features = [
    navigator.userAgent,
    navigator.language,
    screen.width + 'x' + screen.height,
    new Date().getTimezoneOffset(),
    // 可添加更多设备特征
  ];
  
  // 生成稳定的设备 ID
  return btoa(features.join('|')).replace(/[^a-zA-Z0-9]/g, '').substring(0, 32);
}
```

## 12.3 License 状态管理

```typescript
interface LicenseState {
  isActivated: boolean;
  plan: 'free' | 'yearly' | 'lifetime';
  expiryDate: string | null;
  deviceId: string;
  features: string[];
}

class LicenseManager {
  async activate(licenseKey: string): Promise<boolean>;
  async verify(): Promise<boolean>;
  async getUnlockedThemes(): Promise<Theme[]>;
  isFeatureUnlocked(feature: string): boolean;
  getRemainingDays(): number | null;
}
```

---

# 13. 实现状态总结

## 13.1 已完成功能 ✅

- [x] 后端 License Key 生成系统
- [x] **多级加密体系（KEK_BASIC + KEK_PREMIUM）**
- [x] **分级访问控制（lifetime vs yearly）**
- [x] 多设备激活支持（最多3台）
- [x] 统一日期格式 (YYYY-MM-DD)
- [x] CLI 管理工具完整实现
- [x] **分级内容加密命令（encrypt-level）**
- [x] API 接口（激活、验证、公钥获取）
- [x] 加密体系（ECDSA + RSA + AES-GCM）
- [x] 文件存储格式（Registry + Detail）
- [x] 设备绑定和限制机制
- [x] **密码学层面的访问控制**

## 13.2 待实现功能 🚧

- [ ] 前端 TypeScript 模块
- [ ] 主题内容加密工具
- [ ] 设备 ID 生成算法优化
- [ ] License 吊销机制
- [ ] 使用统计和监控
- [ ] 批量管理工具

## 13.3 测试验证 ✅

- [x] CLI 命令完整测试
- [x] 激活流程验证
- [x] 多设备限制测试
- [x] 签名验证测试
- [x] API 接口测试
- [x] **多级加密完整测试**
- [x] **访问控制验证（lifetime 无法访问 premium）**
- [x] **内容完整性验证**
- [x] **端到端加密解密测试**

---

# 14. 部署和使用指南

## 14.1 后端部署

1. 生成密钥对：`hugov license keygen`
2. 批量生成 license：`hugov license generate -plan lifetime -count 100`
3. 启动 API 服务器（包含 license 端点）
4. 将 license 文件分发给用户

## 14.2 前端集成

1. 实现设备 ID 生成
2. 集成激活 API 调用
3. 实现 license 验证和存储
4. 根据 license 状态控制功能访问
5. 实现主题解密和加载

## 14.3 用户使用流程

1. 用户获得 license key（MDF-XXXX-XXXX-XXXX）
2. 在应用中输入 license key
3. 系统自动生成设备 ID 并调用激活 API
4. 激活成功后解锁相应功能
5. 后续启动时自动验证本地 license


