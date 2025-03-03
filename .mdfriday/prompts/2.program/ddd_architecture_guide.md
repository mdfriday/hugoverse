# 项目DDD架构风格指南 (Chain of Thought)

## 1. 当前项目DDD架构分析

通过Chain of Thought方法，让我们分析当前项目的领域驱动设计(DDD)架构风格，以确保新功能遵循一致的模式。

### 1.1 整体架构层次

项目采用了典型的分层架构，符合DDD的战略设计原则：

```
internal/
├── domain/         # 领域层：核心业务逻辑和规则
│   ├── content/    # 内容领域
│   ├── admin/      # 管理员领域
│   ├── host/       # 部署主机领域
│   └── ...
├── application/    # 应用层：协调领域对象，实现用例
└── interfaces/     # 接口层：与外部系统交互
    ├── api/        # API接口
    └── ...
```

这种分层架构确保了：
- **领域层**包含核心业务逻辑，不依赖外部系统
- **应用层**协调领域对象，实现用例
- **接口层**处理与外部系统的交互

### 1.2 领域层结构分析

每个领域模块（如host、content等）都遵循一致的内部结构：

```
domain/host/
├── type.go         # 定义领域接口和类型
├── entity/         # 实体定义和实现
│   ├── host.go     # 主实体
│   ├── netlify.go  # 具体实体实现
│   └── scp.go      # 具体实体实现
├── valueobject/    # 值对象定义
│   ├── netlify_config.go
│   └── scp_config.go
└── factory/        # 工厂方法
    └── factory.go
```

### 1.3 DDD构建块使用模式

#### 1.3.1 接口定义 (type.go)

`type.go`文件用于定义领域接口和类型，作为领域契约：

```go
// domain/host/type.go
package host

// Result 是部署结果接口
type Result interface {
    GetID() string
    GetURL() string
    GetMessage() string
    GetSize() int64
}

// Deployer 是部署器接口
type Deployer interface {
    Deploy(localPath string) (Result, error)
}

// 其他特定接口...
type SCPDeployer interface {
    Deployer
    Connect() error
    Close() error
    // ...
}
```

这些接口定义了领域的行为契约，而不关心具体实现。

#### 1.3.2 实体 (entity/)

实体是具有唯一标识的领域对象，代表业务概念：

```go
// domain/host/entity/host.go
package entity

type Host struct {
    *Netlify
    *SCPHost
}

// Deploy 实现 Deployer 接口
func (h *Host) Deploy(localPath string) (host.Result, error) {
    // 实现逻辑...
}
```

实体特点：
- 具有唯一标识
- 包含业务逻辑和行为
- 可变状态
- 实现领域接口

#### 1.3.3 值对象 (valueobject/)

值对象是无标识的不可变对象，通常用于配置和参数：

```go
// domain/host/valueobject/netlify_config.go
package valueobject

type NetlifyConfig struct {
    AccessToken string
    SiteID      string
    // ...
}

func (c *NetlifyConfig) Validate() error {
    // 验证逻辑...
}
```

值对象特点：
- 无唯一标识
- 不可变
- 通过所有属性值判断相等性
- 可包含验证逻辑

#### 1.3.4 工厂 (factory/)

工厂负责创建复杂的领域对象，封装创建逻辑：

```go
// domain/host/factory/factory.go
package factory

func NewNetlifyHost(config *valueobject.NetlifyConfig) (*entity.Host, error) {
    netlify, err := entity.NewNetlifyWithConfig(config)
    if err != nil {
        return nil, err
    }
    
    return &entity.Host{
        Netlify: netlify,
    }, nil
}
```

工厂特点：
- 封装对象创建逻辑
- 确保创建的对象处于有效状态
- 隐藏创建细节

## 2. 扩展指南：添加新功能

基于上述分析，添加新功能时应遵循以下指南：

### 2.1 确定领域边界

首先确定新功能属于哪个领域，或是否需要创建新的领域：

```
internal/domain/newfeature/
```

### 2.2 定义领域接口 (type.go)

在新领域中，首先定义接口和类型：

```go
// internal/domain/newfeature/type.go
package newfeature

// 定义核心接口
type Service interface {
    DoSomething(param string) (Result, error)
}

// 定义结果接口
type Result interface {
    GetValue() string
    // ...
}

// 定义其他必要接口...
```

接口定义原则：
- 接口应该小而精确
- 遵循单一职责原则
- 只包含必要的方法
- 使用领域语言命名

### 2.3 实现值对象

创建必要的值对象：

```go
// internal/domain/newfeature/valueobject/config.go
package valueobject

type Config struct {
    // 不可变属性
    Parameter1 string
    Parameter2 int
}

func (c *Config) Validate() error {
    // 验证逻辑
    if c.Parameter1 == "" {
        return errors.New("parameter1 is required")
    }
    return nil
}

// 结果值对象
type OperationResult struct {
    Value string
    // 其他属性...
}

func (r *OperationResult) GetValue() string {
    return r.Value
}
```

