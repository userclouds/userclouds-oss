package multirun

import (
	"io"
	"os/exec"
	"path/filepath"

	"userclouds.com/infra/namespace/color"
)

// Command defines a command to run in parallel with others, inc colorizing output etc
type Command struct {
	Bin   string
	Path  string
	Args  []string
	Env   []string
	Color color.Color

	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// GetName returns the command name based on the binary path
func (c Command) GetName() string {
	return filepath.Base(c.Bin)
}
