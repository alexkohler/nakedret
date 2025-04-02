package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/alexkohler/nakedret/v2"
)

const (
	DefaultLines         = 5
	DefaultSkipTestFiles = false
)

func init() {
	// TODO allow build tags
	build.Default.UseAllFiles = true
}

func main() {
	nakedRet := &nakedret.NakedReturnRunner{}

	analyzer := nakedret.NakedReturnAnalyzer(nakedRet)

	analyzer.Flags.Init("nakedret", flag.ExitOnError)

	analyzer.Flags.UintVar(&nakedRet.MaxLength, "l", DefaultLines, "maximum number of lines for a naked return function")
	analyzer.Flags.BoolVar(&nakedRet.SkipTestFiles, "skip-test-files", DefaultSkipTestFiles, "set to true to skip test files")
	analyzer.Flags.Var(versionFlag{}, "V", "print version and exit")

	singlechecker.Main(analyzer)
}

type versionFlag struct{}

func (versionFlag) IsBoolFlag() bool { return true }
func (versionFlag) Get() any         { return nil }
func (versionFlag) String() string   { return "" }
func (versionFlag) Set(s string) error {
	info, ok := debug.ReadBuildInfo()
	if ok {
		fmt.Fprintf(os.Stderr, "%s version %s built with %s (%s/%s)\n",
			filepath.Base(os.Args[0]), info.Main.Version, info.GoVersion, runtime.GOOS, runtime.GOARCH)
	}

	os.Exit(0)
	return nil
}
