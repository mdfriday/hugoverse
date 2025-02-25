你现在是一名高级golang开发人员，要实现根据当前环境，如 prod, dev来动态设置logger级别。

以下是具体实现步骤：

1. 先从命令行获取环境参数 env, 默认值是prod，用户可以设置成dev, -env dev
2. application 里的 generate prompt 方法，得到这个参数，并创建相应级别的 logger 实例. prod只打印error级错误信息, dev则可打出info级
3. 所有日志将被存入本项目的.aupro/log目录下，每一次生成prompt都会创建一个日志文件
4. 可参考以下函数来实现

```go
// setupLogger configures the logger to output to .mdfriday directory
func setupLogger() (loggers.Logger, error) {
// Get project root directory (assuming we're in internal/domain/host/entity)
projectRoot, err := filepath.Abs("../../../../")
if err != nil {
return nil, errors.Wrap(err, "failed to get project root directory")
}

	// Create .mdfriday directory in project root if it doesn't exist
	logDir := filepath.Join(projectRoot, ".mdfriday")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to create log directory")
	}

	// Create log file with timestamp
	logFile := filepath.Join(logDir, fmt.Sprintf("scp_%s.log", time.Now().Format("20060102_150405")))
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create log file")
	}

	// Create logger options
	opts := loggers.Options{
		Level:         logg.LevelDebug,
		Stdout:        f, // 只输出到文件
		Stderr:        f, // 错误也输出到文件
		StoreErrors:   true,
		DistinctLevel: logg.LevelWarn, // Drop duplicate warnings and errors
	}

	fmt.Printf("Log file %p created at: %s\n", f, logFile)
	return loggers.New(opts), nil
}
```