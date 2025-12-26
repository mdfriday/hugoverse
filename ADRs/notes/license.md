# License API 测试命令

## 重要说明

⚠️ **所有 License API 都需要 TOKEN 认证**

所有 License 相关的 API 调用都需要在 Header 中提供有效的 JWT TOKEN：
```bash
-H "Authorization: Bearer YOUR_TOKEN"
```

## 1. 注册测试用户

```bash
curl -X POST http://127.0.0.1:1314/api/user \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=mdf_public@mdfriday.com&password=987123"

curl -X POST http://127.0.0.1:1314/api/login \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=mdf_public@mdfriday.com&password=987123"
```

**响应示例**:
```json
{
  "data": [
    "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9..."
  ]
}
```

## 2. 创建 License (使用真实 API)

使用 `/api/content?type=License` 端点创建 License，需要 TOKEN 认证。

```bash
curl -v -X POST "http://127.0.0.1:1314/api/content?type=License" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9..." \
-F "id=-1" \
-F "license_key=MDF-STARTER-TEST-001" \
-F "plan=starter" \
-F "expiry_days=365" \
-F "max_devices=3" \
-F "max_ips=3"
```

**参数说明**:
- `id=-1`: 表示创建新记录
- `license_key`: License 密钥（格式：MDF-XXXX-XXXX-XXXX）
- `plan`: 套餐类型（starter/creator/pro/enterprise）
- `expiry_days`: 有效期天数（默认 365）
- `max_devices`: 最大设备数（默认 3）
- `max_ips`: 最大 IP 数（默认 3）

## 3. 激活 License (需要 TOKEN)

激活 License 需要 TOKEN 认证。

```bash
curl -X POST http://127.0.0.1:1314/api/license/activate \
-H "Authorization: Bearer YOUR_TOKEN" \
-F "license_key=MDF-STARTER-TEST-001" \
-F "device_id=device-001" \
-F "device_name=MacBook Pro" \
-F "device_type=desktop"
```

**响应示例**:
```json
{
  "success": true,
  "message": "License activated successfully",
  "license": {
    "license_key": "MDF-STARTER-TEST-001",
    "plan": "starter",
    "activated": true,
    "is_valid": true,
    "features": {...}
  },
  "sync": {
    "email": "starter-test-001@mdfriday.com",
    "db_name": "userdb-xxx",
    "db_endpoint": "http://127.0.0.1:5984/userdb-xxx"
  }
}
```

## 4. 查询 License 信息 (需要 TOKEN)

```bash
curl -X GET "http://127.0.0.1:1314/api/license/info?key=MDF-STARTER-TEST-001" \
-H "Authorization: Bearer YOUR_TOKEN"
```

## 5. 查询设备列表 (需要 TOKEN)

```bash
curl -X GET "http://127.0.0.1:1314/api/license/devices?key=MDF-STARTER-TEST-001" \
-H "Authorization: Bearer YOUR_TOKEN"
```

## 6. 查询 IP 列表 (需要 TOKEN)

```bash
curl -X GET "http://127.0.0.1:1314/api/license/ips?key=MDF-STARTER-TEST-001" \
-H "Authorization: Bearer YOUR_TOKEN"
```

## 7. 查询 Sync 信息 (需要 TOKEN)

```bash
curl -X GET "http://127.0.0.1:1314/api/license/sync?key=MDF-STARTER-TEST-001" \
-H "Authorization: Bearer YOUR_TOKEN"
```

## 8. 封禁设备 (需要 TOKEN)

```bash
curl -X POST http://127.0.0.1:1314/api/license/device/block \
-H "Authorization: Bearer YOUR_TOKEN" \
-F "license_key=MDF-STARTER-TEST-001" \
-F "device_id=device-001"
```

## 9. 封禁 IP (需要 TOKEN)

```bash
curl -X POST http://127.0.0.1:1314/api/license/ip/block \
-H "Authorization: Bearer YOUR_TOKEN" \
-F "license_key=MDF-STARTER-TEST-001" \
-F "ip_address=192.168.1.100"
```

## 完整测试流程

运行完整的自动化测试脚本：

```bash
bash ADRs/notes/test-license-real.sh
```

测试脚本将：
1. 注册新用户并获取 TOKEN
2. 创建 3 个不同套餐的 License（Starter/Creator/Pro）
3. 激活其中一个 License
4. 验证 CouchDB 数据库创建
5. 测试设备和 IP 限制
6. 测试幂等性和各种边界情况
7. 自动清理测试数据

## License 套餐对比

| 套餐 | 设备数 | IP数 | Sync | Sync配额 | 站点数 | 存储 | 自定义域名 |
|-----|-------|-----|------|---------|-------|------|----------|
| Starter | 3 | 3 | ✅ | 500MB | 3 | 1GB | ❌ |
| Creator | 5 | 5 | ✅ | 2GB | 10 | 5GB | ✅ |
| Pro | 10 | 10 | ✅ | 10GB | 50 | 20GB | ✅ |
| Enterprise | 无限 | 无限 | ✅ | 50GB | 无限 | 100GB | ✅ |
