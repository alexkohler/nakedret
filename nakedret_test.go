package main

import (
	"bytes"
	"log"
	"os"
	"testing"
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
	{"return in block", `testdata/ret-in-block.go:9: Dummy naked returns on 8 line function
`, testParams{
		filename:  "testdata/ret-in-block.go",
		maxLength: 0,
	}},
	{"ignore short functions", ``, testParams{
		filename:  "testdata/ret-in-block.go",
		maxLength: 10,
	}},
	{"nested function literals", `testdata/nested.go:16: Bad naked returns on 6 line function
testdata/nested.go:21: <func():20> naked returns on 2 line function
testdata/nested.go:28: <func():27> naked returns on 2 line function
testdata/nested.go:32: <func():31> naked returns on 2 line function
testdata/nested.go:36: <func():35> naked returns on 2 line function
testdata/nested.go:40: <func():39> naked returns on 2 line function
`, testParams{
		filename:  "testdata/nested.go",
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
