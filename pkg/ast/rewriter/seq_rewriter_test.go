package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSequenceRewriterConcatSourceBranches(t *testing.T) {
	m := astTestManager(t)
	rw, err := NewSequenceRewriter(m)
	astTestMust(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	str := func(value string) ast.ASTHandle {
		h, e := m.MkString(value)
		astTestMust(t, e)
		return h
	}
	a, b, empty := str("a"), str("b"), str("")
	astTestPin(t, m, a)
	astTestPin(t, m, b)
	astTestPin(t, m, empty)
	checkString := func(want string, status RewriteStatus) {
		t.Helper()
		if status != RewriteDone {
			t.Fatalf("status %d, want done", status)
		}
		h, e := m.RefGet(out)
		astTestMust(t, e)
		got, matched, e := m.SequenceStringValue(h)
		astTestMust(t, e)
		if !matched || got != want {
			t.Fatalf("string result: %q matched=%v, want %q", got, matched, want)
		}
	}
	status, err := rw.RewriteConcat(a, b, out)
	astTestMust(t, err)
	checkString("ab", status)

	status, err = rw.RewriteConcat(empty, b, out)
	astTestMust(t, err)
	checkString("b", status)
	status, err = rw.RewriteConcat(a, empty, out)
	astTestMust(t, err)
	checkString("a", status)

	stringSort, err := m.GetSort(a)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "seq-x"), stringSort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	left, err := m.MkSequenceConcat(x, a)
	astTestMust(t, err)
	astTestPin(t, m, left)
	status, err = rw.RewriteConcat(left, b, out)
	astTestMust(t, err)
	if status != Rewrite2 {
		t.Fatalf("right association status: %d", status)
	}
	result, err := m.RefGet(out)
	astTestMust(t, err)
	leaves, err := m.SequenceFlattenConcat(result)
	astTestMust(t, err)
	if len(leaves) != 3 || leaves[0] != x || leaves[1] != a || leaves[2] != b {
		t.Fatalf("right-associated leaves: %v", leaves)
	}

	tail, err := m.MkSequenceConcat(b, x)
	astTestMust(t, err)
	astTestPin(t, m, tail)
	status, err = rw.RewriteConcat(a, tail, out)
	astTestMust(t, err)
	if status != RewriteDone {
		t.Fatalf("adjacent string fold status: %d", status)
	}
	result, err = m.RefGet(out)
	astTestMust(t, err)
	leaves, err = m.SequenceFlattenConcat(result)
	astTestMust(t, err)
	if len(leaves) != 2 || leaves[1] != x {
		t.Fatalf("adjacent string leaves: %v", leaves)
	}
	text, matched, err := m.SequenceStringValue(leaves[0])
	astTestMust(t, err)
	if !matched || text != "ab" {
		t.Fatalf("adjacent string prefix: %q matched=%v", text, matched)
	}

	status, err = rw.RewriteConcat(x, x, out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatalf("unreduced concat status: %d", status)
	}
}
