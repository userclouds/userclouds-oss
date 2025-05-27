package repopath

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var baseDir string
var baseDirOnce sync.Once

// BaseDir returns the base directory of the userclouds repo, eg for
// /Users/kutta.s/go/src/userclouds/userclouds/infra/uclog, it would return /Users/kutta.s/go/src/userclouds/userclouds
// /Users/sgarrity/projects/userclouds/internal/repopath, it would return /Users/sgarrity/projects/userclouds
// Note there is no trailing slash
func BaseDir() string {

	baseDirOnce.Do(func() {
		// we choose the last case of `userclouds` rather than the first, since we can "guarantee"
		// that we don't nest userclouds inside our repo, but not outside (eg @kutta.s has org/repo nested)
		// TODO: there's probably a more elegant / less fragile way to do this?
		wd, err := os.Getwd()
		if err != nil {
			// I don't love this panic here, but we can't use uclog because of import cycles (yet),
			// and we can't easily return a wrapped error (since this is used in ucerr)
			// But I guess if we can't get a working directory, we're pretty much screwed anyway
			panic(fmt.Sprintf("error getting working directory: %v", err))
		}
		dirs := strings.Split(wd, "/")
		var last int
		for i, d := range dirs {
			if d == "userclouds" {
				last = i
			}
		}
		baseDir = strings.Join(dirs[:last+1], "/")
	})
	return baseDir
}
