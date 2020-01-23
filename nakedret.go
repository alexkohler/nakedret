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
)

const (
	pwd = "./"
)

func init() {
	//TODO allow build tags
	build.Default.UseAllFiles = true
}

func usage() {
	log.Printf("Usage of %s:\n", os.Args[0])
	log.Printf("\nnakedret [flags] # runs on package in current directory\n")
	log.Printf("\nnakedret [flags] [packages]\n")
	log.Printf("Flags:\n")
	flag.PrintDefaults()
}

type returnsVisitor struct {
	f         *token.FileSet
	maxLength uint

	// Details of the function we're currently dealing with
	funcName    string
	funcLength  int
	reportNaked bool
}

func main() {

	// Remove log timestamp
	log.SetFlags(0)

	maxLength := flag.Uint("l", 5, "maximum number of lines for a naked return function")
	flag.Usage = usage
	flag.Parse()

	if err := checkNakedReturns(flag.Args(), maxLength); err != nil {
		log.Println(err)
	}
}

func checkNakedReturns(args []string, maxLength *uint) error {

	fset := token.NewFileSet()

	files, err := parseInput(args, fset)
	if err != nil {
		return fmt.Errorf("could not parse input %v", err)
	}

	if maxLength == nil {
		return errors.New("max length nil")
	}

	retVis := &returnsVisitor{
		f:         fset,
		maxLength: *maxLength,
	}

	for _, f := range files {
		ast.Walk(retVis, f)
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

func (v *returnsVisitor) Visit(node ast.Node) ast.Visitor {
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
		if v.reportNaked && len(s.Results) == 0 {
			file := v.f.File(s.Pos())
			log.Printf("%v:%v %v naked returns on %v line function\n", file.Name(), file.Position(s.Pos()).Line, v.funcName, v.funcLength)
		}
	}

	if funcType != nil {
		// Create a new visitor to track returns for this function
		file := v.f.File(node.Pos())
		length := file.Position(node.End()).Line - file.Position(node.Pos()).Line
		return &returnsVisitor{
			f:           v.f,
			maxLength:   v.maxLength,
			funcName:    funcName,
			funcLength:  length,
			reportNaked: uint(length) > v.maxLength && hasNamedReturns(funcType),
		}
	}

	return v
}
