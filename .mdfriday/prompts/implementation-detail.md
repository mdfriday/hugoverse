## 实现细节补充

本项目是一个web server，提供系列API，让用户可以创建站点，上传文件，预览站点，并部署站点。
我们已经实现了部署站点到Netlify上，现在要拓展部署功能，以支持更多的hosting类型。

现在我们要支持部署到私有服务。
也就是将生成的一个站点文件夹，里面所有的文件都上传到私有服务器。

在Domain里，我们已经有了一个Domain host。
所以我们需要对这个Domain进行拓展，让其支持部署到私有服务器上。

而且host是聚合根，所有的这些服务都应该由domain/host/entity/host.go来处理。
在factory里新加构建私有部署所需要的信息，这里的信息需要用接口来传入，也就是先在type.go里定义。
数据源从domain config里来。

我们提供了两种上传方式：

1. 直接上传（Deploy）：
   - 优点：简单直接，适合文件较少的情况
   - 缺点：每个文件需要单独建立连接，效率较低

2. 压缩上传（DeployWithTar）：
   - 优点：
     * 只需要一次文件传输
     * 压缩传输，节省带宽
     * 适合大量小文件的情况
   - 缺点：
     * 需要本地和远程都有额外的磁盘空间用于临时文件
     * 不适合单个大文件的情况

使用建议：
1. 如果是单个文件或少量文件，使用 `Deploy`
2. 如果是包含大量小文件的目录，使用 `DeployWithTar`

最后，提供一个测试，这样可以在本地测试部署到私有服务器上。
这个测试需要使用scripts/scp_env.sh来设置环境变量，然后运行source scripts/test_scp.sh来测试。

测试的步骤如下：
1. 使用scripts/scp_env.sh来设置环境变量。
2. 使用source scripts/test_scp.sh来运行测试。



## 实现细节补充

本项目是一个web server，提供系列API，让用户可以创建站点，上传文件，预览站点，并部署站点。
我们已经实现了部署站点到Netlify上，现在要拓展部署功能，以支持更多的hosting类型。

现在我们要支持部署到私有服务。
也就是将生成的一个站点文件夹，里面所有的文件都上传到私有服务器。

在Domain里，我们已经有了一个Domain host。
所以我们需要对这个Domain进行拓展，让其支持部署到私有服务器上。

而且host是聚合根，所有的这些服务都应该由domain/host/entity/host.go来处理。
在factory里新加构建私有部署所需要的信息，这里的信息需要用接口来传入，也就是先在type.go里定义。
数据源从domain config里来。

最后，提供一个测试，这样可以在本地测试部署到私有服务器上。
这个测试需要使用scripts/scp_env.sh来设置环境变量，然后运行source scripts/test_scp.sh来测试。

测试的步骤如下：
1. 使用scripts/scp_env.sh来设置环境变量。
2. 使用source scripts/test_scp.sh来运行测试。


