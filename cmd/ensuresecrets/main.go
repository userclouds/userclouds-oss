package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
)

const (
	secretsFile = ".envrc.private"
)

var secretsMap = map[string]string{
	// key (env var) must be the same is in config/**/dev.yaml files under featureflags.api_key
	"STATSIG_DEV_API_KEY": "aws://secrets/local-dev/statsig/v1",
	// See under email object config/plex/dev.yaml and config/idp/dev.yaml
	"AWS_DEV_CREDS_AWS_KEY_ID":             "aws://secrets/dev/aws-ses-creds-aws-key",
	"AWS_DEV_CREDS_AWS_KEY_SECRET":         "aws://secrets/dev/aws-ses-creds-aws-secret",
	"UC_CONFIG_TEST_STAGING_CLIENT_SECRET": "aws://secrets/staging/deploy-tests/uccconfig/client-secret",
	"SDK_TEST_STAGING_CLIENT_SECRET":       "aws://secrets/staging/deploy-tests/sdks/client-secret",
	"UC_CONFIG_TEST_PROD_CLIENT_SECRET":    "aws://secrets/prod/deploy-tests/uccconfig/client-secret",
	"SDK_TEST_PROD_CLIENT_SECRET":          "aws://secrets/prod/deploy-tests/sdks/client-secret",
	"UC_CONFIG_TEST_DEBUG_CLIENT_SECRET":   "aws://secrets/debug/deploy-tests/uccconfig/client-secret",
	"SDK_TEST_DEBUG_CLIENT_SECRET":         "aws://secrets/debug/deploy-tests/sdks/client-secret",
	"SQLSHIM_TEST_STAGING_CLIENT_SECRET":   "aws://secrets/staging/deploy-tests/sqlshim/client-secret",
	"SQLSHIM_TEST_PROD_CLIENT_SECRET":      "aws://secrets/prod/deploy-tests/sqlshim/client-secret",
	"SQLSHIM_TEST_MYSQL_PASSWORD":          "aws://secrets/prod/deploy-tests/sqlshim/mysql-password",
}

type secretsInfo struct {
	lines   []string
	envVars []string
	missing set.Set[string]
}

func newSecretsInfo(ctx context.Context) secretsInfo {
	lines := make([]string, 0, len(secretsMap))
	envVars := make([]string, 0, len(secretsMap))
	for envVar, secretLocation := range secretsMap {
		currSecret := secret.FromLocation(secretLocation)
		secretValue, err := currSecret.Resolve(ctx)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to resolve secret %v: %v", secretLocation, err)
		}
		envVars = append(envVars, envVar)
		lines = append(lines, fmt.Sprintf("export %v=%s", envVar, secretValue))

	}
	return secretsInfo{
		lines:   lines,
		envVars: envVars,
		missing: set.NewStringSet(envVars...), // assume all env vars are missing
	}
}
func (s secretsInfo) GetAllMissingLines() []string {
	lines := make([]string, 0)
	for i, envVar := range s.envVars {
		if s.missing.Contains(envVar) {
			lines = append(lines, s.lines[i])
		}
	}
	return lines
}

func (s secretsInfo) GetMatchingLine(line string) (bool, string) {
	for i, envVar := range s.envVars {
		if !strings.Contains(line, envVar) {
			continue
		}
		s.missing.Evict(envVar)
		if s.lines[i] == line {
			return false, ""
		}
		return true, s.lines[i]
	}
	return false, ""
}

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "ensuresecrets")
	defer logtransports.Close()
	secrets := newSecretsInfo(ctx)
	if writeNewFileIfNotExists(ctx, secrets) {
		return
	}
	file, err := os.Open(secretsFile)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to open file: %v", err)
	}
	defer file.Close()
	uclog.Infof(ctx, "Reading existing env secrets file: %v", secretsFile)
	scanner := bufio.NewScanner(file)
	lines := make([]string, 0)
	madeChanges := false
	for scanner.Scan() {
		line := scanner.Text()
		if replaceLine, newLine := secrets.GetMatchingLine(line); replaceLine {
			line = newLine
			madeChanges = true
		}
		lines = append(lines, line)
	}
	for _, missingLine := range secrets.GetAllMissingLines() {
		lines = append(lines, missingLine)
		madeChanges = true
	}

	if !madeChanges {
		uclog.Infof(ctx, "No changes made to %v", secretsFile)
		return
	}
	writeFile(ctx, lines)
}

func writeNewFileIfNotExists(ctx context.Context, si secretsInfo) bool {
	if _, err := os.Stat(secretsFile); os.IsNotExist(err) {
		uclog.Infof(ctx, "Creating new env secrets file: %v", secretsFile)
		writeFile(ctx, si.GetAllMissingLines())
		return true
	} else if err != nil {
		uclog.Fatalf(ctx, "Failed to check if %v exists: %v", secretsFile, err)
	}
	return false
}

func writeFile(ctx context.Context, lines []string) {
	file, err := os.Create(secretsFile)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to create file: %v: %v", secretsFile, err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	uclog.Infof(ctx, "Writing %v lines to file %v", len(lines), secretsFile)
	for _, line := range lines {
		if _, err = writer.WriteString(line + "\n"); err != nil {
			uclog.Fatalf(ctx, "Failed to write line '%v' to file %v: %v", line, file.Name(), err)
		}
	}
}
