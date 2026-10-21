package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSequenceRewriterAutomatonMembership(t *testing.T) {
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
	toRegex := func(text string) ast.ASTHandle { return pin(m.MkSequenceToRegex(str(text))) }
	truth, err := m.MkTrue()
	astTestMust(t, err)
	falsehood, err := m.MkFalse()
	astTestMust(t, err)
	check := func(name string, input, regex ast.ASTHandle, want ast.ASTHandle) {
		t.Helper()
		status, e := rw.RewriteStringInRegex(input, regex, out)
		astTestMust(t, e)
		result, e := m.RefGet(out)
		astTestMust(t, e)
		if status != RewriteFull || result != want {
			t.Fatalf("%s: status=%d result=%v want=%v", name, status, result, want)
		}
	}

	singleton := toRegex("ab")
	status, err := rw.RewriteStringInRegex(str("input"), singleton, out)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || status != RewriteFull || kind != ast.ASTOpEq {
		t.Fatalf("singleton language: status=%d kind=%d error=%v", status, kind, e)
	}

	union := pin(m.MkRegexUnion(toRegex("a"), toRegex("b")))
	check("union accepts", str("b"), union, truth)
	check("union rejects", str("c"), union, falsehood)
	star := pin(m.MkRegexStar(toRegex("a")))
	check("star empty", str(""), star, truth)
	check("star repeated", str("aaa"), star, truth)
	check("star rejects", str("aab"), star, falsehood)
	plus := pin(m.MkRegexPlus(toRegex("a")))
	check("plus accepts", str("aa"), plus, truth)
	check("plus rejects empty", str(""), plus, falsehood)
	option := pin(m.MkRegexOption(toRegex("a")))
	check("option accepts empty", str(""), option, truth)
	check("option rejects repeated", str("aa"), option, falsehood)

	rangeRegex := pin(m.MkRegexRange(str("a"), str("z")))
	check("range accepts", str("m"), rangeRegex, truth)
	check("range rejects", str("0"), rangeRegex, falsehood)
	regexSort, err := m.GetSort(rangeRegex)
	astTestMust(t, err)
	fullCharacter := pin(m.MkRegexFullChar(regexSort))
	check("full character accepts", str("x"), fullCharacter, truth)
	check("full character rejects pair", str("xy"), fullCharacter, falsehood)

	loop := pin(m.MkRegexLoop(toRegex("a"), 2, 3))
	check("bounded loop lower", str("aa"), loop, truth)
	check("bounded loop upper", str("aaa"), loop, truth)
	check("bounded loop below", str("a"), loop, falsehood)
	check("bounded loop above", str("aaaa"), loop, falsehood)

	byteZero := pin(m.BitVectorNumeralUnsigned(0, 8))
	byteSort, err := m.GetSort(byteZero)
	astTestMust(t, err)
	symbolicByte := pin(m.MkConst(astTestSymbol(t, m, "automaton-byte"), byteSort))
	symbolicUnit := pin(m.MkSequenceUnit(symbolicByte))
	status, err = rw.RewriteStringInRegex(symbolicUnit, rangeRegex, out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || status != RewriteFull || kind != ast.ASTOpAnd {
		t.Fatalf("symbolic range: status=%d kind=%d error=%v", status, kind, e)
	}
}
