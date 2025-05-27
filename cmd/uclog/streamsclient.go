package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/logserver/config"
)

type streamDataHandler func(ctx context.Context, t *time.Time, r []byte, record *uclog.LogRecordArray) error
type streamsClient struct {
	streamName string
	kc         *kinesis.Client
}

func newStreamClient(ctx context.Context, streamName string, tenantID uuid.UUID, region string, service service.Service) (*streamsClient, error) {
	o := config.NewAdvanceAnalyticsResourceNames(tenantID, region, service, config.AWSDefaultOrg)

	// Override the stream name if requested
	if streamName != "" {
		o.StreamName = streamName
	}
	// First try to get the universe from the stream name (debug/staging/prod), if that doesn't work (because the stream in is xxx-log or an SDK stream) then use the current universe
	uv := universe.Universe(streamName)
	if !uv.IsCloud() {
		uv = universe.Current()
	}
	if !uv.IsCloud() {
		return nil, ucerr.Errorf("Can't infer AWS Account from stream name '%v', please set %s explicitly", streamName, universe.EnvKeyUniverse)
	}
	uclog.Infof(ctx, "Using universe %s for stream %s", uv, streamName)
	cfg, err := ucaws.NewConfigForProfileWithDefaultRegion(ctx, uv)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &streamsClient{
		streamName: o.StreamName,
		kc:         kinesis.NewFromConfig(cfg),
	}, nil
}

func (sc *streamsClient) getShardPositions(ctx context.Context, startTime, endTime time.Time, timeWindowEnd int) (map[string]string, *types.StreamDescription, error) {
	var shardPosition = map[string]string{}
	stream, err := sc.kc.DescribeStream(ctx, &kinesis.DescribeStreamInput{StreamName: &sc.streamName})
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	for _, shard := range stream.StreamDescription.Shards {
		uclog.Debugf(ctx, "Processing shard %s from %s ", *shard.ShardId, startTime.Format("Jan _2 15:04:05:00"))
		var pos string
		if pos, err = sc.processShard(ctx, *shard.ShardId, startTime, endTime, timeWindowEnd != 0, ""); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
		shardPosition[*shard.ShardId] = pos
	}
	return shardPosition, stream.StreamDescription, nil
}

func (sc *streamsClient) processShard(ctx context.Context, shardID string,
	shardTimeStamp time.Time, endTimeStamp time.Time, endTimeProvided bool, shardPosition string) (string, error) {
	var currentShardPosition = &shardPosition

	// Verify that the stream passed in exists and is valid
	_, err := sc.kc.DescribeStream(ctx, &kinesis.DescribeStreamInput{StreamName: &sc.streamName})

	if err != nil {
		return *currentShardPosition, ucerr.Wrap(err)
	}

	// Retrieve iterator to the stream that points at the position from which we want to start processing
	var iteratorOutput *kinesis.GetShardIteratorOutput
	if *currentShardPosition != "" {
		// If position was provided start at the next record in the stream
		iteratorOutput, err = sc.kc.GetShardIterator(ctx, &kinesis.GetShardIteratorInput{
			ShardId:                &shardID,
			ShardIteratorType:      types.ShardIteratorTypeAfterSequenceNumber,
			StartingSequenceNumber: currentShardPosition,
			StreamName:             &sc.streamName,
		})
	} else if !shardTimeStamp.IsZero() {
		// If time was provided start closest to that timestamp
		iteratorOutput, err = sc.kc.GetShardIterator(ctx, &kinesis.GetShardIteratorInput{
			ShardId:           &shardID,
			ShardIteratorType: types.ShardIteratorTypeAtTimestamp,
			Timestamp:         aws.Time(shardTimeStamp),
			StreamName:        &sc.streamName,
		})
	} else {
		// If a position or time from which to process up wasn't provided - start from the oldest record in the shard
		iteratorOutput, err = sc.kc.GetShardIterator(ctx, &kinesis.GetShardIteratorInput{
			ShardId:           &shardID,
			ShardIteratorType: types.ShardIteratorTypeTrimHorizon,
			StreamName:        &sc.streamName,
		})
	}

	if err != nil {
		return *currentShardPosition, ucerr.Wrap(err)
	}

	// Iterate through the stream and process each record with content from starting point till tip
	shardIterator := iteratorOutput.ShardIterator
	var caughtUp bool
	recordsRead := 0
	for !caughtUp {
		records, err := sc.kc.GetRecords(ctx, &kinesis.GetRecordsInput{
			ShardIterator: shardIterator,
		})
		if err != nil {
			return *currentShardPosition, ucerr.Wrap(err)
		}

		for _, r := range records.Records {
			err = unmarshalAndProcess(ctx, r.ApproximateArrivalTimestamp, r.Data, nil)
			// Abort if we couldn't process the record correctly
			if err != nil {
				return *currentShardPosition, ucerr.Wrap(err)
			}
			currentShardPosition = r.SequenceNumber
			recordsRead++
		}
		// Advance to the next record in the stream
		shardIterator = records.NextShardIterator
		// Check if we got to the tip of the stream
		if *records.MillisBehindLatest == 0 || records.NextShardIterator == nil {
			caughtUp = true
		}
		// Check if got to the end of requested window
		if endTimeProvided {
			if len(records.Records) != 0 && records.Records[len(records.Records)-1].ApproximateArrivalTimestamp.After(endTimeStamp) {
				caughtUp = true
			}
		}
	}
	uclog.Debugf(ctx, "Read %d records from shard %s", recordsRead, shardID)
	return *currentShardPosition, nil
}

