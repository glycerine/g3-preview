package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// arith_project_util::project, real-arithmetic paths.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyArithmeticProjectionProjectsRealBounds(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArithmeticProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	x := f.constant("legacy-real-x", f.real)
	y := f.constant("legacy-real-y", f.real)
	zero := f.own(m.ArithmeticRealNumeralText("0"))
	one := f.own(m.ArithmeticRealNumeralText("1"))
	two := f.own(m.ArithmeticRealNumeralText("2"))
	three := f.own(m.ArithmeticRealNumeralText("3"))
	upper := f.own(m.MkArithmeticLE(x, three))
	lower := f.own(m.MkArithmeticGE(x, zero))
	equality := f.own(m.MkEq(x, y))
	md, err := model.NewModel(m)
	fmMust(t, err)
	defer func() { fmMust(t, md.Close()) }()
	xDecl, err := m.GetDecl(x)
	fmMust(t, err)
	yDecl, err := m.GetDecl(y)
	fmMust(t, err)
	fmMust(t, md.Core().RegisterConstant(xDecl, one))
	fmMust(t, md.Core().RegisterConstant(yDecl, two))

	finite, projected, err := p.projectReal(md.Core(), x, []ast.ASTHandle{upper, lower})
	fmMust(t, err)
	if !projected || finite.substitution != (ast.ASTRef{}) || len(finite.replacements) != 2 {
		t.Fatal("finite real projection shape", projected, len(finite.replacements))
	}
	for _, replacement := range finite.replacements {
		handle, err := m.RefGet(replacement.replacement)
		fmMust(t, err)
		contains, err := p.contains(handle, x)
		fmMust(t, err)
		if contains {
			t.Fatal("finite replacement retains projected variable")
		}
	}
	fmMust(t, finite.Close())

	infinity, projected, err := p.projectReal(md.Core(), x, []ast.ASTHandle{upper})
	fmMust(t, err)
	if !projected || len(infinity.replacements) != 1 {
		t.Fatal("infinity real projection shape", projected, len(infinity.replacements))
	}
	infinityValue, err := m.RefGet(infinity.replacements[0].replacement)
	fmMust(t, err)
	if !m.IsTrue(infinityValue) {
		t.Fatal("one-sided upper bound was not eliminated with minus infinity")
	}
	fmMust(t, infinity.Close())

	// Make the equality true in the model and check direct substitution.
	fmMust(t, md.Core().RegisterConstant(xDecl, two))
	equal, projected, err := p.projectReal(md.Core(), x, []ast.ASTHandle{equality, upper})
	fmMust(t, err)
	if !projected || equal.substitution == (ast.ASTRef{}) || len(equal.replacements) != 0 {
		t.Fatal("equality real projection shape", projected, len(equal.replacements))
	}
	substitution, err := m.RefGet(equal.substitution)
	fmMust(t, err)
	contains, err := p.contains(substitution, x)
	fmMust(t, err)
	if contains {
		t.Fatal("equality substitution retains projected variable")
	}
	fmMust(t, equal.Close())
}
