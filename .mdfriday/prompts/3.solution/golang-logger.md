你是一名 Golang 开发者，现在我们要用pkg/logger包来实现日志功能，请确保：

如果是需要输出耗时信息的函数，则按照以上步骤来输出performance日志：
1. 在函数入口获取logger Info实例
2. 紧扫着用defer关键字，及loggers.TimeTrackf方法输出时间信息

如果只是普通日志，则按以下步骤来输出**带多field**的日志，
1. **先检查pkg/logger里是否有LogFields struct结构体，如果没有进行创建**：
    - 实现Fields() logg.Fields接口
2. **创建LogFields**：
    - 需要根据前的字段要求，把这些字段都添加到LogFields中
3. **打印日志**：
    - 用logger选取合适的日志等级，如Info, Error
    - 带上Fields信息
    - 带上相关等级信息
    - 最后带上附加信息

请**基于 ReAct 框架**，逐步推理并编写 Golang 代码，实现一个合理高效的 **日志打印功能**。
