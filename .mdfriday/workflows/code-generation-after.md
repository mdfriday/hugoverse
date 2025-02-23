{{ .Content "prompts/implementation-detail.md" }}

## 修改原则

不要修改pkg里面的代码，而是修改internal里面的代码。

## 代码生成日志规则

1. 不要用fmt来实现，而是用loggers.Logger来实现。

log 使用样例

先定义
Log      loggers.Logger `json:"-"`
pagesLog logg.LevelLogger

定义Level log并可以获取执行时间
ch.pagesLog = ch.Log.InfoCommand("ContentHub.ProcessPages")
defer loggers.TimeTrackf(ch.pagesLog, time.Now(), nil, "")

在子函数中，可以定义step名
processLog := ch.pagesLog.WithField("step", "process")
defer loggers.TimeTrackf(processLog, time.Now(), nil, "")

然后就可以输出日志了
ch.pagesLog.Logf("%s", "value")

log 代码样例：

uploadLog := h.logger.Info()
	defer loggers.TimeTrackf(uploadLog, time.Now(), nil, "")

	fields := h.newSCPFields("file_upload")
	fields.addFields(
		logg.Field{Name: "localPath", Value: localPath},
		logg.Field{Name: "remotePath", Value: remotePath},
	)
	uploadLog.WithFields(fields).Logf("Starting file upload")

可以参考contenthub domain 的实现

基于以上信息，实现代码
