package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPASMTEqualityNaNsSignedZerosAndSymbolicRefusal(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	sort, err := m.MkFamilySort(rw.family, 3, nil)
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(sort)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&parameter)) }()
	value := func(kind int32) ast.ASTHandle {
		h, e := m.MkFamilyApp(rw.family, kind, []ast.ASTParameter{parameter}, nil)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	nan, pinf, pzero, nzero := value(8), value(6), value(9), value(10)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	trueTerm, err := m.MkTrue()
	astTestMust(t, err)
	falseTerm, err := m.MkFalse()
	astTestMust(t, err)
	out := astTestPin(t, m, falseTerm)
	for _, tc := range []struct {
		left, right ast.ASTHandle
		status      RewriteStatus
		want        ast.ASTHandle
	}{
		{nan, nan, RewriteDone, trueTerm},
		{nan, pinf, RewriteDone, falseTerm},
		{pzero, nzero, RewriteDone, falseTerm},
		{pzero, pzero, RewriteDone, trueTerm},
		{pinf, pinf, RewriteDone, trueTerm},
		{variable, nan, RewriteFailed, trueTerm},
		{nan, variable, RewriteFailed, trueTerm},
	} {
		astTestMust(t, m.SetRef(out, trueTerm))
		status, err := rw.MkEqCore(tc.left, tc.right, out)
		astTestMust(t, err)
		got, err := m.RefGet(out)
		astTestMust(t, err)
		if status != tc.status || got != tc.want {
			t.Fatalf("SMT equality: status=%v got=%v want=%v", status, got, tc.want)
		}
	}
}
