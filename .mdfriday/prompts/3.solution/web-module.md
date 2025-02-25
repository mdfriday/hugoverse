你现在是一名高级golang开发人员，现在要实现一个用browser查看所有模块目录结构的服务。

具体实现步骤如下：

1. 注册一个新的子命令modules
2. 获取所有模块Dir信息。
   - 参考application/prompt.go的GeneratePrompt方法，我们要实现的方法名叫GetModulesAbsDir方法，以获取所有的模块Dir
   - 在GeneratePrompt中，先创建logger实例，我们默认是dev环境，然后需要调用loadConfig，createModule获取模块信息
   - 通过module的方法：All() []module.Module 方法，获取所有的模块信息
   - 在Module 接口中，从 Dir() string 方法可以获取模块的文件系统绝对地址
3. 启动一个web服务，用来展示各模块的文件结构
4. 展示的顺序按All()返回的顺序展示
5. 因为这些模块的目录结构都相同，所以在展示的时侯，让模块文件的目录在一条线上，这样方便对比查看同一目录有哪些相同的文件
