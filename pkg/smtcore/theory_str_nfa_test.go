package smtcore

import (
	"regexp"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTStringNFAThompsonLanguage(t *testing.T) {
	m := astTestManager(t)
	var literals []ast.ASTHandle
	for _, text := range []string{"ab", "b", "c"} {
		value, err := m.MkString(text)
		astTestMust(t, err)
		regex, err := m.MkSequenceToRegex(value)
		astTestMust(t, err)
		literals = append(literals, regex)
	}
	union, err := m.MkRegexUnion(literals[0], literals[1])
	astTestMust(t, err)
	star, err := m.MkRegexStar(union)
	astTestMust(t, err)
	expression, err := m.MkRegexConcat(star, literals[2])
	astTestMust(t, err)
	nfa, err := newSMTStringNFA(m, expression)
	astTestMust(t, err)
	if !nfa.valid {
		t.Fatal("retained regex was refused")
	}
	reference := regexp.MustCompile("^(ab|b)*c$")
	words := []string{""}
	for length := 0; length <= 6; length++ {
		var next []string
		for _, word := range words {
			if nfa.matches(word) != reference.MatchString(word) {
				t.Fatalf("NFA disagrees with independent language reference on %q", word)
			}
			for _, letter := range []byte{'a', 'b', 'c'} {
				next = append(next, word+string(letter))
			}
		}
		words = next
	}
}

func TestSMTStringNFAEpsilonCyclesAndEveryByte(t *testing.T) {
	m := astTestManager(t)
	empty, err := m.MkString("")
	astTestMust(t, err)
	emptyRegex, err := m.MkSequenceToRegex(empty)
	astTestMust(t, err)
	star, err := m.MkRegexStar(emptyRegex)
	astTestMust(t, err)
	star, err = m.MkRegexStar(star)
	astTestMust(t, err)
	nfa, err := newSMTStringNFA(m, star)
	astTestMust(t, err)
	if !nfa.valid || !nfa.matches("") || nfa.matches("a") {
		t.Fatal("epsilon cycle did not recognize precisely the empty string")
	}
	sort, err := m.GetSort(emptyRegex)
	astTestMust(t, err)
	fullCharacter, err := m.MkRegexFullChar(sort)
	astTestMust(t, err)
	full, err := m.MkRegexFull(sort)
	astTestMust(t, err)
	characterNFA, err := newSMTStringNFA(m, fullCharacter)
	astTestMust(t, err)
	fullNFA, err := newSMTStringNFA(m, full)
	astTestMust(t, err)
	if characterNFA.matches("") || !fullNFA.matches("") {
		t.Fatal("full-character and full-sequence empty cases were conflated")
	}
	for symbol := 0; symbol < 256; symbol++ {
		word := string([]byte{byte(symbol)})
		if !characterNFA.matches(word) || characterNFA.matches(word+word) || !fullNFA.matches(word+word) {
			t.Fatalf("byte alphabet mismatch at 0x%02x", symbol)
		}
	}
	lower, err := m.MkString(string([]byte{255}))
	astTestMust(t, err)
	upper, err := m.MkString(string([]byte{253}))
	astTestMust(t, err)
	rangeRegex, err := m.MkRegexRange(lower, upper)
	astTestMust(t, err)
	rangeNFA, err := newSMTStringNFA(m, rangeRegex)
	astTestMust(t, err)
	for symbol := 0; symbol < 256; symbol++ {
		if rangeNFA.matches(string([]byte{byte(symbol)})) != (symbol >= 253) {
			t.Fatalf("reversed high-byte range mismatch at 0x%02x", symbol)
		}
	}
}

func TestSMTStringNFAUnsupportedAndNonconstantLiteral(t *testing.T) {
	m := astTestManager(t)
	value, err := m.MkString("a")
	astTestMust(t, err)
	literal, err := m.MkSequenceToRegex(value)
	astTestMust(t, err)
	plus, err := m.MkRegexPlus(literal)
	astTestMust(t, err)
	nested, err := m.MkRegexStar(plus)
	astTestMust(t, err)
	nfa, err := newSMTStringNFA(m, nested)
	astTestMust(t, err)
	if nfa.valid || nfa.matches("a") {
		t.Fatal("unsupported source NFA operator acquired accepting behavior")
	}
	sort, err := m.GetSort(value)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "string-nfa-nonconstant"), sort)
	astTestMust(t, err)
	nonconstant, err := m.MkSequenceToRegex(x)
	astTestMust(t, err)
	nfa, err = newSMTStringNFA(m, nonconstant)
	if err == nil || err.Error() != "invalid term in str.to.re, argument must be a string constant" || nfa.valid {
		t.Fatal("nonconstant str.to.re did not preserve the source refusal", err)
	}
}
