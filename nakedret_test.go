package main

import (
	"bytes"
	"log"
	"os"
	"testing"
)

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

	if err := checkNakedReturns([]string{filename}, &maxLength); err != nil {
		t.Fatal(err)
	}
	actual := string(logBuf.Bytes())
	if expected != actual {
		t.Errorf("Unexpected output:\n-----\n%s\n-----\n", actual)
	}
}

func TestReturnInBlock(t *testing.T) {
	expected := `testdata/ret-in-block.go:9: Dummy naked returns on 8 line function
`
	runNakedret(t, "testdata/ret-in-block.go", 0, expected)
}

func TestIgnoreShortFunctions(t *testing.T) {
	runNakedret(t, "testdata/ret-in-block.go", 10, "")
}

func TestNestedFunctionLiterals(t *testing.T) {
	expected := `testdata/nested.go:16: Bad naked returns on 6 line function
testdata/nested.go:21: <func():20> naked returns on 2 line function
`
	runNakedret(t, "testdata/nested.go", 0, expected)
}
