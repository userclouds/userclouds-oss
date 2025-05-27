package logtransports

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/uclog"
)

var numResults = 1000000

func TestLogVolume(t *testing.T) {

	cfg := GoLogJSONTransportConfig{
		Type: TransportTypeGoLogJSON,
		TransportConfig: uclog.TransportConfig{
			Required:    true,
			MaxLogLevel: uclog.LogLevelDebug,
		},
	}

	transports := map[string]uclog.Transport{
		"WrappedGoLogJSON":   cfg.GetWrappedTransport(service.Plex, nil, "test"),
		"UnwrappedGoLogJSON": cfg.GetUnwrappedTransport(service.Plex, nil, "test"),
	}

	for name, trans := range transports {
		t.Run(name, func(t *testing.T) {
			runStderrTest(t, trans)
		})
	}

	t.Run("file", func(t *testing.T) {
		tmpdir := t.TempDir()
		fn := filepath.Join(tmpdir, "test.log")

		fileTrans := FileTransportConfig{
			Type:     TransportTypeFile,
			Filename: fn,
			TransportConfig: uclog.TransportConfig{
				Required:    true,
				MaxLogLevel: uclog.LogLevelDebug,
			},
			Append: false,
		}.GetTransport(service.Plex, nil, "test")

		uclog.InitForService("test", []uclog.Transport{
			fileTrans,
		}, nil)

		for range numResults {
			uclog.Debugf(context.Background(), "this is a unique message")
		}
		missed := fileTrans.GetStats().DroppedEventCount
		uclog.Close()

		// read the file and make sure it's the right size
		content, err := os.ReadFile(fn)
		assert.NoErr(t, err)
		lines := bytes.Count(content, []byte("\n"))
		assert.Equal(t, lines, numResults-int(missed))

		// we shouldn't have dropped any events
		assert.Equal(t, int(missed), 0)
	})

}

func runStderrTest(t *testing.T, trans uclog.Transport) {
	ctx := context.Background()

	// Save original stderr
	oldStderr := os.Stderr

	// Create a pipe to capture stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	// make sure we aren't racing in the test asserts
	var wg sync.WaitGroup

	// Start a goroutine to read from the pipe
	var output []byte
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := r.Read(buf)
			if err != nil {
				break
			}
			output = append(output, buf[:n]...)
		}
	}()

	uclog.InitForService("test", []uclog.Transport{
		trans,
	}, nil)

	// Generate logs
	for range numResults {
		uclog.Debugf(ctx, "this is a unique message")
	}

	missed := trans.GetStats().DroppedEventCount
	// Close logger to ensure we flush
	uclog.Close()

	// Close the writer end of the pipe
	w.Close()

	// Wait for the reader to finish
	wg.Wait()

	// Restore original stderr for other tests
	os.Stderr = oldStderr

	// sanity to make sure our stderr capture worked
	assert.False(t, len(output) == 0, assert.Errorf("Expected to capture stderr output, but got none"))

	// Verify that we got exactly the right number of messages
	assert.Equal(t, strings.Count(string(output), "this is a unique message"), numResults-int(missed))

	// we shouldn't have dropped any events
	assert.Equal(t, int(missed), 0)
}

func BenchmarkLogVolumeGoJSONWrapped(b *testing.B) {
	cfg := GoLogJSONTransportConfig{
		Type: TransportTypeGoLogJSON,
		TransportConfig: uclog.TransportConfig{
			Required:    true,
			MaxLogLevel: uclog.LogLevelDebug,
		},
	}

	ctx := context.Background()
	uclog.InitForService("test", []uclog.Transport{cfg.GetWrappedTransport(service.Plex, nil, "test")}, nil)
	for b.Loop() {
		uclog.Debugf(ctx, "this is a unique message")
	}
	uclog.Close()
}

func BenchmarkLogVolumeGoJSONUnwrapped(b *testing.B) {
	cfg := GoLogJSONTransportConfig{
		Type: TransportTypeGoLogJSON,
		TransportConfig: uclog.TransportConfig{
			Required:    true,
			MaxLogLevel: uclog.LogLevelDebug,
		},
	}

	ctx := context.Background()
	uclog.InitForService("test", []uclog.Transport{cfg.GetUnwrappedTransport(service.Plex, nil, "test")}, nil)
	for b.Loop() {
		uclog.Debugf(ctx, "this is a unique message")
	}
	uclog.Close()
}
func BenchmarkLogVolumeFile(b *testing.B) {
	ctx := context.Background()
	cfg := FileTransportConfig{
		Type: TransportTypeFile,
		TransportConfig: uclog.TransportConfig{
			Required:    true,
			MaxLogLevel: uclog.LogLevelDebug,
		},
		Filename: filepath.Join(b.TempDir(), "test.log"),
		Append:   false,
	}

	uclog.InitForService("test", []uclog.Transport{cfg.GetTransport(service.Plex, nil, "test")}, nil)

	for b.Loop() {
		uclog.Debugf(ctx, "this is a unique message")
	}
	uclog.Close()
}
