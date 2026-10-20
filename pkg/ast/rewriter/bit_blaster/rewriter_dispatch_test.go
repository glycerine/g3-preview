package bit_blaster

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	parent "github.com/glycerine/g3/pkg/ast/rewriter"
)

func dispatchTerm(t *testing.T, m *ast.ASTManager, r *BitBlasterRewriter, term ast.ASTHandle, out ast.ASTRef) parent.RewriteStatus {
	t.Helper()
	decl, err := m.GetDecl(term)
	paramsMust(t, err)
	args, err := m.AppArgumentView(term)
	paramsMust(t, err)
	status, err := r.reduceApp(decl, args, out)
	paramsMust(t, err)
	return status
}

func TestBitBlasterRewriterApplicationDispatcher(t *testing.T) {
	m, r := rewriterFixture(t)
	aRef := rewriterConstant(t, r, 3, 4)
	bRef := rewriterConstant(t, r, 5, 4)
	defer func() {
		paramsMust(t, m.DelRef(&bRef))
		paramsMust(t, m.DelRef(&aRef))
	}()
	a, err := m.RefGet(aRef)
	paramsMust(t, err)
	b, err := m.RefGet(bRef)
	paramsMust(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&out)) }()

	numeral, err := m.BitVectorNumeralUint64(11, 4)
	paramsMust(t, err)
	if status := dispatchTerm(t, m, r, numeral, out); status != parent.RewriteDone {
		t.Fatal("numeral status", status)
	}
	got, err := m.RefGet(out)
	paramsMust(t, err)
	if value := rewriterValue(t, r, got); value != 11 {
		t.Fatal("numeral value", value)
	}

	sort, err := m.GetSort(a)
	paramsMust(t, err)
	add, err := m.MkBitVectorPolynomialAdd(sort, ast.SliceArguments([]ast.ASTHandle{a, b}))
	paramsMust(t, err)
	if status := dispatchTerm(t, m, r, add, out); status != parent.RewriteDone {
		t.Fatal("add status", status)
	}
	got, err = m.RefGet(out)
	paramsMust(t, err)
	if value := rewriterValue(t, r, got); value != 8 {
		t.Fatal("add value", value)
	}

	equality, err := m.MkEq(a, b)
	paramsMust(t, err)
	if status := dispatchTerm(t, m, r, equality, out); status != parent.RewriteDone {
		t.Fatal("equality status", status)
	}
	got, err = m.RefGet(out)
	paramsMust(t, err)
	if !m.IsFalse(got) {
		t.Fatal("equality value")
	}

	extract, err := m.MkBitVectorExtract(3, 1, b)
	paramsMust(t, err)
	if status := dispatchTerm(t, m, r, extract, out); status != parent.RewriteDone {
		t.Fatal("extract status", status)
	}
	got, err = m.RefGet(out)
	paramsMust(t, err)
	if value := rewriterValue(t, r, got); value != 2 {
		t.Fatal("extract value", value)
	}

	r.params.BlastAdd = false
	if status := dispatchTerm(t, m, r, add, out); status != parent.RewriteFailed {
		t.Fatal("disabled add status", status)
	}
}

func TestBitBlasterRewriterDispatcherDivisionPolicy(t *testing.T) {
	m, r := rewriterFixture(t)
	aRef := rewriterConstant(t, r, 3, 4)
	bRef := rewriterConstant(t, r, 1, 4)
	defer func() {
		paramsMust(t, m.DelRef(&bRef))
		paramsMust(t, m.DelRef(&aRef))
	}()
	a, err := m.RefGet(aRef)
	paramsMust(t, err)
	b, err := m.RefGet(bRef)
	paramsMust(t, err)
	division, err := m.MkFamilyApp(r.family, 8, nil, []ast.ASTHandle{a, b})
	paramsMust(t, err)
	decl, err := m.GetDecl(division)
	paramsMust(t, err)
	args, err := m.AppArgumentView(division)
	paramsMust(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&out)) }()
	status, err := r.reduceApp(decl, args, out)
	if status != parent.RewriteFailed || err != ErrUnsupportedOperator {
		t.Fatal("enabled division policy", status, err)
	}
	r.params.BlastMul = false
	status, err = r.reduceApp(decl, args, out)
	if status != parent.RewriteFailed || err != nil {
		t.Fatal("disabled division policy", status, err)
	}
}
