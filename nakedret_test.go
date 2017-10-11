package main

import (
	"go/ast"
	"testing"
)

type MockFileSet struct {
}

func TestVisit(t *testing.T) {

	retVis := &returnsVisitor{
		f:         MockFileSet{},
		maxLength: uint(3),
	}

	node := &ast.FuncDecl{
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{
							{
								Name: "namedReturn1",
							},
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						//TODO check what type this should actually be
						&ast.CallExpr{},
					},
				},
			},
		},
	}

	//TODO need to mock out fileset to return a mock file
	_ = retVis.f.Position(node.End()).Line - file.Position(node.Pos()).Line

	retVis.Visit(node)

}
