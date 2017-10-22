package main

// Problems - dealing with embedded structs
// actually finding the underlying type

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
	"os/exec"
	"path/filepath"
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
	f              *token.FileSet
	maxLength      uint
	currentPackage string
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
		// nil check?
		retVis.currentPackage = f.Name.Name
		ast.Walk(retVis, f)
	}

	retVis.Process()

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
	theField  uint32
	nada1     uint32
	theField2 string
	nada2     string
	zz        zoop
}

//TODO need to catch these types of struct declarations as well
var ZOOPER = theStruct{
	theField: uint32(6),
}

func red() theStruct {

	_ = net.IPConn{}

	myStruct := theStruct{
		theField:  uint32(5),
		nada1:     5,
		theField2: string("hi"),
		nada2:     "kjdp",
		zz:        zoop(5),
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

// maps package name to map of struct name to fields
var pkgRegistry = make(map[string]map[string][]fieldInfo)

type fieldInfo struct {
	name  string
	fType string
}

// ?https://stackoverflow.com/questions/24118011/how-can-i-get-all-struct-under-a-package-in-golang
func (v *returnsVisitor) Visit(node ast.Node) ast.Visitor {

	// I think we still need a registry....
	/*if reflect.TypeOf(node) != nil && reflect.TypeOf(node).Kind() == reflect.Ptr {
		vv := reflect.ValueOf(node).Elem()
		for i, n := 0, vv.NumField(); i < n; i++ {
			fmt.Printf("grrgrgrgrgrggrg %v\n", vv.Field(i).Type().Name())
		}
	}
	return v*/

	var namedReturns []*ast.Ident

	funcDecl, ok := node.(*ast.FuncDecl)
	if !ok {
		if node != nil {
			file := v.f.File(node.Pos())
			functionLineLength := file.Position(node.End()).Line - file.Position(node.Pos()).Line
			fmt.Printf("%T %v\n", node, functionLineLength)
		}
		return v
	}

	// TYPE 1 - searching for structs inside of functions
	// https://play.golang.org/p/rgxTlJhMrq
	for _, stmt := range funcDecl.Body.List {
		// fmt.Printf("     %T\n", stmt)
		//TODO how do we know it's a struct?
		asgnStmt, ok := stmt.(*ast.AssignStmt)
		if ok {
			for _, expr := range asgnStmt.Rhs {
				// fmt.Printf("	%T\n", expr)
				cmpLit, ok := expr.(*ast.CompositeLit)
				if ok {
					// fmt.Printf("TYPE %T\n", cmpLit.Type)
					structName, ok := cmpLit.Type.(*ast.Ident)
					if ok {
						// Range through composite elements
						for _, cmpEle := range cmpLit.Elts {
							kv, ok := cmpEle.(*ast.KeyValueExpr)
							if ok {
								// key is ident
								keyIdent, ok := kv.Key.(*ast.Ident)
								if ok {
									// fmt.Printf("ball hog %v\n", keyIdent.Name)
									// another ident with a name here... we need some definition of the struct.
									//****************** WE NEED TO FIGURE OUT HOW THEFIELD IS A UINT32
									// I don't think there's a reliable way to do this without a registry.

									// val is callexpr
									// fmt.Printf("val %T\n", kv.Value)
									possibleCast, ok := kv.Value.(*ast.CallExpr)
									if ok {
										// we need to find the underlying type

										for _, arg := range possibleCast.Args {
											fmt.Printf("posty %T\n", arg)
											// basic lit for builtin casts
											bl, ok := arg.(*ast.BasicLit)
											if ok {
												fmt.Println("	" + bl.Kind.String())
											}

											///ast.Ident for typedef casts
										}
										// we have an ident here
										valueIdent, ok := possibleCast.Fun.(*ast.Ident)
										if ok {

											//TODO nil check on obj
											fmt.Printf(" WINFO	%v %v %v\n", structName.Name, keyIdent.Name, valueIdent.Name) // hello i am a uint32 part of kv struct

											//valueIdent.Name is the cast, but we need to find what the underlying type on that cast is

											fInfo := fieldInfo{
												name:  keyIdent.Name,
												fType: valueIdent.Name,
											}

											structRegistry, ok := pkgRegistry[v.currentPackage]
											if !ok {
												pkgRegistry[v.currentPackage] = make(map[string][]fieldInfo)
												structRegistry = pkgRegistry[v.currentPackage]
											}

											structRegistry[structName.Name] = append(structRegistry[structName.Name], fInfo)

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

func (v *returnsVisitor) Process() {

	/*

		//TODO later optimize with go

		//TODO need to handle on a per package basis

		// what we need here is to pass all the information we know - (field names and field types)
		// and then compare them against what is in the registry. I think we can take a variety of approaches here
		// (more complex object inside of map), ugly ifs hardcoded (nty)

		// we should also move this outside the loop
		src := []byte(`
			package hw

			import (
			"fmt"
			"reflect"
			)

			type astField struct {
				name string
				fType string
			}

			type structType struct {
					rType reflect.Type
					aField []astField
			}

			var typeRegistry = make(map[string]structType)

			func init() {
				typeRegistry["theStruct"] = structType{rType: reflect.TypeOf(theStruct{})}
			}

			func Zoop() {
				fmt.Println("hello world")
			}
			`)
		f, err := os.Create("src/" + v.currentPackage + "/test.go")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		//TODO defer removing the file

		if _, err := f.Write(src); err != nil {
			panic(err)
		}

		// build out this string with current package
		out, err := exec.Command("sh", "-c", "gorram "+v.currentPackage+" Zoop").Output()
		if err != nil {
			panic(err)
		}

		fmt.Println(string(out))




	*/

	//TODO we'll probably hav to run goimports
	for pkg, structRegistry := range pkgRegistry {
		f, err := os.Create("src/" + v.currentPackage + "/test.go")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		/*
					func init() {
				typeRegistry["theStruct"] = //structType{rType: reflect.TypeOf(theStruct{}), aField:[]astField{name:"hi",fType:"yO"}}
			}

			func Zoop() {
				fmt.Println("hello world")
			}
		*/
		//TODO defer removing the file (either hack a defer by using an inline function or explictly remove it)
		src := `package ` + pkg + `

		import (
			"fmt"
			"reflect"
			)

			type astField struct {
				name string
				fType string
			}

			type structType struct {
					rType reflect.Type
					aField []astField
			}

			var typeRegistry = make(map[string]string)
			func Zoop() {


		`
		//TODO need to come up with unique names
		for structName, fieldInfo := range structRegistry {
			for _, field := range fieldInfo {
				src = src + field.name + `, ok := reflect.TypeOf(` + structName + `{}).FieldByName("` + field.name + `")` + `
				if ok &&` + field.name + `.Type.Name() == "` + field.fType + `"{` + ` 
					fmt.Println("wobbie")
				}
				`
			}

		}
		src = src + `}`
		if _, err := f.Write([]byte(src)); err != nil {
			panic(err)
		}

		// build out this string with current package
		out, err := exec.Command("sh", "-c", "gorram "+v.currentPackage+" Zoop").Output()
		if err != nil {
			panic(err)
		}

		fmt.Println(string(out))
	}

}
