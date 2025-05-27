package multirun

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/color"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// RunSingleCommand runs a single command and returns true if it ran successfully, false otherwise
func RunSingleCommand(ctx context.Context, bin string, cmdArgs ...string) error {
	cmds := []Command{{Bin: bin, Args: cmdArgs}}
	if err := SetupCommands(ctx, cmds); err != nil {
		return ucerr.Wrap(err)
	}
	env := NewEnv(ctx, cmds)
	WrapOutputs(cmds[0], env)
	return ucerr.Wrap(Run(ctx, cmds, env))
}

// SetupCommands creates the *exec.Commands, etc for each command in an array
func SetupCommands(ctx context.Context, cmds []Command) error {
	for i, c := range cmds {
		cmd := exec.Command(c.Bin, c.Args...)
		cmd.Dir = c.Path
		if len(c.Env) != 0 {
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, c.Env...)
		}
		cmds[i].cmd = cmd

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return ucerr.Wrap(err)
		}
		cmds[i].stdout = io.NopCloser(stdout)

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return ucerr.Wrap(err)
		}
		cmds[i].stderr = io.NopCloser(stderr)
	}
	return nil
}

// WrapOutputs sets up all the goroutines that watch each cmd's outputs
// and route them appropriately
func WrapOutputs(cmd Command, env *Env) {
	env.AllDone.Add(2)
	go wrapOutput(cmd, env, cmd.stdout, false)
	go wrapOutput(cmd, env, cmd.stderr, true)
}

type logLevel int

const (
	logLevelNormal logLevel = iota
	logLevelWarning
	logLevelError
)

func wrapOutput(cmd Command, env *Env, in io.ReadCloser, isStderr bool) {
	defer env.AllDone.Done()
	defer in.Close()

	fmtStr := fmt.Sprintf("%s%%s%%-%ds%s%s: %%s%%v", color.ANSIEscapeColor, env.MaxLen, color.ANSIEscapeColor, color.Default)

	scanner := bufio.NewScanner(in)

	// max out scanner's buffer size to avoid hiding errors
	bs := make([]byte, 0, 64*1024)
	scanner.Buffer(bs, 1024*1024)

	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		incoming := scanner.Bytes()
		level := logLevelNormal

		var humanReadableLog string
		unmarshaled := logtransports.JSONLogLine{}
		if json.Unmarshal(incoming, &unmarshaled) != nil {
			// not json, just print it
			humanReadableLog = string(incoming)
			if isStderr {
				level = logLevelError
			}
		} else {
			timestamp := time.Unix(0, unmarshaled.TimestampNS).Local().Format("2006/01/02 15:04:05")
			humanReadableLog = fmt.Sprintf("[%s] [%s] %s", timestamp, unmarshaled.RequestID, unmarshaled.Message)
			if unmarshaled.LogLevel == "error" {
				level = logLevelError
			} else if unmarshaled.LogLevel == "warn" {
				level = logLevelWarning
			}
		}

		var levelLabel string
		if level == logLevelError {
			levelLabel = fmt.Sprintf("%s%sERROR%s%s: ", color.ANSIEscapeColor, color.BrightRed, color.ANSIEscapeColor, color.Default)
		} else if level == logLevelWarning {
			levelLabel = fmt.Sprintf("%s%sWARN%s%s: ", color.ANSIEscapeColor, color.Yellow, color.ANSIEscapeColor, color.Default)
		}
		s := fmt.Sprintf(fmtStr, cmd.Color, cmd.GetName(), levelLabel, humanReadableLog)
		env.Output <- []byte(s)
	}

	// NB: this is a bit weird, but I haven't fully tracked it down yet. I think we
	// have a race between this read loop and the Wait() in RunMax which sometimes
	// causes scanner.Scan() to get os.ErrClosed instead of EOF when the cmd terminates.
	// According to the StdoutPipe docs in golang/os/exec/exec.go,
	//   StdoutPipe returns a pipe that will be connected to the command's
	//   standard output when the command starts.
	//   Wait will close the pipe after seeing the command exit, so most callers
	//   need not close the pipe themselves. It is thus incorrect to call Wait
	//   before all reads from the pipe have completed.
	// How one would not call Wait until the reads have been completed is left as an
	// exercise to the reader, and one I haven't solved. At the same time, the
	// Wait documentation (same file) reads:
	//   Wait waits for the command to exit and waits for any copying to
	//   stdin or copying from stdout or stderr to complete.
	// Which would seem to imply this is safe. Anyhow, just ignoring
	// os.ErrClosed for now :)
	// https://cs.opensource.google/go/go/+/refs/tags/go1.17.7:src/os/exec/exec.go
	if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
		env.Error <- ucerr.Errorf("%s(%s): %w", cmd.GetName(), cmd.Args, err)
	}
}

// Run runs a list of Commands in an Env and quits when they exit
func Run(ctx context.Context, cmds []Command, env *Env) error {
	// start them all now that we're ready
	for _, c := range cmds {
		if err := c.cmd.Start(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	uclog.Debugf(ctx, "processes started")

	// wait for them to terminate then cancel ourselves (shouldn't really be needed with servers)
	go func() {
		env.AllDone.Wait()
		env.Cancel()
	}()

	for {
		select {
		case <-env.Context.Done():
			var commandErrors error
			for _, c := range cmds {
				if err := c.cmd.Wait(); err != nil {
					commandErrors = ucerr.Combine(commandErrors, err)
				}
			}

			return ucerr.Wrap(commandErrors)
		case b := <-env.Output:
			uclog.Debugf(ctx, "%s", string(b))
		case err := <-env.Error:
			if err != nil {
				uclog.Debugf(ctx, "channel error: %v", err)
			}
		}
	}
}

// RunMax runs a list of Commands in an Env and quits when they exit, with
// at most max running at a time. Returns true if all successful, false otherwise
// TODO: unify this with Run some more
func RunMax(ctx context.Context, cmds []Command, env *Env, max int) bool {
	// start them all now that we're ready
	var failures bool

	cmdChannel := make(chan Command, 100) // buffer this so we don't block
	for range max {
		go func() {
			for {
				select {
				case cmd := <-cmdChannel:
					uclog.Debugf(ctx, "starting %s/%s(%v)", cmd.Path, cmd.GetName(), cmd.Args)
					if err := cmd.cmd.Start(); err != nil {
						uclog.Errorf(ctx, "failed to start cmd %s/%s(%v) with %v", cmd.Path, cmd.GetName(), cmd.Args, err)
					}
					if err := cmd.cmd.Wait(); err != nil {
						uclog.Errorf(ctx, "cmd %s/%s(%v) failed: %v", cmd.Path, cmd.GetName(), cmd.Args, err)
						failures = true
					}
				case <-env.Context.Done():
					return
				}
			}
		}()
	}

	// feed all the commands into the channel to start running them
	for _, c := range cmds {
		cmdChannel <- c
	}
	uclog.Debugf(ctx, "Executing %d commands across %d worker threads", len(cmds), max)

	// wait for them to terminate then cancel ourselves (shouldn't really be needed with servers)
	go func() {
		env.AllDone.Wait()
		env.Cancel()
	}()

	for {
		select {
		case <-env.Context.Done():
			return !failures
		case b := <-env.Output:
			uclog.Debugf(ctx, "%s", string(b))
		case err := <-env.Error:
			if err != nil {
				uclog.Debugf(ctx, "channel error: %v", err)
			}
		}
	}
}
