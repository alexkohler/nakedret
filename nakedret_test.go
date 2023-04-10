package nakedret

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

type testParams struct {
	filename  string
	maxLength uint
}

var testcases = []struct {
	name     string
	expected string
	params   testParams
}{
	{"return in block",
		"testdata/src/x/ret-in-block.go:9: naked return in func `Dummy` with 8 lines of code\n",
		testParams{
			filename:  "testdata/src/x/ret-in-block.go",
			maxLength: 0,
		}},
	{"ignore short functions", ``, testParams{
		filename:  "testdata/src/x/ret-in-block.go",
		maxLength: 10,
	}},
	{"nested function literals", strings.Join([]string{
		"testdata/src/x/nested.go:16: naked return in func `Bad` with 6 lines of code",
		"testdata/src/x/nested.go:21: naked return in func `BadNested.<func():20>` with 2 lines of code",
		"testdata/src/x/nested.go:28: naked return in func `MoreBad.<func():27>` with 2 lines of code",
		"testdata/src/x/nested.go:32: naked return in func `MoreBad.<func():31>` with 2 lines of code",
		"testdata/src/x/nested.go:36: naked return in func `MoreBad.<func():35>` with 2 lines of code",
		"testdata/src/x/nested.go:40: naked return in func `MoreBad.<func():39>` with 2 lines of code",
		"testdata/src/x/nested.go:47: naked return in func `LiteralFuncCallReturn.<func():46>` with 2 lines of code",
		"testdata/src/x/nested.go:55: naked return in func `LiteralFuncCallReturn2.<func():53>.<func():54>` with 2 lines of code",
		"testdata/src/x/nested.go:63: naked return in func `ManyReturns` with 8 lines of code",
		"testdata/src/x/nested.go:65: naked return in func `ManyReturns` with 8 lines of code",
		"testdata/src/x/nested.go:67: naked return in func `ManyReturns` with 8 lines of code",
		"testdata/src/x/nested.go:78: naked return in func `DeeplyNested.<func():71>.<func():72>.<func():73>.<func():76>` with 3 lines of code",
		"testdata/src/x/nested.go:81: naked return in func `DeeplyNested.<func():71>.<func():72>.<func():73>` with 12 lines of code",
		"testdata/src/x/nested.go:84: naked return in func `DeeplyNested.<func():71>.<func():72>.<func():73>` with 12 lines of code",
		"testdata/src/x/nested.go:87: naked return in func `DeeplyNested.<func():71>` with 17 lines of code",
		"testdata/src/x/nested.go:89: naked return in func `DeeplyNested` with 20 lines of code",
		"testdata/src/x/nested.go:95: naked return in func `<func():92>.<func():94>` with 2 lines of code",
		"testdata/src/x/nested.go:98: naked return in func `<func():92>` with 7 lines of code",
		"testdata/src/x/nested.go:101: naked return in func `SingleLine` with 1 lines of code",
		"testdata/src/x/nested.go:103: naked return in func `<func():103>` with 1 lines of code",
		"testdata/src/x/nested.go:106: naked return in func `SingleLineNested.<func():106>` with 1 lines of code",
		""}, "\n"),
		testParams{
			filename:  "testdata/src/x/nested.go",
			maxLength: 0,
		}},
}

func runNakedret(t *testing.T, filename string, maxLength uint, expected string) {
	t.Helper()
	defer func() {
		// Reset logging
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	}()
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	log.SetFlags(0)

	if err := checkNakedReturns([]string{filename}, &maxLength, false); err != nil {
		t.Fatal(err)
	}
	actual := logBuf.String()
	if expected != actual {
		t.Errorf("Unexpected output:\n-----\ngot: \n%s\nexpected: \n%v\n-----\n", actual, expected)
	}
}

func TestCheckNakedReturns(t *testing.T) {
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			runNakedret(t, tt.params.filename, tt.params.maxLength, tt.expected)
		})
	}
}

func TestAll(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %s", err)
	}

	testdata := filepath.Join(wd, "testdata")
	analysistest.Run(t, testdata, NakedReturnAnalyzer(0), "x")
}

func TestAllFixes(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %s", err)
	}

	testdata := filepath.Join(wd, "testdata")
	analysistest.RunWithSuggestedFixes(t, testdata, NakedReturnAnalyzer(0), "x")
}
