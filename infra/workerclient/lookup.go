package workerclient

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

// GetSQSUrlForUniverse returns the SQS queue URL for the given universe
func GetSQSUrlForUniverse(ctx context.Context, uv universe.Universe) (string, error) {
	if !uv.IsCloud() {
		return "", ucerr.Errorf("universe is not cloud")
	}
	client, err := GetSQSClientForTool(ctx)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	prefix := string(uv)
	output, err := client.ListQueues(ctx, &sqs.ListQueuesInput{QueueNamePrefix: &prefix})
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	if output.NextToken != nil {
		return "", ucerr.New("Pagination not supported on ListQueues")
	}
	allQueueURLs := make([]string, 0)
	for _, queueURL := range output.QueueUrls {
		if strings.Contains(queueURL, "dead-letter") {
			continue
		}
		allQueueURLs = append(allQueueURLs, queueURL)
	}
	if len(allQueueURLs) == 0 {
		return "", ucerr.New("No queues found")
	} else if len(allQueueURLs) > 1 {
		return "", ucerr.Errorf("Multiple queues found: %v", allQueueURLs)
	}
	return allQueueURLs[0], nil
}
