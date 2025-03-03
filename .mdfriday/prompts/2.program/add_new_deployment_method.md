# 添加新部署方式到Host Domain的流程 (Chain of Thought)

## 1. 理解当前部署系统架构

首先，让我们通过Chain of Thought方法分析当前的部署系统架构，以便理解如何添加新的部署方式。

### 1.1 系统结构分析

从代码结构可以看出，系统采用了接口和实现分离的设计模式：

- **domain/host/type.go**: 定义了部署相关的接口
  - `Deployer`: 基本部署接口，包含`Deploy`方法
  - `Result`: 部署结果接口
  - `SCPDeployer`: SCP特定的部署接口，扩展了`Deployer`
  - `AuthMethod`: 认证方法接口

- **domain/host/entity/**: 包含具体的部署实现
  - `Host`: 主要实体，组合了不同的部署方式
  - `Netlify`: Netlify部署实现
  - `SCPHost`: SCP部署实现

### 1.2 部署流程分析

当前系统支持两种部署方式：

1. **Netlify部署**：通过Netlify API将站点部署到Netlify平台
2. **SCP部署**：通过SSH/SCP协议将站点部署到远程服务器

`Host`实体的`Deploy`方法根据配置选择适当的部署方式：

```go
func (h *Host) Deploy(localPath string) (host.Result, error) {
    if h.SCPHost != nil {
        return h.SCPHost.Deploy(localPath)
    }

    if h.Netlify != nil {
        return h.Netlify.Deploy(localPath)
    }

    return nil, errors.New("no deployment method available")
}
```

## 2. 添加新部署方式的步骤

通过Chain of Thought分析，添加新部署方式（例如AWS S3/CloudFront）需要以下步骤：

### 2.1 定义值对象（Value Object）

在`internal/domain/host/valueobject`目录下创建配置值对象：

```go
// internal/domain/host/valueobject/aws_config.go
package valueobject

type AWSConfig struct {
    Region          string
    AccessKeyID     string
    SecretAccessKey string
    BucketName      string
    CloudFrontID    string // 可选，用于CloudFront分发
    RootPath        string // 可选，指定S3桶中的根路径
}

func (c *AWSConfig) Validate() error {
    if c.Region == "" {
        return errors.New("AWS region is required")
    }
    if c.AccessKeyID == "" {
        return errors.New("AWS access key ID is required")
    }
    if c.SecretAccessKey == "" {
        return errors.New("AWS secret access key is required")
    }
    if c.BucketName == "" {
        return errors.New("S3 bucket name is required")
    }
    return nil
}

// DeployResult实现host.Result接口
type AWSDeployResult struct {
    DeploymentID string
    DeploymentURL string
    Message      string
    DeployedSize int64
}

func (r *AWSDeployResult) GetID() string {
    return r.DeploymentID
}

func (r *AWSDeployResult) GetURL() string {
    return r.DeploymentURL
}

func (r *AWSDeployResult) GetMessage() string {
    return r.Message
}

func (r *AWSDeployResult) GetSize() int64 {
    return r.DeployedSize
}
```

### 2.2 实现部署实体

在`internal/domain/host/entity`目录下创建新的部署实体：

```go
// internal/domain/host/entity/aws.go
package entity

import (
    "github.com/mdfriday/hugoverse/internal/domain/host"
    "github.com/mdfriday/hugoverse/internal/domain/host/valueobject"
    "github.com/mdfriday/hugoverse/pkg/loggers"
    
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/cloudfront"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type AWSHost struct {
    config *valueobject.AWSConfig
    log    loggers.Logger
    
    session *session.Session
    s3Client *s3.S3
    cfClient *cloudfront.CloudFront
}

func NewAWSHost(config *valueobject.AWSConfig) (*AWSHost, error) {
    if err := config.Validate(); err != nil {
        return nil, err
    }
    
    return &AWSHost{
        config: config,
        log:    loggers.NewDefault(),
    }, nil
}

func (a *AWSHost) Connect() error {
    sess, err := session.NewSession(&aws.Config{
        Region:      aws.String(a.config.Region),
        Credentials: credentials.NewStaticCredentials(
            a.config.AccessKeyID,
            a.config.SecretAccessKey,
            "",
        ),
    })
    if err != nil {
        return err
    }
    
    a.session = sess
    a.s3Client = s3.New(sess)
    a.cfClient = cloudfront.New(sess)
    
    return nil
}

func (a *AWSHost) Deploy(localPath string) (host.Result, error) {
    // 1. 连接到AWS
    if a.session == nil {
        if err := a.Connect(); err != nil {
            return nil, err
        }
    }
    
    // 2. 上传文件到S3
    totalSize, err := a.uploadDirectory(localPath)
    if err != nil {
        return nil, err
    }
    
    // 3. 如果配置了CloudFront，创建失效请求
    var message string
    if a.config.CloudFrontID != "" {
        if err := a.invalidateCloudFront(); err != nil {
            message = "Files uploaded but CloudFront invalidation failed: " + err.Error()
        } else {
            message = "Files uploaded and CloudFront invalidation created"
        }
    } else {
        message = "Files uploaded successfully"
    }
    
    // 4. 构建并返回结果
    result := &valueobject.AWSDeployResult{
        DeploymentID:  time.Now().Format("20060102150405"),
        DeploymentURL: a.getDeploymentURL(),
        Message:       message,
        DeployedSize:  totalSize,
    }
    
    return result, nil
}

func (a *AWSHost) uploadDirectory(localPath string) (int64, error) {
    // 实现目录上传逻辑
    // ...
}

func (a *AWSHost) invalidateCloudFront() error {
    // 实现CloudFront缓存失效逻辑
    // ...
}

func (a *AWSHost) getDeploymentURL() string {
    if a.config.CloudFrontID != "" {
        // 返回CloudFront URL
        return fmt.Sprintf("https://%s.cloudfront.net", a.config.CloudFrontID)
    }
    
    // 返回S3 URL
    return fmt.Sprintf("https://%s.s3-website-%s.amazonaws.com", 
        a.config.BucketName, a.config.Region)
}
```

### 2.3 更新Host实体

修改`internal/domain/host/entity/host.go`文件，添加新的部署方式：

```go
package entity

import (
    "errors"
    "github.com/mdfriday/hugoverse/internal/domain/host"
)

type Host struct {
    *Netlify
    *SCPHost
    *AWSHost  // 添加新的部署方式
}

// Deploy implements the Deployer interface
func (h *Host) Deploy(localPath string) (host.Result, error) {
    if h.SCPHost != nil {
        return h.SCPHost.Deploy(localPath)
    }

    if h.Netlify != nil {
        return h.Netlify.Deploy(localPath)
    }
    
    if h.AWSHost != nil {  // 添加新的部署方式判断
        return h.AWSHost.Deploy(localPath)
    }

    return nil, errors.New("no deployment method available")
}
```

### 2.4 添加工厂方法

在`internal/domain/host/factory`目录下添加或修改工厂方法：

```go
// internal/domain/host/factory/factory.go
package factory

import (
    "github.com/mdfriday/hugoverse/internal/domain/host/entity"
    "github.com/mdfriday/hugoverse/internal/domain/host/valueobject"
)

// 添加创建AWS部署的工厂方法
func NewAWSHost(config *valueobject.AWSConfig) (*entity.Host, error) {
    awsHost, err := entity.NewAWSHost(config)
    if err != nil {
        return nil, err
    }
    
    return &entity.Host{
        AWSHost: awsHost,
    }, nil
}
```

### 2.5 更新应用层

在`internal/application`目录下更新应用服务：

```go
// internal/application/host.go
func (a *App) DeployToAWS(localPath, region, accessKey, secretKey, bucket, cloudfrontID string) (host.Result, error) {
    config := &valueobject.AWSConfig{
        Region:          region,
        AccessKeyID:     accessKey,
        SecretAccessKey: secretKey,
        BucketName:      bucket,
        CloudFrontID:    cloudfrontID,
    }
    
    host, err := factory.NewAWSHost(config)
    if err != nil {
        return nil, err
    }
    
    return host.Deploy(localPath)
}
```

### 2.6 添加API处理程序

在`internal/interfaces/api/handler`目录下添加或修改处理程序：

```go
// internal/interfaces/api/handler/handledeploy.go
func (s *Handler) DeployToAWSHandler(res http.ResponseWriter, req *http.Request) {
    // 1. 解析请求参数
    if err := req.ParseForm(); err != nil {
        s.res.Error(res, err)
        return
    }
    
    // 2. 获取部署参数
    region := req.FormValue("region")
    accessKey := req.FormValue("access_key")
    secretKey := req.FormValue("secret_key")
    bucket := req.FormValue("bucket")
    cloudfrontID := req.FormValue("cloudfront_id") // 可选
    
    // 3. 获取本地路径
    localPath := application.PublicDir()
    
    // 4. 调用应用服务进行部署
    result, err := s.hostApp.DeployToAWS(localPath, region, accessKey, secretKey, bucket, cloudfrontID)
    if err != nil {
        s.res.Error(res, err)
        return
    }
    
    // 5. 返回部署结果
    s.res.JSON(res, map[string]interface{}{
        "id":      result.GetID(),
        "url":     result.GetURL(),
        "message": result.GetMessage(),
        "size":    result.GetSize(),
    })
}
```

### 2.7 注册API路由

在`internal/interfaces/api/handlers.go`文件中注册新的路由：

```go
func (s *Server) registerContentHandler() {
    // 现有路由...
    
    // 添加新的部署路由
    s.mux.HandleFunc("/api/deploy/aws", s.wrapContentHandler(s.handler.DeployToAWSHandler))
}
```

## 3. 测试新部署方式

添加新部署方式后，应进行以下测试：

1. **单元测试**: 测试AWSHost的各个方法
2. **集成测试**: 测试与AWS服务的集成
3. **端到端测试**: 测试完整的部署流程
4. **手动测试**: 使用API进行实际部署

## 4. 总结

添加新部署方式的完整流程：

1. **分析需求**: 确定新部署方式的功能和配置
2. **定义值对象**: 创建配置和结果值对象
3. **实现部署实体**: 实现具体的部署逻辑
4. **更新Host实体**: 将新部署方式集成到Host实体
5. **添加工厂方法**: 创建用于构建新部署实体的工厂方法
6. **更新应用层**: 在应用层添加使用新部署方式的方法
7. **添加API处理程序**: 实现API处理程序
8. **注册API路由**: 注册新的API路由
9. **测试**: 确保新部署方式正常工作

通过这种方式，可以保持系统的架构清晰，同时扩展系统支持新的部署方式。 