值对象实现原则：
- 保持不可变性
- 包含验证逻辑
- 实现相关接口
- 不包含业务逻辑

### 2.4 实现实体

创建实体实现：

```go
// internal/domain/newfeature/entity/feature.go
package entity

import (
    "github.com/mdfriday/hugoverse/internal/domain/newfeature"
    "github.com/mdfriday/hugoverse/internal/domain/newfeature/valueobject"
)

type Feature struct {
    config *valueobject.Config
    // 其他依赖和状态...
}

func NewFeature(config *valueobject.Config) (*Feature, error) {
    if err := config.Validate(); err != nil {
        return nil, err
    }
    
    return &Feature{
        config: config,
    }, nil
}

// 实现Service接口
func (f *Feature) DoSomething(param string) (newfeature.Result, error) {
    // 实现业务逻辑
    
    result := &valueobject.OperationResult{
        Value: "processed: " + param,
    }
    
    return result, nil
}
```

实体实现原则：
- 验证构造参数
- 实现领域接口
- 包含业务逻辑
- 保持内部状态一致性

### 2.5 创建工厂

实现工厂方法：

```go
// internal/domain/newfeature/factory/factory.go
package factory

import (
    "github.com/mdfriday/hugoverse/internal/domain/newfeature/entity"
    "github.com/mdfriday/hugoverse/internal/domain/newfeature/valueobject"
)

func NewFeature(param1 string, param2 int) (*entity.Feature, error) {
    config := &valueobject.Config{
        Parameter1: param1,
        Parameter2: param2,
    }
    
    return entity.NewFeature(config)
}
```

工厂实现原则：
- 简化对象创建
- 封装创建逻辑
- 返回接口而非具体类型（当适用时）

### 2.6 更新应用层

在应用层添加新功能的用例：

```go
// internal/application/newfeature.go
package application

import (
    "github.com/mdfriday/hugoverse/internal/domain/newfeature"
    "github.com/mdfriday/hugoverse/internal/domain/newfeature/factory"
)

func (a *App) DoNewFeature(param1 string, param2 int, operationParam string) (newfeature.Result, error) {
    // 1. 创建领域对象
    feature, err := factory.NewFeature(param1, param2)
    if err != nil {
        return nil, err
    }
    
    // 2. 执行领域操作
    return feature.DoSomething(operationParam)
}
```

### 2.7 添加接口层实现

最后，在接口层添加API处理程序：

```go
// internal/interfaces/api/handler/handlenewfeature.go
func (s *Handler) NewFeatureHandler(res http.ResponseWriter, req *http.Request) {
    // 1. 解析请求参数
    param1 := req.FormValue("param1")
    param2, _ := strconv.Atoi(req.FormValue("param2"))
    operationParam := req.FormValue("operation")
    
    // 2. 调用应用服务
    result, err := s.app.DoNewFeature(param1, param2, operationParam)
    if err != nil {
        s.res.Error(res, err)
        return
    }
    
    // 3. 返回响应
    s.res.JSON(res, map[string]string{
        "value": result.GetValue(),
    })
}
```

## 3. DDD最佳实践总结

### 3.1 通用原则

1. **遵循领域语言**：使用业务领域的术语命名接口、类和方法
2. **保持领域纯净**：领域层不应依赖基础设施或外部系统
3. **接口隔离**：定义小而精确的接口
4. **单一职责**：每个类只负责一个功能
5. **封装变化**：隐藏实现细节，只暴露必要的接口

### 3.2 文件组织约定

- **type.go**：定义领域接口和类型
- **entity/xxx.go**：实现具体实体
- **valueobject/xxx.go**：定义值对象
- **factory/factory.go**：提供工厂方法

### 3.3 命名约定

- 接口名应简洁明了（如`Deployer`、`Result`）
- 实体名应反映业务概念（如`Host`、`Netlify`）
- 值对象名应表明其用途（如`NetlifyConfig`）
- 工厂方法应以`New`开头（如`NewNetlifyHost`）

## 4. 总结

当前项目采用了清晰的DDD架构风格，通过明确的分层和领域模型组织代码。添加新功能时，应遵循以下步骤：

1. 在`type.go`中定义领域接口
2. 在`valueobject/`中创建配置和结果值对象
3. 在`entity/`中实现核心业务逻辑
4. 在`factory/`中提供工厂方法
5. 在应用层协调领域对象
6. 在接口层处理外部交互

遵循这些指南，可以确保新功能与现有系统保持一致的架构风格，同时保持代码的可维护性和可扩展性。 