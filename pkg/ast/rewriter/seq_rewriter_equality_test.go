package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSequenceRewriterEqualityBoundaryReductions(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewSequenceRewriter(m)
	astTestMust(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	pin := func(h ast.ASTHandle, err error) ast.ASTHandle {
		t.Helper()
		astTestMust(t, err)
		astTestPin(t, m, h)
		return h
	}
	str := func(text string) ast.ASTHandle { return pin(m.MkString(text)) }
	concat := func(left, right ast.ASTHandle) ast.ASTHandle {
		return pin(m.MkSequenceConcat(left, right))
	}
	stringSort, err := m.GetSort(str("equality sort witness"))
	astTestMust(t, err)
	constant := func(name string) ast.ASTHandle {
		return pin(m.MkConst(astTestSymbol(t, m, name), stringSort))
	}
	x, y := constant("equality-x"), constant("equality-y")
	binary := func(h ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, int32) {
		t.Helper()
		kind, e := m.GetDeclKind(h)
		astTestMust(t, e)
		args, e := m.AppArgumentView(h)
		astTestMust(t, e)
		if args.Count() != 2 {
			t.Fatalf("kind %d argument count: %d", kind, args.Count())
		}
		left, e := args.At(0)
		astTestMust(t, e)
		right, e := args.At(1)
		astTestMust(t, e)
		return left, right, kind
	}

	tail := str("tail")
	status, err := rw.RewriteEquality(concat(x, tail), concat(y, tail), out)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	left, right, kind := binary(result)
	if status != Rewrite3 || kind != ast.ASTOpEq || left != x || right != y {
		t.Fatalf("common suffix: status=%d kind=%d left=%v right=%v", status, kind, left, right)
	}

	head := str("head")
	status, err = rw.RewriteEquality(concat(head, x), concat(head, y), out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	left, right, kind = binary(result)
	if status != Rewrite3 || kind != ast.ASTOpEq || left != x || right != y {
		t.Fatalf("common prefix: status=%d kind=%d left=%v right=%v", status, kind, left, right)
	}

	status, err = rw.RewriteEquality(concat(str("ab"), x), concat(str("a"), y), out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	left, right, kind = binary(result)
	if status != Rewrite3 || kind != ast.ASTOpEq || right != y {
		t.Fatalf("split String prefix: status=%d kind=%d left=%v right=%v", status, kind, left, right)
	}
	parts, err := m.SequenceFlattenConcat(left)
	astTestMust(t, err)
	if len(parts) != 2 || parts[1] != x {
		t.Fatalf("split String residual: %v", parts)
	}
	text, matched, err := m.SequenceStringValue(parts[0])
	astTestMust(t, err)
	if !matched || text != "b" {
		t.Fatalf("split String head: %q matched=%v", text, matched)
	}

	status, err = rw.RewriteEquality(str("left"), str("right"), out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteDone || !m.IsFalse(result) {
		t.Fatalf("different Strings: status=%d result=%v", status, result)
	}

	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	p := pin(m.MkConst(astTestSymbol(t, m, "equality-p"), boolSort))
	q := pin(m.MkConst(astTestSymbol(t, m, "equality-q"), boolSort))
	unitP := pin(m.MkSequenceUnit(p))
	unitQ := pin(m.MkSequenceUnit(q))
	status, err = rw.RewriteEquality(unitP, unitQ, out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	left, right, kind = binary(result)
	if status != Rewrite3 || kind != ast.ASTOpIff || left != p || right != q {
		t.Fatalf("unit equality: status=%d kind=%d left=%v right=%v", status, kind, left, right)
	}

	letter := pin(m.BitVectorNumeralUnsigned('A', 8))
	unitLetter := pin(m.MkSequenceUnit(letter))
	status, err = rw.RewriteEquality(unitLetter, str("A"), out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	_, _, kind = binary(result)
	if status != Rewrite3 || kind != ast.ASTOpEq {
		t.Fatalf("unit/String equality: status=%d kind=%d", status, kind)
	}

	empty := str("")
	status, err = rw.RewriteEquality(concat(x, y), empty, out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || status != Rewrite3 || kind != ast.ASTOpAnd {
		t.Fatalf("empty concat equality: status=%d kind=%d error=%v", status, kind, e)
	}
	status, err = rw.RewriteEquality(x, empty, out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatalf("singleton empty equality status: %d", status)
	}
}
