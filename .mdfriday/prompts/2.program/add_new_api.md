# 新增API流程分析 (Chain of Thought)

## 1. 理解当前系统架构

首先，让我们通过Chain of Thought方法分析系统架构，以便理解如何添加新的API。

### 1.1 系统结构分析

从代码结构可以看出，系统采用了类似于Clean Architecture或Hexagonal Architecture的分层架构:

- **domain层**: 包含核心业务逻辑和实体
- **application层**: 包含应用服务和用例
- **interfaces层**: 包含与外部系统交互的接口，如API、数据库等

API相关代码主要位于`internal/interfaces/api`目录下，包含以下关键组件:

- **server.go**: 定义Server结构体，负责HTTP服务器的初始化和配置
- **handlers.go**: 定义API路由和中间件包装器
- **handler/**: 包含具体的API处理程序实现

### 1.2 API处理流程

从`handlers.go`文件可以看出，API处理流程如下:

1. 在`Server`结构体中注册路由
2. 使用中间件包装处理程序，如认证、CORS、压缩等
3. 处理程序实现具体的业务逻辑

中间件包装器有:
- `wrapContentHandler`: 用于内容相关API，包含记录、CORS、压缩、数据库连接和认证
- `wrapAdminHandler`: 用于管理员API，包含数据库连接和带重定向的认证

## 2. 添加新API的步骤

通过Chain of Thought分析，添加新API需要以下步骤:

### 2.1 定义处理程序

在`internal/interfaces/api/handler/`目录下创建或修改处理程序文件:

```go
// 在现有文件中添加新的处理程序函数，或创建新文件
func (s *Handler) NewFeatureHandler(res http.ResponseWriter, req *http.Request) {
    // 1. 解析请求参数
    q := req.URL.Query()
    // 或解析JSON请求体
    // var requestBody struct { ... }
    // json.NewDecoder(req.Body).Decode(&requestBody)
    
    // 2. 调用应用服务
    result, err := s.someApp.SomeFunction(...)
    if err != nil {
        // 处理错误
        s.res.Error(res, err)
        return
    }
    
    // 3. 返回响应
    s.res.JSON(res, result)
}
```

### 2.2 注册路由

在`internal/interfaces/api/handlers.go`文件中注册新的路由:

```go
func (s *Server) registerContentHandler() {
    // 现有路由...
    
    // 添加新路由
    s.mux.HandleFunc("/api/new-feature", s.wrapContentHandler(s.handler.NewFeatureHandler))
}
```

根据API的性质，选择适当的包装器:
- 内容相关API: `wrapContentHandler`
- 管理员API: `wrapAdminHandler`
- 用户相关API: 直接使用中间件组合

### 2.3 实现业务逻辑

如果需要，在domain或application层实现相应的业务逻辑:

1. 在domain层定义实体和接口
2. 在application层实现用例
3. 在interfaces层调用application层的服务

## 3. 中间件分析

系统使用了多层中间件来处理请求，理解这些中间件对于正确实现API很重要:

### 3.1 wrapContentHandler中间件链

从内到外的中间件链:
1. `auth.Check`: 验证用户认证
2. `db.Open`: 打开数据库连接
3. `comp.Gzip`: 启用Gzip压缩
4. `cors.Handle`: 处理跨域请求
5. `record.Collect`: 记录请求信息

### 3.2 wrapAdminHandler中间件链

从内到外的中间件链:
1. `auth.CheckWithRedirect`: 验证管理员认证，失败时重定向
2. `db.Open`: 打开数据库连接

## 4. 测试新API

添加新API后，应进行以下测试:

1. 单元测试: 测试处理程序的业务逻辑
2. 集成测试: 测试API端点的功能
3. 手动测试: 使用工具如Postman或curl测试API

## 5. 总结

添加新API的完整流程:

1. **分析需求**: 确定API的功能、输入和输出
2. **实现业务逻辑**: 在domain和application层实现核心功能
3. **创建处理程序**: 在handler目录下实现HTTP处理程序
4. **注册路由**: 在handlers.go中注册路由并应用适当的中间件
5. **测试**: 确保API正常工作并符合需求

通过这种方式，可以保持系统的架构清晰，同时确保新API与现有系统无缝集成。 