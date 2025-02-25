# 重构logger不合理的使用

## 问题1: 自定义log.Fields，应该放在logger包中

在添加Fields时，需要使用logger.WithFields(log.Fields{
	"field1": "value1",
	"field2": "value2",
})

这个Fields是一个接口，需要实现Fields() logg.Fields接口。有的类自己实现了一个这样的结构体。
但这是不合理的，因为logger是一个公共pkg，公共组件的实现不应该依赖于具体的业务。
我们需要把相关的定义放在logger包中。

## 重构建议

1. 在logger包中创建一个LogFields结构体，实现Fields() logg.Fields接口。
2. 对外提供新建立，添加field的方法


## 问题2: 生产环境和开发环境使用不同的logger

我们要区分生产环境和开发环境，用合适的日志级别，将日志都写入文件。

## 重构建议

1. 为每一次