func (sc *streamsClient) deregister(ctx context.Context, consumerName, consumerARN, streamARN *string) {
	input := kinesis.DeregisterStreamConsumerInput{ConsumerARN: consumerARN, ConsumerName: consumerName, StreamARN: streamARN}
	if _, err := sc.kc.DeregisterStreamConsumer(ctx, &input); err != nil {
		uclog.Errorf(ctx, "DeregisterStreamConsumer: %v", err)
	}
}

func (sc *streamsClient) startLive(ctx context.Context, streamDescription *types.StreamDescription, startTime time.Time, shardPosition map[string]string, handlerFunc streamDataHandler) ([]chan bool, error) {
	// Create one time consumer name for subscribing to the kinesis stream
	var consumerName string = uuid.Must(uuid.NewV4()).String()
	// Register the consumer
	cr, err := sc.kc.RegisterStreamConsumer(ctx, &kinesis.RegisterStreamConsumerInput{ConsumerName: &consumerName, StreamARN: streamDescription.StreamARN})
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer sc.deregister(ctx, &consumerName, cr.Consumer.ConsumerARN, streamDescription.StreamARN)
	// We can't subscribe to the stream until consumer registration is active
	var waitForCreation = true
	for waitForCreation {
		ocr, err := sc.kc.DescribeStreamConsumer(ctx, &kinesis.DescribeStreamConsumerInput{ConsumerName: &consumerName, StreamARN: streamDescription.StreamARN})
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if ocr.ConsumerDescription.ConsumerStatus == types.ConsumerStatusActive {
			waitForCreation = false
		} else {
			time.Sleep(time.Second)
		}
	}
	doneChannels := make([]chan bool, 0, len(streamDescription.Shards))
	for _, shard := range streamDescription.Shards {
		var sp types.StartingPosition
		if shardPosition[*shard.ShardId] != "" {
			uclog.Debugf(ctx, "Signing up for events from shard %s from %s ", *shard.ShardId, startTime.Format("Jan _2 15:04:05:00"))
			sp = types.StartingPosition{SequenceNumber: aws.String((shardPosition[*shard.ShardId])), Type: types.ShardIteratorTypeAfterSequenceNumber}
		} else {
			uclog.Debugf(ctx, "Signing up for events from shard %s from latest/now", *shard.ShardId)
			sp = types.StartingPosition{Type: types.ShardIteratorTypeLatest}
		}
		so, err := sc.kc.SubscribeToShard(ctx, &kinesis.SubscribeToShardInput{ConsumerARN: cr.Consumer.ConsumerARN, ShardId: shard.ShardId, StartingPosition: &sp})
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		done := make(chan bool)
		doneChannels = append(doneChannels, done)
		go sc.processData(ctx, handlerFunc, so.GetStream(), cr.Consumer.ConsumerARN, shard.ShardId, shardPosition, done)
	}
	return doneChannels, nil
}

