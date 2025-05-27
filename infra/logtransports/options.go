package logtransports

// ToolLogOption defines a way to pass optional configuration parameters.
type ToolLogOption interface {
	apply(*ToolLogConfig)
}

type optFunc func(*ToolLogConfig)

func (o optFunc) apply(po *ToolLogConfig) {
	o(po)
}

// Prefix allows specification of the prefix to be used by the logger on the screen
func Prefix(prefix int) ToolLogOption {
	return optFunc(func(po *ToolLogConfig) {
		po.prefix = prefix
	})
}

// NoPrefix specifies that there should be no prefix used by the logger on the screen
func NoPrefix() ToolLogOption {
	return optFunc(func(po *ToolLogConfig) {
		po.prefix = NoPrefixVal
	})
}

// Filename allows specification of the filename to be used by the file logger
func Filename(filename string) ToolLogOption {
	return optFunc(func(po *ToolLogConfig) {
		po.filename = filename
	})
}

// SupportsColor specifies that terminal supports color
func SupportsColor() ToolLogOption {
	return optFunc(func(po *ToolLogConfig) {
		po.supportsColor = true
	})
}

// UseJSONLog specifies that the user should be logged in JSON format
func UseJSONLog() ToolLogOption {
	return optFunc(func(po *ToolLogConfig) {
		po.useJSONLog = true
	})
}

// ToolLogConfig describes optional parameters for configuring logging for a tool
type ToolLogConfig struct {
	filename      string
	prefix        int
	supportsColor bool
	useJSONLog    bool
}
