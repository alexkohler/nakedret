package main

import (
	"go/build"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/alexkohler/nakedret/v2"
)

const (
	DefaultLines = 5
)

func init() {
	// TODO allow build tags
	build.Default.UseAllFiles = true
}

func main() {
	analyzer := nakedret.NakedReturnAnalyzer(DefaultLines)
	singlechecker.Main(analyzer)
}