func (sc *streamsClient) processData(ctx context.Context, handlerFunc streamDataHandler, sm *kinesis.SubscribeToShardEventStream, consumerARN, shardID *string, shardPosition map[string]string, done chan bool) {
	eventsC := sm.Events()
	resubscribeTicker := *time.NewTicker(resubscribeTickerInterval)
	for {
		select {
		case e := <-eventsC:
			es, ok := e.(*types.SubscribeToShardEventStreamMemberSubscribeToShardEvent)
			if ok {
				writeMutex.Lock()
				for _, rec := range es.Value.Records {
					// If we couldn't process the record correctly skip it vs aborting the subscription
					if err := handlerFunc(ctx, rec.ApproximateArrivalTimestamp, rec.Data, nil); err != nil {
						uclog.Errorf(ctx, "Failed to process %v", rec)
					}
				}
				shardPosition[*shardID] = *(es.Value.ContinuationSequenceNumber)
				writeMutex.Unlock()
			}
		case <-resubscribeTicker.C:
			// Resubscribe the last processed spot
			writeMutex.Lock()
			sp := &types.StartingPosition{SequenceNumber: aws.String((shardPosition[*shardID])), Type: types.ShardIteratorTypeAfterSequenceNumber}
			so, err := sc.kc.SubscribeToShard(ctx, &kinesis.SubscribeToShardInput{ConsumerARN: consumerARN, ShardId: shardID, StartingPosition: sp})
			if err != nil {
				// TODO - we hit this on auth expiring or network failures over multiday runtimes. Need to add re-auth and retry.
				uclog.Infof(ctx, "Failed to resubscribe to %s due to error %v", *shardID, err)
			} else {
				// Switch channels
				eventsC = so.GetStream().Events()
			}
			writeMutex.Unlock()
		case <-done:
			uclog.Infof(ctx, "Exiting thread")
			return
		}
	}
}

// DeregisterConsumer deregisters all stale Kinesis consumers
func (sc *streamsClient) deregisterConsumer(ctx context.Context) error {
	// Get the shards of the stream
	stream, err := sc.kc.DescribeStream(ctx, &kinesis.DescribeStreamInput{StreamName: &sc.streamName})
	if err != nil {
		uclog.Errorf(ctx, "Could not access the kinesis stream")
		return ucerr.Wrap(err)
	}

	// Get list of consumers
	ls, err := sc.kc.ListStreamConsumers(ctx, &kinesis.ListStreamConsumersInput{StreamARN: stream.StreamDescription.StreamARN})
	if err != nil {
		uclog.Errorf(ctx, "Could not get list of consumers")
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "Found %d registered consumers", len(ls.Consumers))

	cutOffTime := time.Now().UTC().Add(time.Duration(-int64(24) * int64(time.Hour)))
	for _, l := range ls.Consumers {
		if l.ConsumerCreationTimestamp.Before(cutOffTime) {
			if _, err := sc.kc.DeregisterStreamConsumer(ctx,
				&kinesis.DeregisterStreamConsumerInput{
					ConsumerARN:  l.ConsumerARN,
					ConsumerName: aws.String(*l.ConsumerName),
					StreamARN:    stream.StreamDescription.StreamARN,
				}); err != nil {
				uclog.Errorf(ctx, "Error DeregisterStreamConsumer %s : %v", *l.ConsumerName, err)
			} else {
				uclog.Infof(ctx, "Successfully  deregistered %s", *l.ConsumerName)
			}
		}
	}
	return nil
}
