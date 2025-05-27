package logtransports

import (
	"encoding/json"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Config defines overall logging configuration
type Config struct {
	Transports   TransportConfigs `yaml:"transports" json:"transports"`
	NoRequestIDs bool             `yaml:"no_request_ids" json:"no_request_ids"`
}

//go:generate genvalidate Config

func (c Config) extraValidate() error {
	uv := universe.Current()
	if len(c.Transports) == 0 && (uv.IsCloud() || uv.IsDev()) {
		return ucerr.Errorf("No log transport configured")
	}
	return nil
}

// TransportConfigs is an alias for an array of TransportConfig so we can handle polymorphic config unmarshalling
type TransportConfigs []TransportConfig

// UnmarshalJSON implements json.Unmarshaler
func (tcs *TransportConfigs) UnmarshalJSON(data []byte) error {
	var c []intermediateConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return ucerr.Wrap(err)
	}

	// init if we're nil
	if tcs == nil {
		*tcs = make([]TransportConfig, 0, len(c))
	}

	// use append here to allow us to merge multiple transports across multiple files
	// see config_test.go:MergeTest
	// We also want one of each transport type, so we'll overwrite any existing transports configs with the same type
	for _, v := range c {
		if !v.c.IsSingleton() {
			*tcs = append(*tcs, v.c)
			continue
		}
		if existing := tcs.getIndexForTransportType(v.c.GetType()); existing == -1 {
			*tcs = append(*tcs, v.c)
		} else {
			(*tcs)[existing] = v.c
		}
	}
	return nil
}

func (tcs *TransportConfigs) getIndexForTransportType(tt TransportType) int {
	for i, v := range *tcs {
		if v.GetType() == tt {
			return i
		}
	}
	return -1
}

// intermediateConfig is a place to unmarshal to before we know the type of transport
type intermediateConfig struct {
	c TransportConfig
}

// UnmarshalJSON implements json.Unmarshaler
func (i *intermediateConfig) UnmarshalJSON(value []byte) error {
	for _, d := range decoders {
		if c, err := d(value); err == nil {
			i.c = c
			return nil
		}
	}
	return ucerr.New("unknown TransportConfig implementation")
}

// decoders allows different files to register themselves as available decoders/types
// so that we can ship some transports externally and leave others internal without causing
// build issues
var decoders = make(map[TransportType]func([]byte) (TransportConfig, error))

// registerDecoder centralizes manipulation of `decoders“
func registerDecoder(name TransportType, f func([]byte) (TransportConfig, error)) {
	decoders[name] = f
}

// TransportConfig defines the interface for a transport config
type TransportConfig interface {
	GetTransport(service.Service, jsonclient.Option, string) uclog.Transport
	GetType() TransportType
	IsSingleton() bool
	Validate() error
}

// TransportType defines the type of transport
type TransportType string
