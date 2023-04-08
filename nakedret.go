package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/singlechecker"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	pwd = "./"

	DefaultLines = 5
)

func init() {
	//TODO allow build tags
	build.Default.UseAllFiles = true
}

func main() {
	analyzer := NakedReturnAnalyzer(DefaultLines)
	singlechecker.Main(analyzer)
}

func NakedReturnAnalyzer(defaultLines uint) *analysis.Analyzer {
	nakedRet := &NakedReturnRunner{}
	flags := flag.NewFlagSet("nakedret", flag.ExitOnError)
	flags.UintVar(&nakedRet.MaxLength, "l", defaultLines, "maximum number of lines for a naked return function")
	var analyzer = &analysis.Analyzer{
		Name:     "nakedret",
		Doc:      "Checks that functions with naked returns are not longer than a maximum size (can be zero).",
		Run:      nakedRet.run,
		Flags:    *flags,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
	return analyzer
}

type NakedReturnRunner struct {
	MaxLength uint
}

func (n *NakedReturnRunner) run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{ // filter needed nodes: visit only them
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
		(*ast.ReturnStmt)(nil),
	}
	retVis := &returnsVisitor{
		pass:      pass,
		f:         pass.Fset,
		maxLength: n.MaxLength,
	}
	inspector.Nodes(nodeFilter, retVis.NodesVisit)
	return nil, nil
}

type returnsVisitor struct {
	pass      *analysis.Pass
	f         *token.FileSet
	maxLength uint

	stack []funcInfo
}

type funcInfo struct {
	// Details of the function we're currently dealing with
	funcType    *ast.FuncType
	funcName    string
	funcLength  int
	reportNaked bool
}

func checkNakedReturns(args []string, maxLength *uint, setExitStatus bool) error {

	fset := token.NewFileSet()

	files, err := parseInput(args, fset)
	if err != nil {
		return fmt.Errorf("could not parse input: %v", err)
	}

	if maxLength == nil {
		return errors.New("max length nil")
	}

	analyzer := NakedReturnAnalyzer(*maxLength)
	pass := &analysis.Pass{
		Analyzer: analyzer,
		Fset:     fset,
		Files:    files,
		Report: func(d analysis.Diagnostic) {
			log.Printf("%s:%d: %s", fset.Position(d.Pos).Filename, fset.Position(d.Pos).Line, d.Message)
		},
		ResultOf: map[*analysis.Analyzer]interface{}{},
	}
	result, err := inspect.Analyzer.Run(pass)
	if err != nil {
		return err
	}
	pass.ResultOf[inspect.Analyzer] = result

	_, err = analyzer.Run(pass)
	if err != nil {
		return err
	}

	return nil
}

func parseInput(args []string, fset *token.FileSet) ([]*ast.File, error) {
	var directoryList []string
	var fileMode bool
	files := make([]*ast.File, 0)

	if len(args) == 0 {
		directoryList = append(directoryList, pwd)
	} else {
		for _, arg := range args {
			if strings.HasSuffix(arg, "/...") && isDir(arg[:len(arg)-len("/...")]) {

				for _, dirname := range allPackagesInFS(arg) {
					directoryList = append(directoryList, dirname)
				}

			} else if isDir(arg) {
				directoryList = append(directoryList, arg)

			} else if exists(arg) {
				if strings.HasSuffix(arg, ".go") {
					fileMode = true
					f, err := parser.ParseFile(fset, arg, nil, 0)
					if err != nil {
						return nil, err
					}
					files = append(files, f)
				} else {
					return nil, fmt.Errorf("invalid file %v specified", arg)
				}
			} else {

				//TODO clean this up a bit
				imPaths := importPaths([]string{arg})
				for _, importPath := range imPaths {
					pkg, err := build.Import(importPath, ".", 0)
					if err != nil {
						return nil, err
					}
					var stringFiles []string
					stringFiles = append(stringFiles, pkg.GoFiles...)
					// files = append(files, pkg.CgoFiles...)
					stringFiles = append(stringFiles, pkg.TestGoFiles...)
					if pkg.Dir != "." {
						for i, f := range stringFiles {
							stringFiles[i] = filepath.Join(pkg.Dir, f)
						}
					}

					fileMode = true
					for _, stringFile := range stringFiles {
						f, err := parser.ParseFile(fset, stringFile, nil, 0)
						if err != nil {
							return nil, err
						}
						files = append(files, f)
					}

				}
			}
		}
	}

	// if we're not in file mode, then we need to grab each and every package in each directory
	// we can to grab all the files
	if !fileMode {
		for _, fpath := range directoryList {
			pkgs, err := parser.ParseDir(fset, fpath, nil, 0)
			if err != nil {
				return nil, err
			}

			for _, pkg := range pkgs {
				for _, f := range pkg.Files {
					files = append(files, f)
				}
			}
		}
	}

	return files, nil
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func hasNamedReturns(funcType *ast.FuncType) bool {
	if funcType == nil || funcType.Results == nil {
		return false
	}
	for _, field := range funcType.Results.List {
		for _, ident := range field.Names {
			if ident != nil {
				return true
			}
		}
	}
	return false
}

func nestedFuncName(stack []funcInfo) string {
	var r string
	for i, f := range stack {
		if i > 0 {
			r += "."
		}
		r += f.funcName
	}
	return r
}

func (v *returnsVisitor) NodesVisit(node ast.Node, push bool) bool {
	var (
		funcType *ast.FuncType
		funcName string
	)
	switch s := node.(type) {
	case *ast.FuncDecl:
		// We've found a function
		funcType = s.Type
		funcName = s.Name.Name
	case *ast.FuncLit:
		// We've found a function literal
		funcType = s.Type
		file := v.f.File(s.Pos())
		funcName = fmt.Sprintf("<func():%v>", file.Position(s.Pos()).Line)
	case *ast.ReturnStmt:
		// We've found a possibly naked return statement
		fun := v.stack[len(v.stack)-1]
		funName := nestedFuncName(v.stack)
		if fun.reportNaked && len(s.Results) == 0 && push {
			//v.pass.Reportf(s.Pos(), "%v naked returns on %v line function", funName, fun.funcLength)
			v.pass.Reportf(s.Pos(), "naked return in func `%s` with %d lines of code", funName, fun.funcLength)
		}
	}

	if !push {
		if funcType == nil {
			return false
		}
		// Pop function info
		v.stack = v.stack[:len(v.stack)-1]
		return false
	}

	if push && funcType != nil {
		// Push function info to track returns for this function
		file := v.f.File(node.Pos())
		length := file.Position(node.End()).Line - file.Position(node.Pos()).Line
		v.stack = append(v.stack, funcInfo{
			funcType:    funcType,
			funcName:    funcName,
			funcLength:  length,
			reportNaked: uint(length) > v.maxLength && hasNamedReturns(funcType),
		})
	}

	return true
}
