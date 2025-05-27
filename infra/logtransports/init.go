package logtransports

import (
	"userclouds.com/infra/uclog"
)

// initializes basic to-screen logging until logger config can be read
func init() {
	recoverPanic = true

	loggerConfig := Config{}
	loggerConfig.Transports = []TransportConfig{
		&GoTransportConfig{
			Type: TransportTypeGo,
			TransportConfig: uclog.TransportConfig{
				Required:    true,
				MaxLogLevel: uclog.LogLevelDebug,
			},
			PrefixFlag:    DefaultPrefixVal,
			SupportsColor: false,
		},
	}
	loggerConfig.NoRequestIDs = true

	transports := initConfigInfoInTransports("", "", &loggerConfig, nil)

	uclog.PreInit(transports)
}
