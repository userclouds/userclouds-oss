package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate/genconstant"
	"userclouds.com/tools/generate/gendbjson"
	"userclouds.com/tools/generate/genevents"
	"userclouds.com/tools/generate/genhandler"
	"userclouds.com/tools/generate/genopenapi"
	"userclouds.com/tools/generate/genorm"
	"userclouds.com/tools/generate/genpageable"
	"userclouds.com/tools/generate/genrouting"
	"userclouds.com/tools/generate/genschemas"
	"userclouds.com/tools/generate/genstringconstenum"
	"userclouds.com/tools/generate/genvalidate"
)

const (
	numThreads = 8
)

type gencmd struct {
	cmd  string
	path string
	args []string
}

// NB: this is the place you need to add new codegen functions
// if you want to run a single codegen command (for debugging, etc), you can
// build the binary directly (eg. `make bin/genevents`), and run the command
// after `go:generate`, like `genevents idp`, from the path of the file where
// the command lives (eg inside `[repo]/idp/`).
func runGenerator(ctx context.Context, packs map[string]*packages.Package, cmd gencmd) {
	switch cmd.cmd {
	case "genvalidate":
		genvalidate.Run(ctx, packs[cmd.path], cmd.path, cmd.args...)
	case "genpageable":
		genpageable.Run(ctx, packs[cmd.path], cmd.path, cmd.args...)
	case "genorm":
		genorm.Run(ctx, packs[cmd.path], cmd.path, cmd.args...)
	case "genschemas":
		genschemas.Run(ctx, packs[cmd.path], cmd.path, cmd.args...)
	case "genstringconstenum":
		genstringconstenum.Run(ctx, packs[cmd.path], cmd.path, cmd.args...)
	case "gendbjson":
		gendbjson.Run(ctx, packs[cmd.path], cmd.path, cmd.args...)
	case "genconstant":
		genconstant.Run(ctx, packs[cmd.path], cmd.path, cmd.args...)
	case "genevents":
		genevents.Run(ctx, cmd.path, cmd.args[1])
	case "genrouting":
		genrouting.Run(ctx, cmd.path)
	case "genhandler":
		genhandler.Run(ctx, packs[cmd.path], cmd.path, cmd.args...)
	case "genopenapi":
		genopenapi.Run(ctx)
	case "go", "terraform", "easyjson":
		// given that we don't have a build graph, we're going to retry on errors
		maxTries := 3
		for i := range maxTries {
			c := exec.Command(cmd.cmd, cmd.args[1:]...)
			c.Dir = cmd.path
			if output, err := c.Output(); err != nil {
				if i == maxTries-1 {
					uclog.Fatalf(ctx, "error running %s %v: %s (%v)", cmd.cmd, cmd.args, err, string(output))
				}
				uclog.Debugf(ctx, "error running %s %v: %s (%v), retrying", cmd.cmd, cmd.args, err, string(output))
			}
		}
	default:
		// TODO: if this becomes an issue we could certainly run the native command
		// at a perf cost, but probably easier to add a case
		uclog.Fatalf(ctx, "found unexpected command %s ... please add it to parallelgen", cmd.cmd)
	}
}

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "parallelgen")
	defer logtransports.Close()

	// --untracked here makes sure that if you run `make codegen` on new code (eg. untracked files)
	// you'll still get up-to-date results. In test / pre-push hooks, we ensure that we stash first
	// so that it won't create unreproducible results
	//
	// we pass "-c grep.lineNumber=false" to override global config that might have line numbers turned on,
	// which would break our parsing later on
	fb, err := exec.Command("git", "-c", "grep.lineNumber=false", "grep", "--untracked", "//go:generate").Output()
	if err != nil {
		uclog.Fatalf(ctx, "error running grep: %v", err)
	}

	files := strings.Split(strings.TrimSpace(string(fb)), "\n")

	// because we're running everything in goroutines instead of separate processes, we need to
	// operate on absolute paths everywhere in the generation functions
	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "error getting working directory: %v", err)
	}

	// first we load all of our packages at once ... this is currently order of 6x faster than
	// loading package-by-package on demand
	packs := loadPackages(ctx, wd)

	// set up all of our generate commands that we grepped for
	cmds := setupCommands(ctx, wd, files)

	// set up our worker pool
	var allDone sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	cmdChannel := make(chan gencmd, 100) // buffer this so we don't block
	for range numThreads {
		go func() {
			for {
				select {
				case cmd := <-cmdChannel:
					runGenerator(ctx, packs, cmd)
					allDone.Done()
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// feed all the commands into the channel to start running them
	for _, c := range cmds {
		allDone.Add(1)
		cmdChannel <- c
	}
	uclog.Debugf(ctx, "Executing %d commands across %d worker threads", len(cmds), numThreads)

	// wait for them to terminate then cancel ourselves (shouldn't really be needed with servers)
	go func() {
		allDone.Wait()
		cancel()
	}()

	// clean everything up
	<-ctx.Done()
}

func setupCommands(ctx context.Context, wd string, files []string) []gencmd {
	cmds := []gencmd{{
		cmd: "genopenapi",
	}}
	for _, f := range files {
		parts := strings.Split(f, ":")

		// ignore ourselves
		if parts[0] == "cmd/parallelgen/main.go" {
			uclog.Verbosef(ctx, "Skipping cmd/parallelgen/main.go from file list")
			continue
		}

		// ignore non-go files (eg README, Makefile, etc for docs)
		if !strings.HasSuffix(parts[0], ".go") {
			uclog.Verbosef(ctx, "Skipping non-go file %s", parts[0])
			continue
		}

		if len(parts) != 3 || parts[1] != "//go" {
			uclog.Fatalf(ctx, "unexpected formatting in grep, check cmd/parallelgen/main.go: %v", parts)
		}

		path := parts[0][:strings.LastIndex(parts[0], "/")]
		path = filepath.Join(wd, path)

		commandLine := strings.TrimPrefix(parts[2], "generate ")

		if strings.Contains(commandLine, `"`) {
			uclog.Fatalf(ctx, "TODO: go:generate commands that included quoted params aren't yet supported in parallelgen")
		}

		cliParts := strings.Split(commandLine, ` `)
		cmds = append(cmds, gencmd{
			cmd:  cliParts[0],
			args: cliParts,
			path: path,
		})
	}

	return cmds
}

func loadPackages(ctx context.Context, wd string) map[string]*packages.Package {
	var packs = map[string]*packages.Package{}

	// TODO: keep this in sync with generate.GetPackage()
	cfg := &packages.Config{Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedName | packages.NeedImports | packages.NeedDeps}
	ps, err := packages.Load(cfg, "./...")
	if err != nil {
		uclog.Fatalf(ctx, "error loading packages: %v", err)
	}

	// stuff all of the packages into a map by absolute path so we can pass
	// the right one to the generation functions
	// TODO: in the future we might want to pass the whole map (too?) so that generators
	// can load other packages (right now this is only used in genorm for non-current-package models)
	for _, p := range ps {
		// we know that our current working directory is the root of the "userclouds.com" package
		// TODO: we could actually check this vs go.mod and error if not true?
		pkgPath := strings.Replace(p.PkgPath, "userclouds.com/", "", 1)
		pkgPath = filepath.Join(wd, pkgPath)
		packs[pkgPath] = p
	}

	return packs
}
