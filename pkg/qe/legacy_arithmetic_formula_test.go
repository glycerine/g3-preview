package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// arith_project_util::operator().

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyArithmeticProjectionCoordinatesFormula(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArithmeticProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	x := f.constant("legacy-formula-x", f.integer)
	zero := f.own(m.ArithmeticIntegerNumeralText("0"))
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	three := f.own(m.ArithmeticIntegerNumeralText("3"))
	lower := f.own(m.MkArithmeticGE(x, zero))
	upper := f.own(m.MkArithmeticLE(x, three))
	formula := f.own(m.MkAnd(lower, upper))
	md, err := model.NewModel(m)
	fmMust(t, err)
	defer func() { fmMust(t, md.Close()) }()
	xDecl, err := m.GetDecl(x)
	fmMust(t, err)
	fmMust(t, md.Core().RegisterConstant(xDecl, one))
	result, err := p.projectFormula(md.Core(), []ast.ASTHandle{x}, formula)
	fmMust(t, err)
	defer func() { fmMust(t, result.Close()) }()
	if len(result.variables) != 0 {
		t.Fatal("projected arithmetic variable retained", len(result.variables))
	}
	handle, err := m.RefGet(result.formula)
	fmMust(t, err)
	contains, err := p.contains(handle, x)
	fmMust(t, err)
	if contains {
		t.Fatal("coordinated formula retains projected variable")
	}

	two := f.own(m.ArithmeticIntegerNumeralText("2"))
	modulus := f.own(m.MkFamilyApp(f.r.arithmetic, 13, nil, []ast.ASTHandle{x, two}))
	modEquality := f.own(m.MkEq(modulus, one))
	modResult, err := p.projectFormula(md.Core(), []ast.ASTHandle{x}, modEquality)
	fmMust(t, err)
	defer func() { fmMust(t, modResult.Close()) }()
	if len(modResult.variables) != 0 {
		t.Fatal("factored mod variables were not projected", len(modResult.variables))
	}
	modHandle, err := m.RefGet(modResult.formula)
	fmMust(t, err)
	contains, err = p.contains(modHandle, x)
	fmMust(t, err)
	if contains {
		t.Fatal("mod-coordinated formula retains projected variable")
	}
}
