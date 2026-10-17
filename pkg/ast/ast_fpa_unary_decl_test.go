package ast

import (
	"strings"
	"testing"
)

func TestASTFPAUnaryNegAbsDeclarationsAndApplications(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(sort)
	astTestMust(t, err)
	zero, err := m.MkFamilyApp(family, astFPAPlusZero, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	sortRef, err := m.internal(sort)
	astTestMust(t, err)
	for _, tc := range []struct {
		kind int32
		name string
	}{{astFPANeg, "fp.neg"}, {astFPAAbs, "fp.abs"}} {
		decl, err := m.MkFamilyFuncDecl(family, tc.kind, nil, []ASTHandle{sort}, ASTHandle{})
		astTestMust(t, err)
		r, err := m.internal(decl)
		astTestMust(t, err)
		name, err := m.Symbols.String(m.declAt(r.slot).name)
		astTestMust(t, err)
		if name != tc.name || m.declAt(r.slot).arity != 1 || m.declAt(r.slot).rangeSort != sortRef {
			t.Fatal("FPA unary declaration", tc.kind, name)
		}
		app, err := m.MkFamilyApp(family, tc.kind, nil, []ASTHandle{zero})
		astTestMust(t, err)
		resultSort, err := m.GetSort(app)
		astTestMust(t, err)
		if resultSort != sort {
			t.Fatal("FPA unary application result sort", tc.kind)
		}
		for _, domain := range [][]ASTHandle{nil, {m.external(m.boolSort)}} {
			_, err := m.MkFamilyFuncDecl(family, tc.kind, nil, domain, ASTHandle{})
			if err == nil || !strings.Contains(err.Error(), "floating point") && !strings.Contains(err.Error(), "FloatingPoint") {
				t.Fatal("FPA unary invalid domain", tc.kind, domain, err)
			}
		}
	}
}
