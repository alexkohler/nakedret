package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/alexkohler/nakedret/v2"
)

const (
	DefaultLines = 5
)

func main() {
	analyzer := nakedret.NakedReturnAnalyzer(DefaultLines)
	singlechecker.Main(analyzer)
}
