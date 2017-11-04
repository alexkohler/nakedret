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

//TODO ensure there are prellocations in the make - (type, size, cap)
// then see the append lines up with the preallocate
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

func (v *returnsVisitor) Visit(node ast.Node) ast.Visitor {

	var makes []string

	switch n := node.(type) {
	case *ast.FuncDecl:
		// var foundMake bool // this will need to be a list
		if n.Body != nil {
			for _, stmt := range n.Body.List {
				switch s := stmt.(type) {
				case *ast.AssignStmt:
					// loop through assignment to determine if it's a make
					for _, expr := range s.Rhs {
						callExpr, ok := expr.(*ast.CallExpr)
						if !ok {
							continue // should this be break?
						}
						ident, ok := callExpr.Fun.(*ast.Ident)
						if !ok {
							continue
						}
						if ident.Name == "make" {
							// check callExpr args
							for _, arg := range callExpr.Args {
								// we only want to suggest this for maps, not slices - this may be caught by just using append
								_, ok := arg.(*ast.ArrayType)
								if !ok {
									continue
								}
								//assign the fact that we have a slice here

								// get the name of the struct being made - TODO support double declarations?
								if len(s.Lhs) == 1 {
									// makes  = append(makes, s.Lhs[0])
									lhsIdent, ok := s.Lhs[0].(*ast.Ident)
									if !ok {
										continue
									}
									makes = append(makes, lhsIdent.Name)
								} else if len(s.Lhs) > 1 {
									fmt.Println("@@@@@@@@@@@@@@@@@@@@@@@@@ wat lhs > 1")
								}

								// *********** we have a make with a slice inside, now we need to see if we have a for loop

							}
						}
					}

				case *ast.RangeStmt: // for statement should literally duplicate this
					if len(makes) == 0 {
						continue
					}
					if s.Body != nil {
						for _, stmt := range s.Body.List {
							asgnStmt, ok := stmt.(*ast.AssignStmt)
							if !ok {
								continue
							}
							for _, expr := range asgnStmt.Rhs {
								callExpr, ok := expr.(*ast.CallExpr)
								if !ok {
									continue // should this be break? comes back to multi-call support I think
								}
								ident, ok := callExpr.Fun.(*ast.Ident)
								if !ok {
									continue
								}
								if ident.Name == "append" {
									file := v.f.File(ident.Pos())
									lineNumber := file.Position(ident.Pos()).Line
									fmt.Printf("%v:%v Consider indexing instead of appending here\n", file.Name(), lineNumber)
								}
							}

						}
					}

				default:
				}
			}
		}

	default:
		return v

	}

	// We've found a function

	return v
}
