// Source: src/ast/datatype_decl_plugin.cpp:193-381;
// src/ast/datatype_decl_plugin.h:24-38,237-288.
package ast

import "testing"

func TestASTDatatypeSortAndOriginalDeclarations(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)

	name := astTestSymbol(t, m, "Box")
	nameParameter, err := m.NewParameterSymbol(name)
	astTestMust(t, err)
	dt, err := m.MkFamilySort(family, ASTDatatypeSort, []ASTParameter{nameParameter})
	astTestMust(t, err)
	astTestPin(t, m, dt)
	astTestMust(t, m.DelParameter(&nameParameter))
	if got, err := m.GetFamilyID(dt); err != nil || got != family {
		t.Fatal("datatype sort family", got, err)
	}

	constructorName := astTestSymbol(t, m, "box")
	constructorParameter, err := m.NewParameterSymbol(constructorName)
	astTestMust(t, err)
	constructor, err := m.MkFamilyFuncDecl(family, ASTDatatypeConstructor, []ASTParameter{constructorParameter}, []ASTHandle{boolSort}, dt)
	astTestMust(t, err)
	astTestPin(t, m, constructor)
	astTestMust(t, m.DelParameter(&constructorParameter))

	accessorName := astTestSymbol(t, m, "unbox")
	accessorNameParameter, err := m.NewParameterSymbol(accessorName)
	astTestMust(t, err)
	constructorNameParameter, err := m.NewParameterSymbol(constructorName)
	astTestMust(t, err)
	accessor, err := m.MkFamilyFuncDecl(family, ASTDatatypeAccessor, []ASTParameter{accessorNameParameter, constructorNameParameter}, []ASTHandle{dt}, boolSort)
	astTestMust(t, err)
	astTestPin(t, m, accessor)
	astTestMust(t, m.DelParameter(&constructorNameParameter))
	astTestMust(t, m.DelParameter(&accessorNameParameter))

	constructorASTParameter, err := m.NewParameterAST(constructor)
	astTestMust(t, err)
	recognizerName := astTestSymbol(t, m, "is-box")
	recognizerNameParameter, err := m.NewParameterSymbol(recognizerName)
	astTestMust(t, err)
	recognizer, err := m.MkFamilyFuncDecl(family, ASTDatatypeRecognizer, []ASTParameter{constructorASTParameter, recognizerNameParameter}, []ASTHandle{dt}, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, recognizer)
	astTestMust(t, m.DelParameter(&recognizerNameParameter))

	isDecl, err := m.MkFamilyFuncDecl(family, ASTDatatypeIs, []ASTParameter{constructorASTParameter}, []ASTHandle{dt}, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, isDecl)
	astTestMust(t, m.DelParameter(&constructorASTParameter))

	accessorASTParameter, err := m.NewParameterAST(accessor)
	astTestMust(t, err)
	update, err := m.MkFamilyFuncDecl(family, ASTDatatypeUpdateField, []ASTParameter{accessorASTParameter}, []ASTHandle{dt, boolSort}, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, update)
	astTestMust(t, m.DelParameter(&accessorASTParameter))

	for _, row := range []struct {
		decl      ASTHandle
		kind      int32
		rangeSort ASTHandle
	}{{constructor, ASTDatatypeConstructor, dt}, {accessor, ASTDatatypeAccessor, boolSort}, {recognizer, ASTDatatypeRecognizer, boolSort}, {isDecl, ASTDatatypeIs, boolSort}, {update, ASTDatatypeUpdateField, dt}} {
		kind, e := m.GetDeclKind(row.decl)
		astTestMust(t, e)
		rangeSort, e := m.GetRange(row.decl)
		astTestMust(t, e)
		if kind != row.kind || rangeSort != row.rangeSort {
			t.Fatal("declaration identity", kind, rangeSort, row)
		}
	}

	truth, err := m.MkTrue()
	astTestMust(t, err)
	boxed, err := m.MkApp(constructor, []ASTHandle{truth})
	astTestMust(t, err)
	astTestPin(t, m, boxed)
	read, err := m.MkApp(accessor, []ASTHandle{boxed})
	astTestMust(t, err)
	astTestPin(t, m, read)
	if sort, err := m.GetSort(read); err != nil || sort != boolSort {
		t.Fatal("accessor application sort", sort, err)
	}
	updated, err := m.MkApp(update, []ASTHandle{boxed, truth})
	astTestMust(t, err)
	astTestPin(t, m, updated)
	if sort, err := m.GetSort(updated); err != nil || sort != dt {
		t.Fatal("update application sort", sort, err)
	}
}

func TestASTDatatypeDeclarationRefusals(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	name, err := m.NewParameterSymbol(astTestSymbol(t, m, "D"))
	astTestMust(t, err)
	dt, err := m.MkFamilySort(family, ASTDatatypeSort, []ASTParameter{name})
	astTestMust(t, err)
	astTestPin(t, m, dt)
	astTestMust(t, m.DelParameter(&name))

	constructorName, err := m.NewParameterSymbol(astTestSymbol(t, m, "d"))
	astTestMust(t, err)
	if _, err = m.MkFamilyFuncDecl(family, ASTDatatypeConstructor, []ASTParameter{constructorName}, nil, boolSort); err == nil {
		t.Fatal("constructor accepted non-datatype range")
	}
	constructor, err := m.MkFamilyFuncDecl(family, ASTDatatypeConstructor, []ASTParameter{constructorName}, nil, dt)
	astTestMust(t, err)
	astTestPin(t, m, constructor)
	astTestMust(t, m.DelParameter(&constructorName))

	constructorParameter, err := m.NewParameterAST(constructor)
	astTestMust(t, err)
	if _, err = m.MkFamilyFuncDecl(family, ASTDatatypeIs, []ASTParameter{constructorParameter}, []ASTHandle{boolSort}, ASTHandle{}); err == nil {
		t.Fatal("recognizer accepted non-datatype domain")
	}
	astTestMust(t, m.DelParameter(&constructorParameter))
}
