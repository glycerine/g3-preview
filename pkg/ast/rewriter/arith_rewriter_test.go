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

func arithRewriterWantOperation(t *testing.T, r *ArithRewriter, h ast.ASTHandle, kind int32, args ...ast.ASTHandle) {
	t.Helper()
	gotKind, gotArgs, matched, err := r.operation(h, kind)
	astTestMust(t, err)
	if !matched || gotKind != kind || len(gotArgs) != len(args) {
		t.Fatal("arithmetic operation", gotKind, len(gotArgs), matched, "want", kind, len(args))
	}
	for i := range args {
		if gotArgs[i] != args[i] {
			t.Fatal("arithmetic argument", i, gotArgs[i], args[i])
		}
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

func TestArithRewriterSymbolicModRemAndCasts(t *testing.T) {
	m, c, g, local := boolRewriterConfigFixture(t)
	r, err := NewArithRewriter(m, c, g, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	iSort, err := m.MkFamilySort(r.family, 1, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "arith-symbolic-x"), iSort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	y, err := m.MkConst(astTestSymbol(t, m, "arith-symbolic-y"), iSort)
	astTestMust(t, err)
	astTestPin(t, m, y)
	i1 := arithRewriterNumeral(t, m, "1", true)
	i2 := arithRewriterNumeral(t, m, "2", true)
	iNeg2 := arithRewriterNumeral(t, m, "-2", true)
	r1 := arithRewriterNumeral(t, m, "1", false)

	status, result := arithRewriterCore(t, r, arithOpMod, x, i1)
	if status != RewriteDone {
		t.Fatal("modulo one status", status)
	}
	arithRewriterWantNumeral(t, m, result, "0", true)

	status, result = arithRewriterCore(t, r, arithOpMod, x, x)
	if status != RewriteDone {
		t.Fatal("self modulo status", status)
	}
	if kind, e := m.GetDeclKind(result); e != nil || kind != ast.ASTOpIte {
		t.Fatal("self modulo ITE", kind, e)
	}

	innerMod, err := r.app(arithOpMod, x, i2)
	astTestMust(t, err)
	astTestPin(t, m, innerMod)
	status, result = arithRewriterCore(t, r, arithOpMod, innerMod, i2)
	if status != RewriteDone || result != innerMod {
		t.Fatal("idempotent modulo", status, result, innerMod)
	}

	sum, err := r.app(arithOpAdd, x, i1)
	astTestMust(t, err)
	astTestPin(t, m, sum)
	status, result = arithRewriterCore(t, r, arithOpMod, sum, i2)
	if status != Rewrite3 {
		t.Fatal("modulo propagation status", status)
	}
	_, outerArgs, matched, err := r.operation(result, arithOpMod)
	astTestMust(t, err)
	if !matched || len(outerArgs) != 2 || outerArgs[1] != i2 {
		t.Fatal("modulo propagation outer", matched, len(outerArgs))
	}
	_, addArgs, matched, err := r.operation(outerArgs[0], arithOpAdd)
	astTestMust(t, err)
	if !matched || len(addArgs) != 2 {
		t.Fatal("modulo propagation sum", matched, len(addArgs))
	}
	for _, arg := range addArgs {
		_, modArgs, yes, e := r.operation(arg, arithOpMod)
		astTestMust(t, e)
		if !yes || len(modArgs) != 2 || modArgs[1] != i2 {
			t.Fatal("modulo propagation child", yes, len(modArgs))
		}
	}

	status, result = arithRewriterCore(t, r, arithOpRem, x, i2)
	if status != Rewrite1 {
		t.Fatal("positive symbolic remainder", status)
	}
	arithRewriterWantOperation(t, r, result, arithOpMod, x, i2)
	status, result = arithRewriterCore(t, r, arithOpRem, x, iNeg2)
	if status != Rewrite2 {
		t.Fatal("negative symbolic remainder", status)
	}
	_, negativeArgs, matched, err := r.operation(result, arithOpUMinus)
	astTestMust(t, err)
	if !matched || len(negativeArgs) != 1 {
		t.Fatal("negative symbolic remainder shape", matched, len(negativeArgs))
	}
	arithRewriterWantOperation(t, r, negativeArgs[0], arithOpMod, x, iNeg2)

	toRealX, err := r.app(arithOpToReal, x)
	astTestMust(t, err)
	astTestPin(t, m, toRealX)
	realSum, err := r.app(arithOpAdd, r1, toRealX)
	astTestMust(t, err)
	astTestPin(t, m, realSum)
	status, result = arithRewriterCore(t, r, arithOpToInt, realSum)
	if status != Rewrite1 {
		t.Fatal("to-int distribution status", status)
	}
	arithRewriterWantOperation(t, r, result, arithOpAdd, i1, x)

	status, result = arithRewriterCore(t, r, arithOpToReal, sum)
	if status != Rewrite2 {
		t.Fatal("to-real distribution status", status)
	}
	_, realArgs, matched, err := r.operation(result, arithOpAdd)
	astTestMust(t, err)
	if !matched || len(realArgs) != 2 {
		t.Fatal("to-real distribution sum", matched, len(realArgs))
	}
	arithRewriterWantOperation(t, r, realArgs[0], arithOpToReal, x)
	arithRewriterWantOperation(t, r, realArgs[1], arithOpToReal, i1)

	key := "elim_rem"
	astTestMust(t, c.SetBoolName(local, &key, true))
	eliminating, err := NewArithRewriter(m, c, g, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, eliminating.Close()) })
	status, result = arithRewriterCore(t, eliminating, arithOpRem, x, y)
	if status != Rewrite3 {
		t.Fatal("remainder elimination status", status)
	}
	if kind, e := m.GetDeclKind(result); e != nil || kind != ast.ASTOpIte {
		t.Fatal("remainder elimination ITE", kind, e)
	}
}
