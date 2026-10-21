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

func TestSequenceRewriterByteUnit(t *testing.T) {
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
	for _, value := range []uint32{0, 65, 255} {
		literal, e := m.BitVectorNumeralUnsigned(value, 8)
		astTestMust(t, e)
		astTestPin(t, m, literal)
		status, e := rw.RewriteUnit(literal, out)
		astTestMust(t, e)
		if status != RewriteDone {
			t.Fatalf("byte %d status: %d", value, status)
		}
		result, e := m.RefGet(out)
		astTestMust(t, e)
		text, matched, e := m.SequenceStringValue(result)
		astTestMust(t, e)
		if !matched || text != string([]byte{byte(value)}) {
			t.Fatalf("byte %d result: %q matched=%v", value, text, matched)
		}
	}
	wide, err := m.BitVectorNumeralUnsigned(65, 16)
	astTestMust(t, err)
	astTestPin(t, m, wide)
	status, err := rw.RewriteUnit(wide, out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatalf("wide unit status: %d", status)
	}
}

func TestSequenceRewriterConstantPredicates(t *testing.T) {
	m := astTestManager(t)
	rw, err := NewSequenceRewriter(m)
	astTestMust(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	values := map[string]ast.ASTHandle{}
	for _, text := range []string{"", "ab", "abc", "bc", "z"} {
		values[text], err = m.MkString(text)
		astTestMust(t, err)
		astTestPin(t, m, values[text])
	}
	truth, err := m.MkTrue()
	astTestMust(t, err)
	falsehood, err := m.MkFalse()
	astTestMust(t, err)
	check := func(name string, rewrite func(ast.ASTHandle, ast.ASTHandle, ast.ASTRef) (RewriteStatus, error), a, b ast.ASTHandle, want ast.ASTHandle) {
		t.Helper()
		status, e := rewrite(a, b, out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if status != RewriteDone || got != want {
			t.Fatalf("%s: status=%d result=%v want=%v", name, status, got, want)
		}
	}
	check("contains", rw.RewriteContains, values["abc"], values["bc"], truth)
	check("not-contains", rw.RewriteContains, values["ab"], values["bc"], falsehood)
	check("prefix", rw.RewritePrefix, values["ab"], values["abc"], truth)
	check("not-prefix", rw.RewritePrefix, values["abc"], values["ab"], falsehood)
	check("empty-prefix", rw.RewritePrefix, values[""], values["z"], truth)
	check("suffix", rw.RewriteSuffix, values["bc"], values["abc"], truth)
	check("not-suffix", rw.RewriteSuffix, values["abc"], values["bc"], falsehood)
	check("identity-suffix", rw.RewriteSuffix, values["ab"], values["ab"], truth)
	status, err := rw.RewriteSuffix(values["ab"], values[""], out)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || status != Rewrite3 || kind != ast.ASTOpEq {
		t.Fatalf("empty target suffix: status=%d kind=%d error=%v", status, kind, e)
	}
}
