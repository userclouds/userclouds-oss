package secret

// Derived from AWS sample code
// To insert secrets into secret manager, you can either use the AWS Console or the AWS CLI
// If you need more information about configurations or implementing the sample code, visit the AWS docs:
// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/setting-up.html

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	// PrefixAWS tells secret that this string is in fact resolvable with AWS secret manager
	// as opposed to other systems in the future, or just plaintext (for eg. dev)
	// TODO: config linter in the future that ensures all secret.* fields are prefixed in prod configs?
	PrefixAWS Prefix = "aws://secrets/"

	secretRecoveryWindowInDays = 7
)

// AWS insists on returning a JSON blob of key/value instead of just a string
// if/when we implement eg. GCP's equivalent, or Hashicorp, then some of this
// could get moved over to aws.go and UnmarshalYAML could have a "provider" interface
type awsSecret struct {
	String string `json:"string" yaml:"string"`
}

type awsSecretsProvider interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
	DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
}

type testAWSSecretsProvider struct {
}

func (p *testAWSSecretsProvider) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(`{"string":"testsecret"}`),
	}, nil
}

func (p *testAWSSecretsProvider) CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	return nil, nil
}

func (p *testAWSSecretsProvider) UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	return nil, nil
}

func (p *testAWSSecretsProvider) DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
	return nil, nil
}

func getClient(ctx context.Context) (awsSecretsProvider, string, error) {
	if universe.Current().IsTestOrCI() {
		return &testAWSSecretsProvider{}, "", nil
	}

	cfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return nil, "", ucerr.Wrap(err)
	}

	return secretsmanager.NewFromConfig(cfg), cfg.Region, nil
}

// getAWSSecret is the base-level API for grabbing secrets from AWS SecretManager
// This could be made public if useful later, but right now it's entirely
// implemented as a YAML field.
// TODO: need to turn on multi-region replication for secret manager
// TODO: need to turn on secret rotation
// TODO: need to audit which creds have access to which secrets
func getAWSSecret(ctx context.Context, secretName string) (string, error) {
	svc, awsRegion, err := getClient(ctx)
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	// VersionStage defaults to AWSCURRENT if unspecified
	input := &secretsmanager.GetSecretValueInput{SecretId: &secretName, VersionStage: aws.String("AWSCURRENT")}
	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
	result, err := svc.GetSecretValue(ctx, input)
	if err != nil {
		return "", ucerr.Errorf("failed to load AWS secret '%s' from '%s': %w", secretName, awsRegion, err)
	}
	uclog.Debugf(ctx, "Loaded AWS secret '%s' from '%s'", secretName, awsRegion)
	return decodeSecret(result)
}

func decodeSecret(result *secretsmanager.GetSecretValueOutput) (string, error) {
	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secret string
	if result.SecretString != nil {
		secret = *result.SecretString
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			return "", ucerr.Wrap(err)
		}
		secret = string(decodedBinarySecretBytes[:len])
	}
	if secret == "" {
		return "", ucerr.Errorf("failed to decode secret %s", *result.Name)
	}
	return secret, nil
}

func saveAWSSecret(ctx context.Context, path, secret string) error {
	localClient, _, err := getClient(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	err = saveSecretWithClient(ctx, path, secret, localClient)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func saveSecretWithClient(ctx context.Context, path, secret string, svc awsSecretsProvider) error {
	// serialize the secret into our silly awsSecret JSON blob
	j, err := json.Marshal(awsSecret{secret})
	if err != nil {
		return ucerr.Wrap(err)
	}
	js := string(j)

	uclog.Infof(ctx, "creating secret '%s' in AWS", path)
	_, err = svc.CreateSecret(ctx, &secretsmanager.CreateSecretInput{Name: &path, SecretString: &js, Tags: getTagsForSecret()})
	if err == nil {
		return nil
	}
	var resourceExistsErr *types.ResourceExistsException
	if errors.As(err, &resourceExistsErr) {
		uclog.Infof(ctx, "Secret '%s' already exists, updating it instead", path)
		_, err = svc.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{SecretId: &path, SecretString: &js})
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(err)
}

func deleteAWSSecret(ctx context.Context, path string) error {
	svc, _, err := getClient(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "Delete secret '%s' in AWS", path)
	_, err = svc.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{SecretId: &path, RecoveryWindowInDays: aws.Int64(secretRecoveryWindowInDays)})
	return ucerr.Wrap(err)
}

func newAWSString(ctx context.Context, uv universe.Universe, serviceName, name, secret string) (*String, error) {
	path := getSecretPath(uv, serviceName, name)
	loc := fmt.Sprintf("%s%s", PrefixAWS, path)
	if err := saveAWSSecret(ctx, path, secret); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &String{location: loc}, nil
}

func getTagsForSecret() []types.Tag {
	uv := universe.Current()
	tags := []types.Tag{
		{
			Key:   aws.String(universe.EnvKeyUniverse),
			Value: aws.String(string(uv)),
		},
	}
	if uv.IsCloud() {
		tags = append(tags, types.Tag{
			Key:   aws.String("UC_ENV_TYPE"),
			Value: aws.String("eks"),
		})
	}
	return tags
}
