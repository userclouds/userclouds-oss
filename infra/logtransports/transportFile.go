package logtransports

// Basic transport logging the raw events to a file in /tmp directory

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

func init() {
	registerDecoder(TransportTypeFile, func(value []byte) (TransportConfig, error) {
		var f FileTransportConfig
		// NB: we need to check the type here because the yaml decoder will happily decode an
		// empty struct, since dec.KnownFields(true) gets lost via the yaml.Unmarshaler
		// interface implementation
		if err := json.Unmarshal(value, &f); err == nil && f.Type == TransportTypeFile {
			return &f, nil
		}
		return nil, ucerr.New("Unknown transport type")
	})
}

// TransportTypeFile defines the file transport
const TransportTypeFile TransportType = "file"

// FileTransportConfig defines log-to-file client config
type FileTransportConfig struct {
	Type                  TransportType `yaml:"type" json:"type"`
	uclog.TransportConfig `yaml:"transportconfig" json:"transportconfig"`
	Filename              string `yaml:"filename" json:"filename"`
	BaseFilename          string `yaml:"base_filename" json:"base_filename"`

	Append       bool `yaml:"append" json:"append"`
	PrefixFlag   int  `yaml:"prefix_flag" json:"prefix_flag"`
	NoRequestIDs bool `yaml:"no_request_ids" json:"no_request_ids"`
}

// GetType implements TransportConfig
func (c FileTransportConfig) GetType() TransportType {
	return TransportTypeFile
}

// IsSingleton implements TransportConfig
func (c FileTransportConfig) IsSingleton() bool {
	return false
}

// GetTransport implements TransportConfig
func (c FileTransportConfig) GetTransport(svc service.Service, _ jsonclient.Option, _ string) uclog.Transport {
	return newTransportBackgroundIOWrapper(newFileTransport(&c, string(svc)))
}

// Validate implements Validateable
func (c *FileTransportConfig) Validate() error {
	if !c.Required {
		return nil
	}

	if c.Filename == "" && c.BaseFilename == "" {
		return ucerr.New("logging config invalid - missing filename or base_filename")
	} else if c.Filename != "" && c.BaseFilename != "" {
		return ucerr.New("logging config invalid - cannot specify filename and base_filename")
	}
	return nil
}

type fileTransport struct {
	svcName        string
	filename       string
	fileHandle     *os.File
	fileWriter     *bufio.Writer
	fileWriteMutex sync.Mutex
	config         FileTransportConfig
	re             *regexp.Regexp
	prefix         bool
}

const (
	fileTransportName = "FileTransport"
	defaultFilename   = "/tmp/user_cloud_log" // Default filename if the configuration file doesn't specify one
)

// Interval for flushing logged messages to disk
const writeToFileInterval time.Duration = 100 * time.Millisecond

func newFileTransport(c *FileTransportConfig, svcName string) *fileTransport {
	return &fileTransport{
		svcName: svcName,
		config:  *c,
	}
}

func (t *fileTransport) getFileName() string {
	if t.config.BaseFilename != "" {
		return fmt.Sprintf("%s%s.log", t.config.BaseFilename, t.svcName)
	}
	if t.config.Filename == "" {
		return defaultFilename
	}
	return t.config.Filename
}
func (t *fileTransport) init(ctx context.Context) (*uclog.TransportConfig, error) {
	c := &uclog.TransportConfig{Required: t.config.Required, MaxLogLevel: t.config.MaxLogLevel}

	// Extract data from the config object into the transport state
	t.filename = t.getFileName()
	// Check if we should append to the existing file or replace it
	var err error
	if t.config.Append {
		t.fileHandle, err = os.OpenFile(t.filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	} else {
		t.fileHandle, err = os.Create(t.filename)
	}
	t.fileWriter = bufio.NewWriter(t.fileHandle)
	t.fileWriteMutex = sync.Mutex{}

	t.prefix = true
	if t.config.PrefixFlag != DefaultPrefixVal {
		t.prefix = false
	}

	// Create a regex to strip color information prior to writing out the message
	// error-message coloring is done in GoLog, but devbox uses colors to annotate
	// different services at a higher level, which is what this solves.
	const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"
	t.re = regexp.MustCompile(ansi)

	return c, ucerr.Wrap(err)
}

func (t *fileTransport) writeMessages(ctx context.Context, logRecords *logRecord, startTime time.Time, count int) {
	// We flush the writes every time even if the buffer is not full to make sure the file log
	// doesn't fall behind. The Writer will flush 4096 bytes chunks by default.
	if logRecords != nil {
		defer t.fileWriter.Flush()
	}

	for ; logRecords != nil; logRecords = logRecords.next {
		message := logRecords.event.Message
		if !t.config.NoRequestIDs && !logRecords.event.RequestID.IsNil() {
			message = fmt.Sprintf("%v: %s", logRecords.event.RequestID, message)
		}
		// Append the time to the messages
		var recordBuffer = bytes.NewBuffer(make([]byte, 0, len(message)+21))
		if t.prefix {
			recordBuffer.WriteString(logRecords.timestamp.Format("Jan _2 15:04:05:00"))
			recordBuffer.WriteString(" ")
		}
		recordBuffer.WriteString(t.re.ReplaceAllString(message, ""))
		recordBuffer.WriteString("\n")
		// Write the message to the file
		t.fileWriter.Write(recordBuffer.Bytes())
	}
}
func (t *fileTransport) getFailedAPICallsCount() int64 {
	return 0
}

func (t *fileTransport) getIOInterval() time.Duration {
	return writeToFileInterval
}
func (t *fileTransport) getMaxLogLevel() uclog.LogLevel {
	return t.config.MaxLogLevel
}

func (t *fileTransport) getTransportName() string {
	return fileTransportName
}

func (t *fileTransport) supportsCounters() bool {
	return false
}

func (t *fileTransport) flushIOResources() {
	t.fileHandle.Sync()
}

func (t *fileTransport) closeIOResources() {
	_ = t.fileHandle.Close()
}
