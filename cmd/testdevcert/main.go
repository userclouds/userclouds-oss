// Starts an HTTPS server and makes a request to ensure that the devlb certificate is trusted.

package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/repopath"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "testdevcert")
	defer logtransports.Close()

	// Start HTTPS server
	port := "31632" // randomly chosen
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})
	uclog.Infof(ctx, "Starting simple server on 127.0.0.1: %s to check whether devlb HTTPS cert is trusted...", port)
	go func() {
		rootPath := repopath.BaseDir()
		uclog.Fatalf(ctx, "%v", http.ListenAndServeTLS("127.0.0.1:"+port, filepath.Join(rootPath, "/cert/devlb.crt"), filepath.Join(rootPath, "/cert/devlb.key"), nil))
	}()

	// Send test request. If the cert is not trusted, we will get an error here. Server might take a
	// few moments to start, so add retries
	for range 10 {
		// dev.userclouds.tools should point to 127.0.0.1
		uclog.Infof(ctx, "Sending a request to the test server")
		resp, err := http.Get("https://dev.userclouds.tools:" + port)
		if err == nil {
			uclog.Infof(ctx, "Got a successful response with status %v", resp.StatusCode)
			return
		}
		uclog.Infof(ctx, "Got error: %v", err)
		if strings.Contains(err.Error(), "failed to verify certificate") {
			// Bail out early, retries won't help
			uclog.Fatalf(ctx, "Certificate is not trusted")
		}
		time.Sleep(time.Second)
	}

	// If we made it here, didn't get a successful response
	os.Exit(1)
}
