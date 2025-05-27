package logtransports

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/uclog"
)

const maxLogRecordPerTransferRecord = 1000

// EncodeLogForTransfer aggregates log messages for more efficient the wire transfer to avoid hitting limits
// on number of Kinesis record per second and to improve the read performance
// For now we unify up to 1001 (maxLogRecordPerKinesisRecord) log messages/events into a single LogRecordArray to be sent as
// a single Kinesis record. Each Kinesis record is limited to 1MB so we are staying well under that.
// Each LogRecordArray has same service/region/host/tenant and then a set of log message/event contents
func EncodeLogForTransfer(logRecords *logRecord, region region.MachineRegion, host string, service service.Service) []*uclog.LogRecordArray {
	recordsMap := make(map[uuid.UUID]*uclog.LogRecordArray)
	recordsReady := []*uclog.LogRecordArray{}

	currRecord := logRecords
	for currRecord != nil {
		message := currRecord.event.Message
		if message != "" {
			message = fmt.Sprintf("%v: %s", currRecord.event.RequestID, message)
		}
		lRC := uclog.LogRecordContent{Message: message, Code: currRecord.event.Code, Payload: currRecord.event.Payload,
			Timestamp: int(currRecord.timestamp.UnixMilli())}
		rm, ok := recordsMap[currRecord.event.TenantID]

		if ok && rm != nil {
			rm.Records = append(rm.Records, lRC)

			// Each kinesis record needs to be less than 1MB, so limit the size of a entry at 1000 log messages
			if len(rm.Records) > maxLogRecordPerTransferRecord {
				recordsReady = append(recordsReady, rm)
				recordsMap[currRecord.event.TenantID] = nil
			}
		} else {
			rm = &uclog.LogRecordArray{Service: service, TenantID: currRecord.event.TenantID, Region: region, Host: host,
				Records: []uclog.LogRecordContent{lRC}}
			recordsMap[currRecord.event.TenantID] = rm
		}
		currRecord = currRecord.next
	}

	// Add all the record which are not full, all the LogRecordArray with maxLogRecordPerKinesisRecord log events/messages were
	// appended inside the loop
	for _, rm := range recordsMap {
		if rm != nil {
			recordsReady = append(recordsReady, rm)
		}
	}
	return recordsReady
}

// initConfigInfoInTransports passes the config data to each transport
func initConfigInfoInTransports(name service.Service, machineName string, config *Config, tokenSource jsonclient.Option) []uclog.Transport {
	var transports []uclog.Transport = make([]uclog.Transport, 0, 4)

	for _, tr := range config.Transports {
		transports = append(transports, tr.GetTransport(name, tokenSource, machineName))
	}

	return transports
}
