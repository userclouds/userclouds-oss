package logtransports

// Transport directing event stream to a Kinesis data stream

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

func init() {
	registerDecoder(TransportTypeKinesis, func(value []byte) (TransportConfig, error) {
		var k KinesisTransportConfig
		// NB: we need to check the type here because the yaml decoder will happily decode an
		// empty struct, since dec.KnownFields(true) gets lost via the yaml.Unmarshaler
		// interface implementation
		if err := json.Unmarshal(value, &k); err == nil && k.Type == TransportTypeKinesis {
			return &k, nil
		}
		return nil, ucerr.New("Unknown transport type")
	})
}

// TransportTypeKinesis defines the Kinesis transport
const TransportTypeKinesis TransportType = "kinesis"

// KinesisTransportConfig defines logger client config
type KinesisTransportConfig struct {
	Type                  TransportType `yaml:"type" json:"type"`
	uclog.TransportConfig `yaml:"transportconfig" json:"transportconfig"`
	AwsRegion             string `yaml:"aws_region" json:"aws_region"`
	StreamName            string `yaml:"stream_name" json:"stream_name"`
	Filename              string `yaml:"filename" json:"filename"`
	ShardCount            int32  `yaml:"shard_count" json:"shard_count"`
}

// GetType implements TransportConfig
func (c KinesisTransportConfig) GetType() TransportType {
	return TransportTypeKinesis
}

// IsSingleton implements TransportConfig
func (c KinesisTransportConfig) IsSingleton() bool {
	return true
}

// GetTransport implements TransportConfig
func (c KinesisTransportConfig) GetTransport(name service.Service, _ jsonclient.Option, machineName string) uclog.Transport {
	return newTransportBackgroundIOWrapper(newKinesisTransport(&c, name, machineName))
}

// Validate implements Validateable
func (c *KinesisTransportConfig) Validate() error {
	// Skip validation for loggers that may not be included
	if !c.Required {
		return nil
	}

	if rg := region.FromAWSRegion(c.AwsRegion); !region.IsValid(rg, universe.Prod) {
		return ucerr.Errorf("Invalid region for Kinesis transport config: %v", c.AwsRegion)
	}
	if c.StreamName == "" {
		return ucerr.New("logging config invalid - missing stream name")
	}
	if c.ShardCount < 1 {
		return ucerr.New("logging config invalid - shard count must be greater than 0")
	}

	return nil
}

const (
	kinesisTransportName = "KinesisTransport"
	// Intervals for sending event data to the server
	kinesisSendInterval       time.Duration = 1 * time.Second
	maxKinesisRecordsPerBatch               = 5
)

type kinesisTransport struct {
	config           KinesisTransportConfig
	streamName       *string
	service          service.Service
	host             string
	region           region.MachineRegion
	kc               *kinesis.Client
	writeLocalStream bool
	filename         string
	fileHandle       *os.File
	fileWriter       *bufio.Writer
	failedAPICalls   int64
}

func newKinesisTransport(c *KinesisTransportConfig, name service.Service, machineName string) *kinesisTransport {
	return &kinesisTransport{config: *c, service: name, host: machineName, region: region.Current()}
}

