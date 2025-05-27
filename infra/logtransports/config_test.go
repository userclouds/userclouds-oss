package logtransports

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/yamlconfig"
)

func TestMerge(t *testing.T) {
	ctx := context.Background()
	var lc Config
	err := yamlconfig.LoadEnv(ctx, "testdata", &lc, yamlconfig.LoadParams{Universe: universe.Test, BaseDirs: []string{""}})
	assert.NoErr(t, err)

	assert.Equal(t, len(lc.Transports), 4)
	assert.Equal(t, lc.Transports[0].GetType(), TransportTypeGoLogJSON)
	assert.Equal(t, lc.Transports[1].GetType(), TransportTypeKinesis)
	assert.Equal(t, lc.Transports[2].GetType(), TransportTypeFile)
	assert.Equal(t, lc.Transports[3].GetType(), TransportTypeFile)
	ts1 := lc.Transports[1].(*KinesisTransportConfig)
	assert.Equal(t, ts1.StreamName, "newman")
	ts2 := lc.Transports[2].(*FileTransportConfig)
	assert.Equal(t, ts2.Filename, "foo")
	ts3 := lc.Transports[3].(*FileTransportConfig)
	assert.Equal(t, ts3.Filename, "bar")
}
