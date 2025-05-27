package kinesisbackend

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/logserver/config"
)

// GC interval
const (
	gcInterval time.Duration = 10 * time.Minute
)

type kinesisStreamRecord struct {
	streamName *string
	kc         *kinesis.Client
	lastUsed   time.Time
}

type kinesisStreamKey struct {
	tenantID uuid.UUID
	service  service.Service
}

// KinesisConnections exposes a backend for writing to a collection of Kinesis streams
type KinesisConnections struct {
	connections map[kinesisStreamKey]kinesisStreamRecord
	regionName  string
	cfg         aws.Config
	mutex       sync.Mutex
	gcTicker    time.Ticker
	done        chan bool
}

// NewKinesisConnections returns an instance of the kinesis backend
func NewKinesisConnections(ctx context.Context, awsRegion string) (*KinesisConnections, error) {
	var c KinesisConnections
	// Create a cache for storing connections
	c.connections = make(map[kinesisStreamKey]kinesisStreamRecord)
	// Create mutex to protect read and write access to the map since it is not thread safe
	c.mutex = sync.Mutex{}

	var err error
	c.cfg, err = ucaws.NewConfigWithRegion(ctx, awsRegion)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Initialize a GC timer to collect connections every X minutes
	c.gcTicker = *time.NewTicker(gcInterval)
	c.done = make(chan bool)
	go func() {
		for {
			select {
			case <-c.done:
				return
			case <-c.gcTicker.C:
				c.prune()
			}
		}
	}()
	return &c, nil
}

// Write function writes out a record to the appropriate kinesis stream
func (c *KinesisConnections) Write(ctx context.Context, tenantID uuid.UUID, service service.Service, records [][]byte) error {
	key := kinesisStreamKey{tenantID: tenantID, service: service}
	var r kinesisStreamRecord
	var ok bool
	var err error

	// Look up if we have a cached connection
	c.mutex.Lock()
	r, ok = c.connections[key]
	c.mutex.Unlock()
	// If we don't have one create it. Note that because the lock is released in rare cases multiple instance of
	// kinesisStreamRecord will be created and all but the last one will be GCed
	if !ok {
		o := config.NewAdvanceAnalyticsResourceNames(tenantID, c.regionName, service, config.AWSDefaultOrg)
		r, err = newKinesisStreamRecord(ctx, c.cfg, o.StreamName)
		if err != nil {
			return ucerr.Wrap(err)
		}
		c.mutex.Lock()
		c.connections[key] = r
		c.mutex.Unlock()
	}
	return ucerr.Wrap(r.writeRecords(ctx, records))
}

func (c *KinesisConnections) prune() {
	t := time.Now().UTC()
	for key, record := range c.connections {
		if t.Sub(record.lastUsed) > gcInterval {
			c.mutex.Lock()
			delete(c.connections, key)
			c.mutex.Unlock()
		}
	}
}

func newKinesisStreamRecord(ctx context.Context, cfg aws.Config, streamName string) (kinesisStreamRecord, error) {
	// Create a connection to the stream
	var r kinesisStreamRecord
	r.kc = kinesis.NewFromConfig(cfg)
	r.streamName = &streamName
	// Update last used time so record doesn't get GCed before it is used
	r.lastUsed = time.Now().UTC()
	// Validate that the stream exists and can receive data
	_, err := r.kc.DescribeStream(ctx, &kinesis.DescribeStreamInput{StreamName: r.streamName})

	var aerr *types.ResourceNotFoundException
	if errors.As(err, &aerr) {
		_, err = r.kc.CreateStream(ctx, &kinesis.CreateStreamInput{
			ShardCount: aws.Int32(1),
			StreamName: r.streamName,
		})
		if err != nil {
			return r, ucerr.Wrap(err)
		}
		// wait to make sure that the stream exists
		err = kinesis.NewStreamExistsWaiter(r.kc).Wait(ctx, &kinesis.DescribeStreamInput{StreamName: r.streamName}, 30*time.Second)
	}
	return r, ucerr.Wrap(err)
}

func (r *kinesisStreamRecord) writeRecords(ctx context.Context, messages [][]byte) error {
	// Update last used time
	r.lastUsed = time.Now().UTC()

	// Wrap the incoming messages in the kinesis data structure
	var records []types.PutRecordsRequestEntry = make([]types.PutRecordsRequestEntry, 0, len(messages))
	for _, message := range messages {
		records = append(records, types.PutRecordsRequestEntry{Data: message, PartitionKey: aws.String(r.lastUsed.String())})
	}

	// Write the records to the kinesis stream
	_, err := r.kc.PutRecords(ctx, &kinesis.PutRecordsInput{
		Records:    records,
		StreamName: r.streamName,
	})
	return ucerr.Wrap(err)
}
