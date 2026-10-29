package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// arith_project_util::project, integer virtual substitution paths.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyArithmeticProjectionProjectsIntegerPaths(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArithmeticProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	x := f.constant("legacy-project-integer-x", f.integer)
	zero := f.own(m.ArithmeticIntegerNumeralText("0"))
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	two := f.own(m.ArithmeticIntegerNumeralText("2"))
	three := f.own(m.ArithmeticIntegerNumeralText("3"))
	four := f.own(m.ArithmeticIntegerNumeralText("4"))
	upper := f.own(m.MkArithmeticLE(x, three))
	lower := f.own(m.MkArithmeticGE(x, zero))
	twoX := f.own(m.MkArithmeticMul(two, x))
	equality := f.own(m.MkEq(twoX, four))
	modulus := f.own(m.MkFamilyApp(f.r.arithmetic, 13, nil, []ast.ASTHandle{x, two}))
	divides := f.own(m.MkEq(modulus, zero))
	xDecl, err := m.GetDecl(x)
	fmMust(t, err)
	withModel := func(value ast.ASTHandle, body func(*model.ModelCore)) {
		md, err := model.NewModel(m)
		fmMust(t, err)
		fmMust(t, md.Core().RegisterConstant(xDecl, value))
		body(md.Core())
		fmMust(t, md.Close())
	}
	checkVariableFree := func(result *legacyRealProjection) {
		for _, replacement := range result.replacements {
			handle, err := m.RefGet(replacement.replacement)
			fmMust(t, err)
			contains, err := p.contains(handle, x)
			fmMust(t, err)
			if contains {
				t.Fatal("integer replacement retains projected variable")
			}
		}
		if result.divisibility != (ast.ASTRef{}) {
			handle, err := m.RefGet(result.divisibility)
			fmMust(t, err)
			contains, err := p.contains(handle, x)
			fmMust(t, err)
			if contains {
				t.Fatal("integer side constraint retains projected variable")
			}
		}
	}

	withModel(one, func(md *model.ModelCore) {
		result, projected, err := p.projectInteger(md, x, []ast.ASTHandle{upper, lower})
		fmMust(t, err)
		defer func() { fmMust(t, result.Close()) }()
		if !projected || len(result.replacements) != 2 || result.divisibility != (ast.ASTRef{}) {
			t.Fatal("finite integer projection shape", projected, len(result.replacements))
		}
		checkVariableFree(result)
	})

	withModel(two, func(md *model.ModelCore) {
		result, projected, err := p.projectInteger(md, x, []ast.ASTHandle{equality})
		fmMust(t, err)
		defer func() { fmMust(t, result.Close()) }()
		if !projected || len(result.replacements) != 1 || result.divisibility == (ast.ASTRef{}) {
			t.Fatal("integer equality projection shape", projected, len(result.replacements))
		}
		checkVariableFree(result)
	})

	withModel(two, func(md *model.ModelCore) {
		result, projected, err := p.projectInteger(md, x, []ast.ASTHandle{upper, divides})
		fmMust(t, err)
		defer func() { fmMust(t, result.Close()) }()
		if !projected || len(result.replacements) != 2 {
			t.Fatal("integer infinity projection shape", projected, len(result.replacements))
		}
		checkVariableFree(result)
		evaluator, err := model.NewModelEvaluator(md, model.ModelEvaluatorConfig{Context: f.r.config.Context, Globals: f.r.config.Globals, Memory: m.NumeralService().RuntimeOwner(), Cooperation: f.r.config.Cooperation})
		fmMust(t, err)
		defer func() { fmMust(t, evaluator.Close()) }()
		for _, replacement := range result.replacements {
			handle, err := m.RefGet(replacement.replacement)
			fmMust(t, err)
			value, err := evaluator.EvaluateValue(handle)
			fmMust(t, err)
			defer func() { fmMust(t, m.DelRef(&value)) }()
			valueHandle, err := m.RefGet(value)
			fmMust(t, err)
			if !m.IsTrue(valueHandle) {
				t.Fatal("model congruence did not preserve one-sided integer literal")
			}
		}
	})
}
