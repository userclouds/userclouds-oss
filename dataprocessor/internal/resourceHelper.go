package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/logserver/config"
)

// Fixed filenames for the resource files
const appPolicyResourceFile string = "plex_analytics_app_policy_config.json"
const userPolicyResourceFile string = "plex_analytics_user_policy_config.json"
const appCreateConfigResourceFile string = "plex_analytics_app_create_config.json"

// PolicyDocument is our definition of our policies to be uploaded to IAM.
type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

// StatementEntry will dictate what this policy will allow or not allow.
type StatementEntry struct {
	Effect   string
	Action   []string
	Resource []string
}

// Template parsing structure used to populate JSON resource files with names of particular resources

// AppCreateConfig contains data to fill in appCreateConfigResourceFile
type AppCreateConfig struct {
	ApplicationName string
	RoleARN         string
	LogSteamARN     string
	Region          string
	OutputPath      string
	StreamName      string
	CompanyID       string
	BucketARN       string
	CodePath        string
}

// UserPolicyConfigData contains data to fill in userPolicyResourceFile
type UserPolicyConfigData struct {
	UserPolicyName string
	StreamARN      string
}

// AppPolicyConfigData contains data to fill in appPolicyResourceFile
type AppPolicyConfigData struct {
	AppPolicyName string
	StreamARN     string
	LogStreamARN  string
	CodePath      string
	OutputPath    string
}

// NewAppCreateConfig fills in the data from populated AdvancedAnalyticsResourceNames
func NewAppCreateConfig(o config.AdvancedAnalyticsResourceNames) AppCreateConfig {
	var a AppCreateConfig

	a.ApplicationName = o.ApplicationName
	a.BucketARN = o.BucketARN
	a.LogSteamARN = o.LogStreamARN
	a.CodePath = o.CodePath
	a.CompanyID = fmt.Sprint(o.TenantID)
	a.OutputPath = o.OutputPath
	a.Region = o.Region
	a.RoleARN = o.AppRoleARN
	a.StreamName = o.StreamName

	return a
}

// NewUserPolicyConfigData fills in the data from populated AdvancedAnalyticsResourceNames
func NewUserPolicyConfigData(o config.AdvancedAnalyticsResourceNames) UserPolicyConfigData {
	var u UserPolicyConfigData

	u.UserPolicyName = o.UserPolicyName
	u.StreamARN = o.StreamARN

	return u
}

// NewAppPolicyConfigData fills in the data from populated AdvancedAnalyticsResourceNames
func NewAppPolicyConfigData(o config.AdvancedAnalyticsResourceNames) AppPolicyConfigData {
	var a AppPolicyConfigData

	a.AppPolicyName = o.AppPolicyName
	a.StreamARN = o.StreamARN
	a.LogStreamARN = o.LogStreamARN
	a.CodePath = o.BucketARN + "/" + o.CodeFolder
	a.OutputPath = o.BucketARN + "/" + o.OutputFolder

	return a
}

// Load template from a file and populate with provided data into a provided bytes buffer
func populateTemplate(ctx context.Context, filePath string, b *bytes.Buffer, data any, ignoreErrors bool) error {
	t, err := template.ParseFiles(filePath)
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to parse the tempate file: %v", err)
		return ucerr.Wrap(err)
	}
	err = t.Execute(b, data)
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to populate the app creation template : %v", err)
		return ucerr.Wrap(err)
	}
	return nil
}

// GetObjectFromTemplate loads template from a file populate it with provided data and marshal into an object
func GetObjectFromTemplate(ctx context.Context, filePath string, output any, data any, ignoreErrors bool) error {
	var tpl bytes.Buffer
	err := populateTemplate(ctx, filePath, &tpl, data, ignoreErrors)
	if err != nil && !ignoreErrors {
		return ucerr.Wrap(err)
	}

	err = json.Unmarshal(tpl.Bytes(), output)
	if err != nil && !ignoreErrors {
		uclog.Errorf(ctx, "Failed to populate the template : %v", err)
		return ucerr.Wrap(err)
	}
	return nil
}

// GetJSONEncStringFromTemplate loads template from a file populate it with provided data and put it into json encoded string
func GetJSONEncStringFromTemplate(ctx context.Context, filePath string, data any, s *string, ignoreErrors bool) error {
	var tpl bytes.Buffer
	err := populateTemplate(ctx, filePath, &tpl, data, ignoreErrors)
	if err != nil && !ignoreErrors {
		return ucerr.Wrap(err)
	}

	*s = tpl.String()

	return nil
}
