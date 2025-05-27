package generate

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"

	"userclouds.com/infra/ucerr"
)

const maxTries = 5

// GetPackage loads the non-test package from the current directory
func GetPackage() *packages.Package {
	return GetPackageForPath(".", true)
}

// GetPackageForPath loads the non-test package for the given path
func GetPackageForPath(p string, importsAndDeps bool) *packages.Package {
	return tryGetPackageForPath(p, importsAndDeps, 0)
}

// since we run this stuff in parallel, we sometimes fail to load packages
// (eg. when one branch is trying to load a package at the same time another
// branch is writing out a file to that package), so this lets us retry a bit.
func tryGetPackageForPath(p string, importsAndDeps bool, tries int) *packages.Package {
	// this relies on go:generate running this binary from the directory in which the directive lives
	// we could always specify a package path separately on the command line if we change eg. codegen system
	cfg := &packages.Config{Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedName}
	if importsAndDeps {
		cfg.Mode = cfg.Mode | packages.NeedImports | packages.NeedDeps
	}

	var pkgs []*packages.Package
	var err error

	pkgs, err = packages.Load(cfg, p)
	if err != nil {
		handleError(p, importsAndDeps, tries, err)
	}

	// technically we can have pkg and pkg_test in the same directory, the real package always seems
	// to come first but not sure that's guaranteed so safer to iterate. Otherwise never >2
	for _, pkg := range pkgs {
		if strings.HasSuffix(pkg.Name, "_test") {
			continue
		}
		if len(pkg.Errors) > 0 {
			handleError(p, importsAndDeps, tries, ucerr.Errorf("package.Errors: %v", pkg.Errors))
		}
		return pkg
	}
	return nil
}

func handleError(p string, importsAndDeps bool, tries int, err error) *packages.Package {
	if tries > maxTries {
		log.Fatalf("error loading packages: %v", err)
	}
	time.Sleep(time.Millisecond * 500)
	return tryGetPackageForPath(p, importsAndDeps, tries+1)
}

// LoadDir loads the directory and parses the files
func LoadDir(ctx context.Context, base string, try int) (*token.FileSet, []*ast.File, error) {
	set := token.NewFileSet()
	packs, err := parser.ParseDir(set, base, nil, parser.ParseComments|parser.AllErrors)
	if err != nil {
		// we likely hit a temporary race in other codegen, so we'll just try again
		if try < 5 {
			time.Sleep(100 * time.Millisecond)
			return LoadDir(ctx, base, try+1)
		}
		return nil, nil, ucerr.Wrap(err)
	}
	allFiles := make([]*ast.File, 0, len(packs)*10)
	for _, pack := range packs {
		for _, file := range pack.Files {
			allFiles = append(allFiles, file)
		}
	}
	return set, allFiles, nil
}
