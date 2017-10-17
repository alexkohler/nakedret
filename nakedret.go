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
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	pwd = "./"
)

func init() {
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

type zoop uint32

type theStruct struct {
	theField uint32
}

func red() theStruct {

	_ = net.IPConn{}

	myStruct := theStruct{
		theField: uint32(5),
	}
	//
	//
	//
	//
	//
	//
	return myStruct //
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// ?https://stackoverflow.com/questions/24118011/how-can-i-get-all-struct-under-a-package-in-golang
func (v *returnsVisitor) Visit(node ast.Node) ast.Visitor {
	var namedReturns []*ast.Ident

	funcDecl, ok := node.(*ast.FuncDecl)
	if !ok {
		return v
	}

	// can we generate a temp file with an init registry?
	// https://play.golang.org/p/rgxTlJhMrq
	fmt.Println(funcDecl.Name.Name)
	for _, stmt := range funcDecl.Body.List {
		fmt.Printf("     %T\n", stmt)
		//TODO how do we know it's a struct?
		asgnStmt, ok := stmt.(*ast.AssignStmt)
		if ok {
			for _, expr := range asgnStmt.Rhs {
				fmt.Printf("	%T\n", expr)
				cmpLit, ok := expr.(*ast.CompositeLit)
				if ok {
					fmt.Printf("TYPE %T\n", cmpLit.Type)
					iiiiii, ok := cmpLit.Type.(*ast.Ident)
					if ok {
						fmt.Printf("@@@@@@@@@@@@@@@@@@@@@@22tiddddle %v\n", iiiiii.Name) // this is where we have struct name which I think we can reflect belowx
						v := reflect.ValueOf(node).Elem()
						for i, n := 0, v.NumField(); i < n; i++ {
							 fmt.Printf("%T\n", v.Field(i).Addr().Interface())
							//switch s := v.Field(i).Addr().Interface().(type) {
							//case **ast.FieldList:
							//	fmt.Println("shhhhhhhhhhhhhheeeeeeeeeeeeeeeeeeeeet")
								// dref1 := *s
								// dref2 := *dref1
								// for _, zoop := range dref2.List {
								// fmt.Println(zoop)
								// }
							}
						}
					}

					// Range through composite elements
					for _, cmpEle := range cmpLit.Elts {
						kv, ok := cmpEle.(*ast.KeyValueExpr)
						if ok {
							// key is ident
							keyIdent, ok := kv.Key.(*ast.Ident)
							if ok {
								fmt.Printf("ball hog %v\n", keyIdent.Name)
								// another ident with a name here... we need some definition of the struct.
							}

							// val is callexpr
							// fmt.Printf("val %T\n", kv.Value)
							possibleCast, ok := kv.Value.(*ast.CallExpr)
							if ok {
								// we have an ident here
								_, ok := possibleCast.Fun.(*ast.Ident)
								if ok {
									//TODO nil check on obj
									// fmt.Printf(" sssss	%v\n", wtfIdent.Name)
									// primitive cast - uint32
									// SO what we need to figure out is how to map the primitive name to the field type of the struct
									// to see if we have a redundant cast

									// no cast - we won't get here

									// ast.BasicLit means there was no cast

									// looks like typedefs will be ast.Object. this will probably be harder. expand on this
								}
							}
						}
					}
				}
			}
		}
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
		// Scan the body for usage of the named returns
		for _, stmt := range funcDecl.Body.List {

			switch s := stmt.(type) {
			case *ast.ReturnStmt:
				if len(s.Results) == 0 {
					file := v.f.File(s.Pos())
					if file != nil && uint(functionLineLength) > v.maxLength {
						if funcDecl.Name != nil {
							log.Printf("%v:%v %v naked returns on %v line function \n", file.Name(), file.Position(s.Pos()).Line, funcDecl.Name.Name, functionLineLength)
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
