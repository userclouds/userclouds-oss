package ucerr

import "userclouds.com/internal/repopath"

func repoRelativeBasePath() string {
	return repopath.BaseDir()
}
