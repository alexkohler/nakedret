package main

import (
	"go/ast"
	"testing"
)

func TestVisit(t *testing.T) {
	_ = &ast.FuncDecl{
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
			List: []ast.Stmt{},
		},
	}

}
