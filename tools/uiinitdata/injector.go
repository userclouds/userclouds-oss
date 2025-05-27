package uiinitdata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/yamlconfig"
)

const (
	htmlInjectLocation = `window.ucAppInitData={}`
	existingDataRegEx  = `<script defer="defer">window\.ucAppInitData=.*}</script>`
)

type appInitData struct {
	Universe       string
	VersionSha     string
	SentryDsn      string
	StatsSigAPIKey string
}

// LoadAndInjectAppInitData loads the config for the specified UI service and injects the data into the specified index.html file.
func LoadAndInjectAppInitData(ctx context.Context, indexFilePath string, failOnNoInject, dryRun bool) error {
	var cfg Config
	if err := yamlconfig.LoadToolConfig(ctx, "consoleui", &cfg); err != nil {
		return ucerr.Wrap(err)
	}
	versionSha, err := getVersionSha()
	if err != nil {
		return ucerr.Wrap(err)
	}
	initData := appInitData{
		Universe:       string(universe.Current()),
		VersionSha:     versionSha,
		SentryDsn:      cfg.Sentry.Dsn,
		StatsSigAPIKey: cfg.StatsigAPIKey,
	}
	return ucerr.Wrap(injectInitDataToIndex(ctx, indexFilePath, initData, failOnNoInject, dryRun))
}

// ClearUIUnitData clears the UI unit data from the specified index.html file.
func ClearUIUnitData(ctx context.Context, indexFilePath string, dryRun bool) error {
	htmlData, err := getHTMLData(indexFilePath)
	if err != nil {
		return ucerr.Wrap(err)
	}
	updatedHTML := regexp.MustCompile(existingDataRegEx).ReplaceAllString(htmlData, "")
	if updatedHTML == htmlData {
		return ucerr.New("No existing data found to clear")
	}
	if dryRun {
		uclog.Infof(ctx, "Dry run: Would have written %v bytes to %v", len(updatedHTML), indexFilePath)
		return nil
	}
	return ucerr.Wrap(saveHTMLData(ctx, indexFilePath, updatedHTML))
}

func getVersionSha() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", ucerr.Wrap(err)
	}
	return strings.TrimSpace(strings.ReplaceAll(stdout.String(), "\n", "")), nil
}

func getHTMLData(indexFilePath string) (string, error) {
	htmlFile, err := os.Open(indexFilePath)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	htmlBytes, err := io.ReadAll(htmlFile)
	htmlFile.Close()
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return string(htmlBytes), nil
}
func saveHTMLData(ctx context.Context, indexFilePath string, data string) error {
	newFile, err := os.Create(indexFilePath)
	if err != nil {
		return ucerr.Wrap(err)
	}
	defer newFile.Close()
	nbytes, err := newFile.Write([]byte(data))
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "New Html file written to %v, %v bytes", indexFilePath, nbytes)
	return nil
}

func injectInitDataToIndex(ctx context.Context, indexFilePath string, initData appInitData, failOnNoInject, dryRun bool) error {
	htmlData, err := getHTMLData(indexFilePath)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if !strings.Contains(htmlData, htmlInjectLocation) {
		if failOnNoInject {
			return ucerr.Errorf("No injection location found in %v", indexFilePath)
		}
		uclog.Debugf(ctx, "No injection location found in %v (data already injected)", indexFilePath)
		return nil
	}

	initDataJSON, err := json.Marshal(initData)
	if err != nil {
		return ucerr.Wrap(err)
	}
	newHTML := fmt.Sprintf(`window.ucAppInitData=%s`, initDataJSON)
	updatedHTML := strings.Replace(htmlData, htmlInjectLocation, newHTML, 1)
	if !dryRun {
		return ucerr.Wrap(saveHTMLData(ctx, indexFilePath, updatedHTML))

	}
	return nil
}
