package internal

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesisanalyticsv2"
	katypes "github.com/aws/aws-sdk-go-v2/service/kinesisanalyticsv2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/logserver/config"
)

type connectToDefaultDBValidator struct {
	dbName string
}

// Validate implements ucdb.Validator
func (sv connectToDefaultDBValidator) Validate(ctx context.Context, db *ucdb.DB) error {
	return nil
}

// DefaultDBValidator returns a Validator
func DefaultDBValidator(dbName string) ucdb.Validator {
	return connectToDefaultDBValidator{dbName: dbName}
}

func provisionAdvancedAnalyticsForTenant(ctx context.Context, cfg aws.Config, regionName string, tenantID uuid.UUID,
	service service.Service, resourcePath string, binaryPath string, ignoreErrors bool) error {
	uclog.Debugf(ctx, "Provisioning tenant %s in %s for service %s started", tenantID.String(), regionName, service)

	o := config.NewAdvanceAnalyticsResourceNames(tenantID, regionName, service, config.AWSDefaultOrg) // TODO create wrapper orgs
	// Remove the resources if provisioning fails mid way
	if !ignoreErrors {
		defer func() {
			if err := cleanupHelper(ctx, cfg, regionName, tenantID, service, &o); err != nil {
				uclog.Errorf(ctx, "cleanupHelper: %v", err)
			}
		}()
	}

	// Create a kinesis stream
	kc := kinesis.NewFromConfig(cfg)
	_, err := kc.CreateStream(ctx, &kinesis.CreateStreamInput{ShardCount: aws.Int32(1), StreamName: &o.StreamName})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "failed to create kinesis stream: %v", err)
		return ucerr.Wrap(err)
	}

	err = kinesis.NewStreamExistsWaiter(kc).Wait(ctx, &kinesis.DescribeStreamInput{StreamName: &o.StreamName}, 30*time.Second)
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Waiting for kinesis stream creation to complete: %v", err)
		return ucerr.Wrap(err)
	}

	// Create a policy that only allows writing to the newly created kinesis stream
	svcIAM := iam.NewFromConfig(cfg)

	//Load the user policy config file and populate it with values from the template
	userPolicyData := NewUserPolicyConfigData(o)
	var userPolicyDoc string

	err = GetJSONEncStringFromTemplate(ctx, resourcePath+"/"+userPolicyResourceFile, &userPolicyData, &userPolicyDoc, ignoreErrors)
	if err != nil && !ignoreErrors {
		return ucerr.Wrap(err)
	}

	_, err = svcIAM.CreatePolicy(ctx, &iam.CreatePolicyInput{PolicyDocument: aws.String(userPolicyDoc),
		PolicyName: &o.UserPolicyName})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to create policy : %v", err)
		return ucerr.Wrap(err)
	}

	// Create the aim account associated with the above policy
	_, err = svcIAM.CreateUser(ctx, &iam.CreateUserInput{UserName: &o.UserName, PermissionsBoundary: &o.UserPolicyARN})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to create user : %v", err)
		return ucerr.Wrap(err)
	}
	_, err = svcIAM.AttachUserPolicy(ctx, &iam.AttachUserPolicyInput{UserName: &o.UserName, PolicyArn: &o.UserPolicyARN})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to attach user policy : %v", err)
		return ucerr.Wrap(err)
	}

	// Create S3 bucket for code and output
	svcS3 := s3.NewFromConfig(cfg)

	_, err = svcS3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &o.Bucket})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to create S3 bucket : %v", err)
		return ucerr.Wrap(err)
	}

	// Create CloudWatch log group and log stream
	svcCloudWatch := cloudwatchlogs.NewFromConfig(cfg)

	_, err = svcCloudWatch.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{LogGroupName: &o.LogGroupName})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to create log group : %v", err)
		return ucerr.Wrap(err)
	}

	_, err = svcCloudWatch.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{LogGroupName: &o.LogGroupName,
		LogStreamName: &o.LogStreamName})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to create log stream : %v", err)
		return ucerr.Wrap(err)
	}

	// Create a policy for the role running the analytics app
	appPolicyData := NewAppPolicyConfigData(o)
	var appPolicyDoc string

	err = GetJSONEncStringFromTemplate(ctx, resourcePath+"/"+appPolicyResourceFile, &appPolicyData, &appPolicyDoc, ignoreErrors)
	if err != nil && !ignoreErrors {
		return ucerr.Wrap(err)
	}

	_, err = svcIAM.CreatePolicy(ctx, &iam.CreatePolicyInput{PolicyDocument: aws.String(appPolicyDoc),
		PolicyName: &o.AppPolicyName})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to create app policy : %v", err)
		return ucerr.Wrap(err)
	}

	// Create a role associated with this policy
	assumeRolePolicy := "{ \"Version\": \"2012-10-17\", \"Statement\": { \"Effect\": \"Allow\", \"Principal\": {\"Service\": \"kinesisanalytics.amazonaws.com\"}, \"Action\": \"sts:AssumeRole\"}}"
	_, err = svcIAM.CreateRole(ctx, &iam.CreateRoleInput{AssumeRolePolicyDocument: &assumeRolePolicy,
		PermissionsBoundary: &o.AppPolicyARN,
		RoleName:            &o.AppRoleName})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to create app role : %v", err)
		return ucerr.Wrap(err)
	}

	_, err = svcIAM.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{RoleName: &o.AppRoleName, PolicyArn: &o.AppPolicyARN})
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to attach app policy to the role : %v", err)
		return ucerr.Wrap(err)
	}

	// Upload the application jar file to S3
	err = AddFileToS3(ctx, svcS3, binaryPath+"/kinesis-plexanalytics-1.0.jar",
		o.Bucket, o.CodePath)
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to upload the binary : %v", err)
		return ucerr.Wrap(err)
	}

	// Create the kinesis application
	ka := kinesisanalyticsv2.NewFromConfig(cfg)

	// Load the values into the template struct
	a := NewAppCreateConfig(o)
	var aI kinesisanalyticsv2.CreateApplicationInput

	//Load the application config file and populate it with values from the template
	err = GetObjectFromTemplate(ctx, resourcePath+"/"+appCreateConfigResourceFile, &aI, &a, ignoreErrors)
	if err != nil && !ignoreErrors {
		return ucerr.Wrap(err)
	}

	_, err = ka.CreateApplication(ctx, &aI)
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to create the application : %v", err)
		return ucerr.Wrap(err)
	}

	// Start the newly created application
	/* TODO to avoid extra costs don't start the app yet
	_, err = ka.StartApplication(&kinesisanalyticsv2.StartApplicationInput{ApplicationName: &o.ApplicationName})
	if err != nil && !ignoreErrors {
		uclog.Errorf(c, "Failed to start the application : %v", err)
		return ucerr.Wrap(err)
	}*/

	// Success provisioning AWS resources
	o.Provisioned = true
	uclog.Debugf(ctx, "Success provisioning tenant %s in %s for service %s", tenantID.String(), regionName, service)
	return nil
}

