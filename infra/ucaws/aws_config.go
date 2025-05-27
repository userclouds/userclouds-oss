package ucaws

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	smithy "github.com/aws/smithy-go/middleware"
	prometheusv2 "github.com/jonathan-innis/aws-sdk-go-prometheus/v2"
	awsmetrics "github.com/jonathan-innis/aws-sdk-go-prometheus/v2/awsmetrics/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

// Note: We can't use prometheusv2.WithPrometheusMetrics since it call the logic to register the metrics every time.
// so every time we call NewConfigWithRegion we will register the metrics again, which will cause a panic.
// So we do this instead
var awsPrometheusPublisher = prometheusv2.NewPrometheusPublisher(prometheus.DefaultRegisterer)

const (
	// DefaultRegion is the default region for AWS services
	// We use us-west-2 as the default region, this is mostly for AWS secrets manager, SQS, AWS SES, etc..
	// In cases where the the code that uses AWS API cares about the region, `NewConfigWithRegion` should be used.
	DefaultRegion = "us-west-2"
)

// NewConfigWithDefaultRegion creates an AWS Config Object that can be used to create clients for AWS services for the default region (us-west-2)
func NewConfigWithDefaultRegion(ctx context.Context) (aws.Config, error) {
	return NewConfigWithRegion(ctx, DefaultRegion)
}

// NewConfigForProfileWithDefaultRegion creates an AWS Config Object that can be used to create clients for AWS services used for when running locally on a dev machine
func NewConfigForProfileWithDefaultRegion(ctx context.Context, uv universe.Universe) (aws.Config, error) {
	return NewConfigForProfile(ctx, DefaultRegion, uv)
}

// NewConfigForProfile creates an AWS Config Object that can be used to create clients for AWS services used for when running locally on a dev machine
func NewConfigForProfile(ctx context.Context, region string, uv universe.Universe) (aws.Config, error) {
	if !uv.IsCloud() {
		return aws.Config{}, ucerr.Errorf("universe '%v' is not cloud, can't use w/ AWS profile", uv)
	}
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#loading-aws-shared-configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region), config.WithSharedConfigProfile(string(uv)))
	if err != nil {
		// AWS API calls expect an aws Config object (and not a pointer to one), so we return an empty one here
		// instead of of changing the method signature to return a pointer to an aws.Config object and then dereferencing it on every AWS API call
		return aws.Config{}, ucerr.Wrap(err)
	}
	// We are not creating the metric middleware here since this code path is typically used for tools running locally
	return cfg, nil
}

// NewConfigWithRegion creates an AWS Config Object that can be used to create clients for AWS services for a specific region
func NewConfigWithRegion(ctx context.Context, region string) (aws.Config, error) {
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#loading-aws-shared-configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		// AWS API calls expect an aws Config object (and not a pointer to one), so we return an empty one here
		// instead of of changing the method signature to return a pointer to an aws.Config object and then dereferencing it on every AWS API call
		return aws.Config{}, ucerr.Wrap(err)
	}
	// until https://github.com/aws/aws-sdk-go-v2/issues/1744 is implemented.
	cfg.APIOptions = append(cfg.APIOptions, func(s *smithy.Stack) error {
		return ucerr.Wrap(awsmetrics.WithMetricMiddlewares(awsPrometheusPublisher, http.DefaultClient)(s))
	})
	return cfg, nil
}

// NewFromConfig creates an AWS Config Object that can be used to create clients for AWS services for a specific region
func NewFromConfig(ctx context.Context, cfg Config) (aws.Config, error) {
	if !cfg.UseAccessKey() {
		return NewConfigWithRegion(ctx, cfg.Region)
	}
	cc := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret, ""))
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region), config.WithCredentialsProvider(cc))
	if err != nil {
		// AWS API calls expect an aws Config object (and not a pointer to one), so we return an empty one here
		// instead of of changing the method signature to return a pointer to an aws.Config object and then dereferencing it on every AWS API call
		return aws.Config{}, ucerr.Wrap(err)
	}
	return awsCfg, nil
}
