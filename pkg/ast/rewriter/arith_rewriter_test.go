package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func arithRewriterNumeral(t *testing.T, m *ast.ASTManager, text string, integer bool) ast.ASTHandle {
	t.Helper()
	service := m.NumeralService()
	n, err := service.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, service.Del(&n)) }()
	astTestMust(t, service.SetString(&n, text))
	h, err := m.ArithmeticNumeral(&n, integer)
	astTestMust(t, err)
	astTestPin(t, m, h)
	return h
}

func arithRewriterCore(t *testing.T, r *ArithRewriter, kind int32, args ...ast.ASTHandle) (RewriteStatus, ast.ASTHandle) {
	t.Helper()
	app, err := r.manager.MkFamilyApp(r.family, kind, nil, args)
	astTestMust(t, err)
	decl, err := r.manager.GetDecl(app)
	astTestMust(t, err)
	out := astTestPin(t, r.manager, app)
	status, err := r.MkAppCore(decl, ast.SliceArguments(args), out)
	astTestMust(t, err)
	result, err := r.manager.RefGet(out)
	astTestMust(t, err)
	return status, result
}

func arithRewriterWantNumeral(t *testing.T, m *ast.ASTManager, h ast.ASTHandle, text string, integer bool) {
	t.Helper()
	got, gotInteger, matched, err := m.ArithmeticNumeralText(h)
	astTestMust(t, err)
	if !matched || got != text || gotInteger != integer {
		t.Fatal("arithmetic numeral", got, gotInteger, matched, "want", text, integer)
	}
}

func TestArithRewriterRationalScalarCores(t *testing.T) {
	m, c, g, local := boolRewriterConfigFixture(t)
	r, err := NewArithRewriter(m, c, g, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	i2 := arithRewriterNumeral(t, m, "2", true)
	i3 := arithRewriterNumeral(t, m, "3", true)
	iNeg3 := arithRewriterNumeral(t, m, "-3", true)
	iNeg2 := arithRewriterNumeral(t, m, "-2", true)
	r2 := arithRewriterNumeral(t, m, "2", false)
	r3 := arithRewriterNumeral(t, m, "3", false)
	rHalf := arithRewriterNumeral(t, m, "1/2", false)
	rNegThreeHalves := arithRewriterNumeral(t, m, "-3/2", false)

	for _, row := range []struct {
		kind        int32
		args        []ast.ASTHandle
		want        string
		wantInteger bool
		wantStatus  RewriteStatus
	}{
		{arithOpAdd, []ast.ASTHandle{i2, i3}, "5", true, RewriteDone},
		{arithOpMul, []ast.ASTHandle{rHalf, r3}, "3/2", false, RewriteDone},
		{arithOpDiv, []ast.ASTHandle{r3, r2}, "3/2", false, RewriteDone},
		{arithOpIDiv, []ast.ASTHandle{iNeg3, i2}, "-2", true, RewriteDone},
		{arithOpIDiv, []ast.ASTHandle{i3, iNeg2}, "-1", true, RewriteDone},
		{arithOpMod, []ast.ASTHandle{iNeg3, i2}, "1", true, RewriteDone},
		{arithOpMod, []ast.ASTHandle{i3, iNeg2}, "1", true, RewriteDone},
		{arithOpRem, []ast.ASTHandle{i3, iNeg2}, "-1", true, RewriteDone},
		{arithOpToInt, []ast.ASTHandle{rNegThreeHalves}, "-2", true, RewriteDone},
		{arithOpToReal, []ast.ASTHandle{i3}, "3", false, RewriteDone},
		{arithOpPower, []ast.ASTHandle{i2, i3}, "8", true, RewriteDone},
		{arithOpPower, []ast.ASTHandle{r2, iNeg2}, "1/4", false, RewriteDone},
	} {
		status, result := arithRewriterCore(t, r, row.kind, row.args...)
		if status != row.wantStatus {
			t.Fatal("arithmetic status", row.kind, status, row.wantStatus)
		}
		arithRewriterWantNumeral(t, m, result, row.want, row.wantInteger)
	}

	status, result := arithRewriterCore(t, r, arithOpLE, i2, i3)
	if status != RewriteDone || !m.IsTrue(result) {
		t.Fatal("numeric comparison", status, result)
	}
	status, result = arithRewriterCore(t, r, arithOpIsInt, rHalf)
	if status != RewriteDone || !m.IsFalse(result) {
		t.Fatal("numeric is-int", status, result)
	}
	out := astTestPin(t, m, result)
	status, err = r.MkEqCore(i2, i3, out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteDone || !m.IsFalse(result) {
		t.Fatal("numeric equality", status, result)
	}
}

func TestArithRewriterStructuralScalarRules(t *testing.T) {
	m, c, g, local := boolRewriterConfigFixture(t)
	r, err := NewArithRewriter(m, c, g, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	iSort, err := m.MkFamilySort(r.family, 1, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "arith-rewriter-x"), iSort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	i2 := arithRewriterNumeral(t, m, "2", true)

	status, result := arithRewriterCore(t, r, arithOpLT, x, i2)
	if status != Rewrite2 {
		t.Fatal("less-than status", status)
	}
	if yes, _, err := m.IsNot(result); err != nil || !yes {
		t.Fatal("less-than source negated comparison", yes, err)
	}
	status, result = arithRewriterCore(t, r, arithOpAbs, x)
	if status != Rewrite2 {
		t.Fatal("absolute-value status", status)
	}
	if kind, err := m.GetDeclKind(result); err != nil || kind != ast.ASTOpIte {
		t.Fatal("absolute-value ITE", kind, err)
	}
	toReal, err := m.MkFamilyApp(r.family, arithOpToReal, nil, []ast.ASTHandle{x})
	astTestMust(t, err)
	astTestPin(t, m, toReal)
	status, result = arithRewriterCore(t, r, arithOpToInt, toReal)
	if status != RewriteDone || result != x {
		t.Fatal("to-int/to-real cancellation", status, result, x)
	}
	status, result = arithRewriterCore(t, r, arithOpIsInt, toReal)
	if status != RewriteDone || !m.IsTrue(result) {
		t.Fatal("to-real is integral", status, result)
	}
}
