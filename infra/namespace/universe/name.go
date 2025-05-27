package universe

import (
	"os"
	"path/filepath"
)

// ServiceName returns the name of the currently-running service.
// TODO: this probably should live in service or something else, but
// that currently creates a cycle so here is fine (and it's not unrelated
// to universe)
func ServiceName() string {
	return filepath.Base(os.Args[0])
}
