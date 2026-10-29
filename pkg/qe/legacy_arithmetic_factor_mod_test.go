package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// arith_project_util::factor_mod_terms.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyArithmeticProjectionFactorsSharedModTerms(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArithmeticProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	x := f.constant("legacy-factor-mod-x", f.integer)
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	three := f.own(m.ArithmeticIntegerNumeralText("3"))
	four := f.own(m.ArithmeticIntegerNumeralText("4"))
	sum := f.own(m.MkArithmeticAdd(x, one))
	modulus := f.own(m.MkFamilyApp(f.r.arithmetic, 13, nil, []ast.ASTHandle{sum, three}))
	formula := f.own(m.MkEq(modulus, modulus))
	md, err := model.NewModel(m)
	fmMust(t, err)
	defer func() { fmMust(t, md.Close()) }()
	xDecl, err := m.GetDecl(x)
	fmMust(t, err)
	fmMust(t, md.Core().RegisterConstant(xDecl, four))
	result, err := p.factorModTerms(md.Core(), formula)
	fmMust(t, err)
	defer func() { fmMust(t, result.Close()) }()
	if len(result.variables) != 1 {
		t.Fatal("shared mod term fresh-variable count", len(result.variables))
	}
	variable, err := m.RefGet(result.variables[0])
	fmMust(t, err)
	decl, err := m.GetDecl(variable)
	fmMust(t, err)
	value, err := md.Core().ConstantInterpretation(decl)
	fmMust(t, err)
	if value == (ast.ASTHandle{}) {
		t.Fatal("fresh mod variable missing model interpretation")
	}
	text, _, numeral, err := m.ArithmeticNumeralText(value)
	fmMust(t, err)
	if !numeral || text != "2" {
		t.Fatal("fresh mod variable model value", text, numeral)
	}
	handle, err := m.RefGet(result.formula)
	fmMust(t, err)
	family, err := m.GetFamilyID(handle)
	fmMust(t, err)
	op, err := m.GetDeclKind(handle)
	fmMust(t, err)
	if family != m.BasicFamilyID() || op != ast.ASTOpAnd {
		t.Fatal("factored formula did not conjoin definition", family, op)
	}
}
