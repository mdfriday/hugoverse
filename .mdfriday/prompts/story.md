# SCP 目录上传器实现 Prompt

### 目标
实现一个基于 Go 的 SCP 目录上传工具，支持通过用户名和密码将整个目录递归上传到远程服务器。

### 关键特性
- ✅ 递归上传整个目录结构
- ✅ 自动创建远程目录
- ✅ 保持文件权限
- ✅ 不依赖系统 scp 命令
- ✅ 用户名密码认证
- ✅ 跨平台支持

### 依赖包
```go
import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
    "golang.org/x/crypto/ssh"
)
```

### 核心结构
```go
type SCPUploader struct {
    Host     string
    Port     int
    Username string
    Password string
}
```

### 实现步骤

1. **创建上传器实例**
   - 初始化配置信息
   - 设置连接参数

2. **建立 SSH 连接**
   - 配置 SSH 客户端
   - 设置认证方式
   - 处理超时

3. **递归处理目录**
   - 使用 `filepath.Walk` 遍历
   - 区分文件和目录处理
   - 计算相对路径

4. **目录处理**
   - 远程创建目录结构
   - 保持权限

5. **文件上传**
   - 发送文件元信息
   - 传输文件内容
   - 发送结束标记

### 使用示例
```go
uploader := NewSCPUploader("server.com", 22, "user", "pass")
err := uploader.UploadDirectory("/local/path", "/remote/path")
```

### 注意事项
1. 生产环境需要proper host key verification
2. 考虑添加进度回调
3. 可选择性添加并发支持
4. 注意远程路径的斜杠格式

### 扩展建议
1. 添加传输进度显示
2. 实现并发上传队列
3. 支持断点续传
4. 添加传输速度限制
5. 支持其他认证方式（密钥等）

### 错误处理
- SSH 连接失败
- 文件访问权限
- 远程目录创建失败
- 传输中断