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

	"golang.org/x/tools/go/ssa"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa/ssautil"
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

type f struct{}

func (f) Lol() {}

func checkNakedReturns(args []string, maxLength *uint) error {

	myF := f{}
	myF.Lol()

	fset := token.NewFileSet()

	files, err := parseInput(args, fset)
	if err != nil {
		return fmt.Errorf("could not parse input %v", err)
	}

	if maxLength == nil {
		return errors.New("max length nil")
	}

	// Load, parse, and type-check the whole program.
	cfg := packages.Config{Mode: packages.LoadAllSyntax}
	//TODO - don't hardcode package
	initial, err := packages.Load(&cfg, "_/home/alex/workspace/nakedret") // this package needs to be imported
	if err != nil {
		log.Fatal(err)
	}

	// Create SSA packages for well-typed packages and their dependencies.
	ssaProg, _ := ssautil.AllPackages(initial, 0)

	ssaProg.Build()

	functionsMap := ssautil.AllFunctions(ssaProg)

	fmt.Println(len(functionsMap))

	nameMap := make(map[string]*ssa.Function)
	for f := range functionsMap {
		if f.Signature.Results().Len() > 0 {
			fmt.Printf("funcName %v results %v\n", f.String(), f.Signature.Results().Underlying().String())
		} else {
			fmt.Printf("funcName %v results none\n", f.String())
		}
		nameMap[f.String()] = f
		// if i == 5 {
		// 	break
		// }
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

// Other ideas - see if you can see where returns are being ignored i.e. client.Close() should be _ = client.Close()
// AssignStmt - LHS is empty?

//TODO - could also look for methods with receivers that don't actually use the receiver? eh

func (v *returnsVisitor) Visit(node ast.Node) ast.Visitor {

	// os.Stat("hihiihihihi")

	// // search for call expressions
	// assignStmt, ok := node.(*ast.AssignStmt)
	// if !ok {
	// 	return v
	// }

	// file := v.f.File(assignStmt.Pos())
	// // fmt.Printf("%v:%v got one %T\n", file.Name(), file.Position(assignStmt.Pos()).Line, assignStmt.Fun)

	// fmt.Printf("%v:%v got one\n", file.Name(), file.Position(assignStmt.Pos()).Line)
	// fmt.Printf("I have a LHS assignment with size %+v\n", len(assignStmt.Lhs))

	// Next up is to check if there are any receivers

	// search for call expressions
	callExpr, ok := node.(*ast.CallExpr)
	if !ok {
		return v
	}

	// file := v.f.File(callExpr.Pos())

	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return v
	}

	// if selExpr.Sel != nil {
	// 	selIden, ok := selExpr.Sel.
	// }

	ident, ok := selExpr.X.(*ast.Ident)
	if !ok {
		return v
	}

	if selExpr.Sel != nil {
		// fmt.Printf("@@@ nonFunction qualifier %v.%v\n", ident.Name, selExpr.Sel.Name)
	} else {
		panic("haha i am in danger")
	}

	if ident.Obj != nil {
		// receiverName := ident.Obj.Name

		// fmt.Printf("~~ I have a receiver with name %v.%v\n", ident.Name, selExpr.Sel.Name)

		// fmt.Println(receiverName)
		// Next up is to check if there are any receivers
	}

	return v
}
