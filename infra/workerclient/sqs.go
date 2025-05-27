package workerclient

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/worker"
)

// TypeSQS defines the SQS client type
const TypeSQS Type = "sqs"

// SQSClient talks to AWS SQS
type SQSClient struct {
	queueURL string
	client   *sqs.Client
}

// GetSQSClient returns a new SQS client
func GetSQSClient(ctx context.Context) (*sqs.Client, error) {
	cfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return sqs.NewFromConfig(cfg), nil
}

// GetSQSClientForTool returns a new SQS client to use with a tool running locally on a dev machine
func GetSQSClientForTool(ctx context.Context) (*sqs.Client, error) {
	cfg, err := ucaws.NewConfigForProfileWithDefaultRegion(ctx, universe.Current())
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return sqs.NewFromConfig(cfg), nil
}
func newSQSClient(ctx context.Context, cfg *Config) (*SQSClient, error) {
	queueURL := cfg.GetURL()
	awsCfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &SQSClient{queueURL: queueURL, client: sqs.NewFromConfig(awsCfg)}, nil
}

// NewSQSWorkerClientForTool returns a new SQSClient for a given sqs url to be used with a tool running locally on a dev machine
func NewSQSWorkerClientForTool(ctx context.Context, queueURL string) (*SQSClient, error) {
	cfg, err := ucaws.NewConfigForProfileWithDefaultRegion(ctx, universe.Current())
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &SQSClient{queueURL: queueURL, client: sqs.NewFromConfig(cfg)}, nil
}

// Send implements Client
func (c *SQSClient) Send(ctx context.Context, msg worker.Message) error {
	msg.SetSourceRegionIfNotSet()
	if err := msg.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if _, err := c.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &c.queueURL,
		MessageBody: aws.String(string(body)), // required by validation, but unused?
	}); err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "sent message %v to %s", msg.Task, c.queueURL)
	return nil
}

func (c SQSClient) String() string {
	return c.queueURL
}
