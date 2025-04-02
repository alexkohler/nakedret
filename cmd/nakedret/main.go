package main

import (
	"go/build"

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

	analyzer.Flags.UintVar(&nakedRet.MaxLength, "l", DefaultLines, "maximum number of lines for a naked return function")
	analyzer.Flags.BoolVar(&nakedRet.SkipTestFiles, "skip-test-files", DefaultSkipTestFiles, "set to true to skip test files")

	singlechecker.Main(analyzer)
}
