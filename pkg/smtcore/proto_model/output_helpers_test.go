package protomodel

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

type testPairFactory interface {
	SomeValues(ast.ASTHandle, ast.ASTRef, ast.ASTRef) (bool, error)
}

// The test is the caller and owns both outputs for its scope. This helper
// reads them for assertions; no production factory returns borrowed pairs.
func protoTestSomeValues(t *testing.T, m *ast.ASTManager, f testPairFactory, sort ast.ASTHandle) (a, b ast.ASTHandle, ok bool, result error) {
	t.Helper()
	first, err := m.NewRef(ast.ASTHandle{})
	if err != nil {
		return a, b, false, err
	}
	t.Cleanup(func() { protoMust(t, m.DelRef(&first)) })
	second, err := m.NewRef(ast.ASTHandle{})
	if err != nil {
		return a, b, false, err
	}
	t.Cleanup(func() { protoMust(t, m.DelRef(&second)) })
	ok, result = f.SomeValues(sort, first, second)
	var e error
	a, e = m.RefGet(first)
	result = errors.Join(result, e)
	b, e = m.RefGet(second)
	result = errors.Join(result, e)
	return
}

func protoTestOutputRefs(t *testing.T, m *ast.ASTManager) (ast.ASTRef, ast.ASTRef) {
	t.Helper()
	first, err := m.NewRef(ast.ASTHandle{})
	protoMust(t, err)
	t.Cleanup(func() { protoMust(t, m.DelRef(&first)) })
	second, err := m.NewRef(ast.ASTHandle{})
	protoMust(t, err)
	t.Cleanup(func() { protoMust(t, m.DelRef(&second)) })
	return first, second
}
