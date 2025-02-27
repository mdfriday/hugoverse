package loggers

var global Logger
var fields *LogFields

func SetGlobal(logger Logger) {
	global = logger
}

func SetGlobalFields(fs *LogFields) {
	fields = fs
}

func GetGlobalFields() *LogFields {
	return fields
}