func cleanupHelper(ctx context.Context,
	s aws.Config,
	regionName string,
	tenantID uuid.UUID,
	service service.Service,
	o *config.AdvancedAnalyticsResourceNames) error {

	if !o.Provisioned {
		return ucerr.Wrap(deProvisionAdvancedAnalyticsForTenant(ctx, s, regionName, tenantID, service, true))
	}

	return ucerr.Wrap(validateAdvancedAnalyticsForTenant(ctx, s, regionName, tenantID, service))
}

// AddFileToS3 uploads a file to a given path within a given bucket
func AddFileToS3(ctx context.Context, s3client *s3.Client, fileDir string, bucket string, outputPath string) error {
	// Create an uploader with the session and default options
	uploader := manager.NewUploader(s3client)

	f, err := os.Open(fileDir)
	if err != nil {
		uclog.Errorf(ctx, "Failed to open file %q, %v", fileDir, err)
		return ucerr.Wrap(err)
	}

	// Upload the file to S3.
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(outputPath),
		Body:   f,
	})

	return ucerr.Wrap(err)
}

func deProvisionAdvancedAnalyticsForTenant(ctx context.Context, cfg aws.Config, regionName string, tenantID uuid.UUID, service service.Service, ignoreErrors bool) error {

	uclog.Debugf(ctx, "Deprovisioning tenant %s in %s for service %s started", tenantID.String(), regionName, service)

	o := config.NewAdvanceAnalyticsResourceNames(tenantID, regionName, service, config.AWSDefaultOrg) // TODO create wrapper orgs

	// Get the status of the kinesis app
	ka := kinesisanalyticsv2.NewFromConfig(cfg)
	kinesisApp, err := ka.DescribeApplication(ctx, &kinesisanalyticsv2.DescribeApplicationInput{ApplicationName: &o.ApplicationName})
	err = checkForAWSError(ctx, err, "ResourceNotFoundException", "Failed to describe the application", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}
	var appStatus katypes.ApplicationStatus
	if kinesisApp != nil && kinesisApp.ApplicationDetail != nil {
		appStatus = kinesisApp.ApplicationDetail.ApplicationStatus
	}

	// Stop the kinesis analytics app if it is running or starting up
	if appStatus == katypes.ApplicationStatusRunning || appStatus == katypes.ApplicationStatusStarting {
		_, err = ka.StopApplication(ctx, &kinesisanalyticsv2.StopApplicationInput{ApplicationName: &o.ApplicationName,
			Force: aws.Bool(true)})
		err = checkForAWSError(ctx, err, "ResourceNotFoundException",
			"Failed to stop the application", ignoreErrors)
		if err != nil {
			return ucerr.Wrap(err)
		}
	}

	// Delete the analytics app
	if kinesisApp != nil && kinesisApp.ApplicationDetail != nil {
		_, err = ka.DeleteApplication(ctx, &kinesisanalyticsv2.DeleteApplicationInput{ApplicationName: &o.ApplicationName,
			CreateTimestamp: kinesisApp.ApplicationDetail.CreateTimestamp})
		err = checkForAWSError(ctx, err, "ResourceNotFoundException",
			"Failed to delete the application", ignoreErrors)
		if err != nil {
			return ucerr.Wrap(err)
		}
	}
	// Delete the role for analytics app
	svcIAM := iam.NewFromConfig(cfg)
	_, err = svcIAM.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{RoleName: &o.AppRoleName, PolicyArn: &o.AppPolicyARN})
	err = checkForAWSError(ctx, err, "NoSuchEntity", "Failed to detach app policy from role", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	_, err = svcIAM.DeleteRole(ctx, &iam.DeleteRoleInput{RoleName: &o.AppRoleName})
	err = checkForAWSError(ctx, err, "NoSuchEntity", "Failed to delete app role", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Delete the policy for the role for analytics app
	_, err = svcIAM.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: &o.AppPolicyARN})
	err = checkForAWSError(ctx, err, "NoSuchEntity",
		"Failed to delete app policy", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Delete the log stream
	svcCloudWatch := cloudwatchlogs.NewFromConfig(cfg)
	_, err = svcCloudWatch.DeleteLogStream(ctx, &cloudwatchlogs.DeleteLogStreamInput{LogGroupName: &o.LogGroupName,
		LogStreamName: &o.LogStreamName})
	err = checkForAWSError(ctx, err, "ResourceNotFoundException", "Failed to delete log stream", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Delete the log group
	_, err = svcCloudWatch.DeleteLogGroup(ctx, &cloudwatchlogs.DeleteLogGroupInput{LogGroupName: &o.LogGroupName})
	err = checkForAWSError(ctx, err, "ResourceNotFoundException", "Failed to delete log group", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Delete the S3 bucket used for the logs and the code
	svcS3 := s3.NewFromConfig(cfg)

	output, err := svcS3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &o.Bucket})
	if err != nil {
		return ucerr.Wrap(err)
	}
	for _, obj := range output.Contents {
		_, err = svcS3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &o.Bucket, Key: obj.Key})
		if err != nil {
			return ucerr.Wrap(err)
		}
	}
	// Delete the bucket itself (which should now be empty)
	_, err = svcS3.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: &o.Bucket})
	err = checkForAWSError(ctx, err, "NoSuchBucket", "Failed to delete S3 bucket", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Delete the IAM user for writing to the kinesis stream
	_, err = svcIAM.DetachUserPolicy(ctx, &iam.DetachUserPolicyInput{UserName: &o.UserName, PolicyArn: &o.UserPolicyARN})
	err = checkForAWSError(ctx, err, "NoSuchEntity",
		"Failed to detach policy from IAM user", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	_, err = svcIAM.DeleteUser(ctx, &iam.DeleteUserInput{UserName: &o.UserName})
	err = checkForAWSError(ctx, err, "NoSuchEntity",
		"Failed to delete IAM user", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Delete the user policy
	_, err = svcIAM.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: &o.UserPolicyARN})
	err = checkForAWSError(ctx, err, "NoSuchEntity",
		"Failed to delete user policy", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Delete the kinesis stream
	kc := kinesis.NewFromConfig(cfg)
	_, err = kc.DeleteStream(ctx, &kinesis.DeleteStreamInput{StreamName: &o.StreamName})
	err = checkForAWSError(ctx, err, "ResourceNotFoundException", "Failed to delete kinesis stream", ignoreErrors)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// TODO - Delete the company for now using our company as an umbrella

	// Success or errors were ignored
	uclog.Debugf(ctx, "Success deprovisioning tenant %s in %s for service %s", tenantID.String(), regionName, service)
	return nil
}

func checkForAWSError(ctx context.Context, err error, expectedErrorCode string, errorMessage string, ignoreErrors bool) error {
	if err == nil {
		return nil
	}
	uclog.Debugf(ctx, "%s: %v", errorMessage, err)
	if ignoreErrors {
		return nil
	}
	var awsErr smithy.APIError
	if errors.As(err, &awsErr) {
		if awsErr.ErrorCode() == expectedErrorCode {
			return nil
		}
	}
	return ucerr.Wrap(err)
}

func validateAdvancedAnalyticsForTenant(ctx context.Context,
	cfg aws.Config,
	regionName string,
	tenantID uuid.UUID,
	service service.Service) error {

	uclog.Debugf(ctx, "Validating tenant %s in %s for service %s", tenantID.String(), regionName, service)

	o := config.NewAdvanceAnalyticsResourceNames(tenantID, regionName, service, config.AWSDefaultOrg) // TODO create wrapper orgs

	// Check if the app exists
	ka := kinesisanalyticsv2.NewFromConfig(cfg)
	kinesisApp, err := ka.DescribeApplication(ctx, &kinesisanalyticsv2.DescribeApplicationInput{ApplicationName: &o.ApplicationName})
	if err != nil {
		return ucerr.Wrap(err)
	}
	// Check if the app is running or starting state
	var appStatus katypes.ApplicationStatus = kinesisApp.ApplicationDetail.ApplicationStatus
	if appStatus != katypes.ApplicationStatusRunning &&
		appStatus != katypes.ApplicationStatusStarting &&
		appStatus != katypes.ApplicationStatusReady {
		return ucerr.Errorf("Kinesis app is in unexpected state - %s", appStatus)
	}
	// Check if the ARNs match
	if *kinesisApp.ApplicationDetail.ApplicationARN != o.ApplicationARN {
		return ucerr.Errorf("Kinesis app ARN is unexpected - %s", *kinesisApp.ApplicationDetail.ApplicationARN)
	}
	// Check for existence of the kinesis stream
	kc := kinesis.NewFromConfig(cfg)
	streamDesc, err := kc.DescribeStream(ctx, &kinesis.DescribeStreamInput{StreamName: &o.StreamName})
	if err != nil {
		uclog.Errorf(ctx, "Failed to describe kinesis stream : %v", err)
		return ucerr.Wrap(err)
	}
	if *streamDesc.StreamDescription.StreamARN != o.StreamARN {
		return ucerr.Errorf("Kinesis app ARN is unexpected - %s", *streamDesc.StreamDescription.StreamARN)
	}

	// Check if the log group exists and contains the expected log stream
	svcCloudWatch := cloudwatchlogs.NewFromConfig(cfg)
	logGroup, err := svcCloudWatch.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{LogGroupNamePrefix: &o.LogGroupName})
	if err != nil || len(logGroup.LogGroups) != 1 {
		uclog.Errorf(ctx, "Failed to get log group : %v", err)
		return ucerr.Wrap(err)
	}
	if *logGroup.LogGroups[0].Arn != (o.LogGroupARN + ":*") {
		return ucerr.Errorf("Unexpected log group ARN - %s", *logGroup.LogGroups[0].Arn)
	}
	logStreams, err := svcCloudWatch.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{LogGroupName: &o.LogGroupName})
	if err != nil {
		uclog.Errorf(ctx, "Failed to describe log streams : %v", err)
		return ucerr.Wrap(err)
	}
	var streamFound bool
	for _, v := range logStreams.LogStreams {
		if *v.Arn == o.LogStreamARN && *v.LogStreamName == o.LogStreamName {
			streamFound = true
		}
	}
	if !streamFound {
		return ucerr.Errorf("Failed to find cloud watch stream - %s", o.LogStreamARN)
	}

	// Check for existance of the role for analytics app
	svcIAM := iam.NewFromConfig(cfg)

	appRole, err := svcIAM.GetRole(ctx, &iam.GetRoleInput{RoleName: &o.AppRoleName})
	if err != nil {
		uclog.Errorf(ctx, "Failed to get the app role : %v", err)
		return ucerr.Wrap(err)
	}

	if *appRole.Role.Arn != o.AppRoleARN {
		return ucerr.Errorf("App role ARN is unexpected - %s", *appRole.Role.Arn)
	}

	if appRole.Role.PermissionsBoundary == nil ||
		*appRole.Role.PermissionsBoundary.PermissionsBoundaryArn != o.AppPolicyARN {
		return ucerr.Errorf("App role boundary permission is missing or unexpected - %v", appRole.Role.PermissionsBoundary)
	}
	// Check on the policy for the app role
	appPolicy, err := svcIAM.GetPolicy(ctx, &iam.GetPolicyInput{PolicyArn: &o.AppPolicyARN})
	if err != nil {
		uclog.Errorf(ctx, "Failed to get the app policy : %v", err)
		return ucerr.Wrap(err)
	}
	if *appPolicy.Policy.PolicyName != o.AppPolicyName {
		return ucerr.Errorf("App policy name is unexpected - %s", *appPolicy.Policy.PolicyName)
	}
	if *appPolicy.Policy.AttachmentCount != 1 {
		return ucerr.Errorf("App count is unexpected - %d", *appPolicy.Policy.AttachmentCount)
	}
	// Check on the policy for the user
	userPolicy, err := svcIAM.GetPolicy(ctx, &iam.GetPolicyInput{PolicyArn: &o.UserPolicyARN})
	if err != nil {
		uclog.Errorf(ctx, "Failed to get the app policy : %v", err)
		return ucerr.Wrap(err)
	}
	if *userPolicy.Policy.PolicyName != o.UserPolicyName {
		return ucerr.Errorf("App policy name is unexpected - %s", *userPolicy.Policy.PolicyName)
	}
	if *userPolicy.Policy.AttachmentCount != 1 {
		return ucerr.Errorf("App count is unexpected - %d", *userPolicy.Policy.AttachmentCount)
	}
	// Check the user
	user, err := svcIAM.GetUser(ctx, &iam.GetUserInput{UserName: &o.UserName})
	if err != nil {
		uclog.Errorf(ctx, "Failed to get the user : %v", err)
		return ucerr.Wrap(err)
	}
	if *user.User.Arn != o.UserARN {
		return ucerr.Errorf("User ARN is unexpected - %s", *userPolicy.Policy.PolicyName)
	}
	if user.User.PermissionsBoundary == nil ||
		*user.User.PermissionsBoundary.PermissionsBoundaryArn != o.UserPolicyARN {
		return ucerr.Errorf("User boundary ARN is unexpected - %v", user.User.PermissionsBoundary)
	}
	// Check the S3 bucket
	svcS3 := s3.NewFromConfig(cfg)

	objectsS3, err := svcS3.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{Bucket: &o.Bucket, MaxKeys: aws.Int32(10)})
	if err != nil || len(objectsS3.Versions) == 0 {
		uclog.Errorf(ctx, "Couldn't list uploaded jar in the bucket : %v", err)
		return ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "Success validating tenant %s in %s for service %s", tenantID.String(), regionName, service)
	return nil
}

// ProvisionKinesisRegionResourcesForService creates the resources for a [tenant, region, service] combo requires to run
// a Kinesis data analytics app
func ProvisionKinesisRegionResourcesForService(ctx context.Context, cfg *Config, tenantID uuid.UUID, awsRegion string, service service.Service, force bool) error {

	awsCfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// TODO copy the JAVA jar into the central bin directory and fix up the paths
	err = provisionAdvancedAnalyticsForTenant(ctx, awsCfg, awsRegion, tenantID, service, "/Users/vladf/Documents/GitHub/userclouds/dataprocessor/internal",
		"/Users/vladf/Documents/GitHub/analytics/plex-kinesis-basic/target", force)

	return ucerr.Wrap(err)
}

// DeProvisionKinesisRegionResourcesForService removes AWS resources for a [tenant, region, service] combo
func DeProvisionKinesisRegionResourcesForService(ctx context.Context, cfg *Config, tenantID uuid.UUID, awsRegion string, service service.Service, force bool) error {
	awsCfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(deProvisionAdvancedAnalyticsForTenant(ctx, awsCfg, awsRegion, tenantID, service, force))
}
