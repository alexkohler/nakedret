package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
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

	shouldFix bool
	hasNaked  *bool
}

func main() {

	// Remove log timestamp
	log.SetFlags(0)

	maxLength := flag.Uint("l", 5, "maximum number of lines for a naked return function")
	shouldFix := flag.Bool("fix", false, "whether or not the tool should fix the naked returns")
	flag.Usage = usage
	flag.Parse()

	if err := run(flag.Args(), *maxLength, *shouldFix); err != nil {
		log.Fatalf("Encountered an error: %+v", err)
	}
}

func run(args []string, maxLength uint, shouldFix bool) error {
	if len(args) == 0 {
		// We're just going to check for the current directory
		checkRequestedFiles("", maxLength, shouldFix)
		return nil
	}

	for _, arg := range args {
		if strings.HasSuffix(arg, "/...") && isDir(arg[:len(arg)-len("/...")]) {
			checkRequestedFiles(arg, maxLength, shouldFix)
		} else if isDir(arg) {
			checkRequestedFiles(arg, maxLength, shouldFix)
		} else if exists(arg) {
			if strings.HasSuffix(arg, ".go") {
				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, arg, nil, parser.ParseComments)
				if err != nil {
					return err
				}
				err = checkNakedReturns(maxLength, shouldFix, fset, map[string]*ast.File{arg: f})
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("invalid file %v specified", arg)
			}
		} else {
			log.Printf("not sure what you want here\n")
		}
	}

	return nil //files, nil
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func checkNakedReturns(maxLength uint, shouldFix bool, fset *token.FileSet, files map[string]*ast.File) error {
	hasNaked := false
	retVis := &returnsVisitor{
		f:         fset,
		maxLength: maxLength,
		shouldFix: shouldFix,
		hasNaked:  &hasNaked,
	}

	updatedFiles := []*ast.File{}

	for _, f := range files {
		ast.Walk(retVis, f)
		if hasNaked && shouldFix {
			updatedFiles = append(updatedFiles, f)
			hasNaked = false
		}
	}

	if shouldFix {
		for _, f := range updatedFiles {
			file := fset.File(f.Package)
			reportFile := file.Name()

			b := &bytes.Buffer{}
			printer.Fprint(b, fset, f)

			formatted, err := format.Source(b.Bytes())
			if err != nil {
				formatted = b.Bytes()
				log.Printf("format.Source error: %v\n", err)
			}

			err = ioutil.WriteFile(reportFile, formatted, 0644)
			if err != nil {
				log.Printf("ioutil.WriteFile error: %v\n", err)
			}
		}
	}

	return nil
}

func (v *returnsVisitor) Visit(node ast.Node) ast.Visitor {
	var namedReturns []*ast.Ident

	funcDecl, ok := node.(*ast.FuncDecl)
	if !ok {
		return v
	}
	var functionLineLength int
	// We've found a function
	if funcDecl.Type != nil && funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			for _, ident := range field.Names {
				if ident != nil {
					namedReturns = append(namedReturns, ident)
				}
			}
		}
		file := v.f.File(funcDecl.Pos())
		functionLineLength = file.Position(funcDecl.End()).Line - file.Position(funcDecl.Pos()).Line
	}

	if len(namedReturns) > 0 && funcDecl.Body != nil {
		nameExprs := make([]ast.Expr, len(namedReturns))
		for i := range namedReturns {
			nameExprs[i] = namedReturns[i]
		}
		// Scan the body for usage of the named returns
		for _, stmt := range funcDecl.Body.List {

			switch s := stmt.(type) {
			case *ast.ReturnStmt:
				if len(s.Results) == 0 {
					file := v.f.File(s.Pos())
					if file != nil && uint(functionLineLength) > v.maxLength {
						if funcDecl.Name != nil {
							log.Printf("%v:%v %v naked returns on %v line function \n", file.Name(), file.Position(s.Pos()).Line, funcDecl.Name.Name, functionLineLength)

							if v.shouldFix {
								s.Results = nameExprs
								*v.hasNaked = true
							}
						}
					}
					continue
				}

			default:
			}
		}
	}

	return v
}

func checkRequestedFiles(dirName string, maxLength uint, shouldFix bool) {
	path, err := filepath.Abs(dirName)
	if err != nil {
		log.Fatal(err)
	}

	checkAllDirectories(path, maxLength, shouldFix)
}

func checkAllDirectories(path string, maxLength uint, shouldFix bool) {
	_ = filepath.Walk(path, func(directory string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		fset := token.NewFileSet()

		if _, folderName := filepath.Split(directory); folderName == `vendor` {
			return filepath.SkipDir
		}

		allFiles := parseAllGoFilesInDir(directory, fset)

		err = checkNakedReturns(maxLength, shouldFix, fset, allFiles)
		if err != nil {
			log.Printf("checkNakedReturns error: %v\n", err)
			return nil
		}
		return nil
	})
}

func parseAllGoFilesInDir(dir string, fset *token.FileSet) map[string]*ast.File {
	files := map[string]*ast.File{}

	_ = filepath.Walk(dir, func(filename string, info os.FileInfo, err error) error {
		if info == nil {
			return nil
		}
		if info.IsDir() {
			if dir != filename {
				return filepath.SkipDir
			}
			return nil
		}
		if err != nil {
			return err
		}

		if filepath.Ext(filename) != `.go` {
			return nil
		}

		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Printf("ioutil.ReadFile error on %s: %v\n", filename, err)
			return nil
		}

		f, err := parser.ParseFile(fset, filename, bytes, parser.ParseComments)
		if err != nil {
			log.Printf("parser.ParseFile error: %v\n", err)
			return nil
		}

		files[filename] = f
		return nil
	})

	return files
}
