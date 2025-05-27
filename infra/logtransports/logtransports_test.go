package logtransports

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/uclog"
)

func TestClient(t *testing.T) {

	t.Run("EmptyEncode", func(t *testing.T) {
		r := EncodeLogForTransfer(nil, region.MachineRegionsForUniverse(universe.Prod)[0], "host", service.Console)
		assert.Equal(t, len(r), 0)
	})

	t.Run("MultiItemSingleTenantSmallEncode", func(t *testing.T) {

		var logRecords *logRecord
		tenantID := uuid.Must(uuid.NewV4())
		for range 10 {
			logRecord := logRecord{time.Now().UTC(), uclog.LogEvent{TenantID: tenantID, Message: tenantID.String()}, logRecords}
			logRecords = &logRecord
		}

		r := EncodeLogForTransfer(logRecords, region.MachineRegionsForUniverse(universe.Prod)[0], "host", service.Console)
		assert.Equal(t, len(r), 1)
		assert.Equal(t, len(r[0].Records), 10)
	})

	t.Run("MultiItemSingleTenantLargeEncode", func(t *testing.T) {

		var logRecords *logRecord

		tenantID := uuid.Must(uuid.NewV4())
		for range 3000 {
			logRecord := logRecord{time.Now().UTC(), uclog.LogEvent{TenantID: tenantID, Message: tenantID.String()}, logRecords}
			logRecords = &logRecord
		}

		r := EncodeLogForTransfer(logRecords, region.MachineRegionsForUniverse(universe.Prod)[0], "host", service.Console)
		assert.Equal(t, len(r), 3)
		assert.Equal(t, len(r[0].Records), 1001)
		assert.Equal(t, len(r[1].Records), 1001)
		assert.Equal(t, len(r[2].Records), 998)
	})

	t.Run("MultiItemMultiTenantSmallEncode", func(t *testing.T) {

		var logRecords *logRecord

		tenants := make([]uuid.UUID, 0, 10)
		for j := range tenants {
			tenants[j] = uuid.Must(uuid.NewV4())
			for range 10 {
				logRecord := logRecord{time.Now().UTC(), uclog.LogEvent{TenantID: tenants[j], Message: tenants[j].String()}, logRecords}
				logRecords = &logRecord
			}
		}

		r := EncodeLogForTransfer(logRecords, region.MachineRegionsForUniverse(universe.Prod)[0], "host", service.Console)
		assert.Equal(t, len(r), len(tenants))
		for i := range r {
			assert.Equal(t, len(r[i].Records), 10)
			for j := range r[i].Records {
				assert.Equal(t, tenants[i].String(), r[i].Records[j].Message)
			}

		}
	})

	t.Run("MultiItemMultiTenantLargeEncode", func(t *testing.T) {

		var logRecords *logRecord

		tenants := make([]uuid.UUID, 0, 10)
		for j := range tenants {
			tenants[j] = uuid.Must(uuid.NewV4())
			for range 2000 {
				logRecord := logRecord{time.Now().UTC(), uclog.LogEvent{TenantID: tenants[j], Message: tenants[j].String()}, logRecords}
				logRecords = &logRecord
			}
		}

		r := EncodeLogForTransfer(logRecords, region.MachineRegionsForUniverse(universe.Prod)[0], "host", service.Console)
		assert.Equal(t, len(r), len(tenants)*2)
		for i := range r {
			assert.Equal(t, len(r[i].Records) == 1001 || len(r[i].Records) == 999, true)
			assert.Equal(t, len(r[i+1].Records), 999)

			for j := range r[i].Records {
				assert.Equal(t, tenants[i%10].String(), r[i].Records[j].Message)
			}

		}
	})

}