func (t *kinesisTransport) init(ctx context.Context) (*uclog.TransportConfig, error) {
	c := &uclog.TransportConfig{Required: t.config.Required, MaxLogLevel: t.config.MaxLogLevel}
	if !service.IsValid(t.service) {
		return c, ucerr.New("Invalid service name")
	}
	cfg, err := ucaws.NewConfigWithRegion(ctx, t.config.AwsRegion)
	if err != nil {
		return c, ucerr.Wrap(err)
	}
	t.kc = kinesis.NewFromConfig(cfg)

	// TODO - the stream name has to contain the company_id, environment and the app
	t.streamName = &t.config.StreamName

	if err = t.initStream(ctx); err != nil {
		return c, ucerr.Wrap(err)
	}
	if err = t.writeTestRecord(ctx); err != nil {
		return c, ucerr.Wrap(err)
	}

	// If running in local mode - open a file where the records will be written
	if len(t.config.Filename) > 0 {
		t.writeLocalStream = true
		t.filename = t.config.Filename
		t.fileHandle, err = os.OpenFile(t.filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		t.fileWriter = bufio.NewWriter(t.fileHandle)
	}
	return c, ucerr.Wrap(err)
}

func (t *kinesisTransport) writeTestRecord(ctx context.Context) error {
	logRecord := &logRecord{
		time.Now().UTC(),
		uclog.LogEvent{
			Message:  fmt.Sprintf("%v: [D] Test write to kinesis", uuid.Nil),
			Code:     uclog.EventCodeNone,
			Count:    1,
			LogLevel: uclog.LogLevelDebug},
		nil,
	}
	recordsReady := EncodeLogForTransfer(logRecord, t.region, t.host, t.service)
	jsonVal, err := json.Marshal(recordsReady[0])
	if err != nil {
		return ucerr.Wrap(err)
	}
	testRecords := []types.PutRecordsRequestEntry{{
		Data:         jsonVal,
		PartitionKey: aws.String(time.Now().UTC().String()),
	}}
	_, err = t.kc.PutRecords(ctx, &kinesis.PutRecordsInput{StreamName: t.streamName, Records: testRecords})
	return ucerr.Wrap(err)
}

func (t *kinesisTransport) initStream(ctx context.Context) error {
	_, err := t.kc.DescribeStream(ctx, &kinesis.DescribeStreamInput{StreamName: t.streamName})
	// If the stream doesn't already exist create it
	var aerr *types.ResourceNotFoundException
	if errors.As(err, &aerr) {
		_, err = t.kc.CreateStream(ctx, &kinesis.CreateStreamInput{
			ShardCount: &t.config.ShardCount,
			StreamName: t.streamName,
		})
		if err != nil {
			return ucerr.Wrap(err)
		}
		// wait to make sure that the stream exists
		err = kinesis.NewStreamExistsWaiter(t.kc).Wait(ctx, &kinesis.DescribeStreamInput{StreamName: t.streamName}, 10*time.Minute)
	}
	return ucerr.Wrap(err)
}

func (t *kinesisTransport) writeMessages(ctx context.Context, logRecords *logRecord, startTime time.Time, count int) {
	var recordsToSend []types.PutRecordsRequestEntry

	// If local file is enabled we flush the writes every time even if the buffer is not full to make sure the file log
	// doesn't fall behind. The Writer will flush 4096 bytes chunks by default.
	if t.writeLocalStream {
		defer t.fileWriter.Flush()
	}

	recordsReady := EncodeLogForTransfer(logRecords, t.region, t.host, t.service)
	i := 0
	for i < len(recordsReady) {
		if len(recordsReady)-i < maxKinesisRecordsPerBatch {
			recordsToSend = make([]types.PutRecordsRequestEntry, len(recordsReady)-i)
		} else {
			recordsToSend = make([]types.PutRecordsRequestEntry, maxKinesisRecordsPerBatch)
		}
		batchTime := time.Now().UTC().String()
		for j := range recordsToSend {
			jsonVal, _ := json.Marshal(recordsReady[i+j])
			recordsToSend[j] = types.PutRecordsRequestEntry{
				Data:         jsonVal,
				PartitionKey: &batchTime, //TODO - this should be balanced across messages
			}

			// Write out the record to a local file if configured
			if t.writeLocalStream {
				t.fileWriter.Write(recordsToSend[j].Data)
			}
		}

		// Write a batch of records to Kinesis
		if _, err := t.kc.PutRecords(ctx, &kinesis.PutRecordsInput{StreamName: t.streamName, Records: recordsToSend}); err != nil {
			failedCalls.WithLabelValues(string(TransportTypeKinesis), "put_records").Inc()
			t.failedAPICalls++ // Reports back to stats endpoint so we can detect w/ monitoring tools
			return
		}
		successfulCalls.WithLabelValues(string(TransportTypeKinesis)).Inc()
		i += len(recordsToSend)
	}
}
func (t *kinesisTransport) getFailedAPICallsCount() int64 {
	return t.failedAPICalls
}

func (t *kinesisTransport) getIOInterval() time.Duration {
	return kinesisSendInterval
}

func (t *kinesisTransport) getMaxLogLevel() uclog.LogLevel {
	return t.config.MaxLogLevel
}

func (t *kinesisTransport) supportsCounters() bool {
	return true
}

func (t *kinesisTransport) getTransportName() string {
	return kinesisTransportName
}

func (t *kinesisTransport) flushIOResources() {
	if t.writeLocalStream {
		t.fileHandle.Sync()
	}
}

func (t *kinesisTransport) closeIOResources() {
	if t.writeLocalStream {
		_ = t.fileHandle.Close()
	}
}
