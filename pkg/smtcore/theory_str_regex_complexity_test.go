package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/automata"
)

func TestSMTStringRegexComplexityUnsignedPolicy(t *testing.T) {
	max := ^uint32(0)
	for _, tc := range []struct{ a, b, sum, product uint32 }{
		{0, 9, 9, 0}, {2, 3, 5, 6}, {max, 0, max, max},
		{0, max, max, max}, {max - 1, 2, max, max},
		{65536, 65536, 131072, max},
		// The source's comparison-based overflow test deliberately misses this wrap.
		{65537, 65537, 131074, 131073},
	} {
		if got := stringRegexComplexityAdd(tc.a, tc.b); got != tc.sum {
			t.Fatal("native unsigned addition policy changed", tc, got)
		}
		if got := stringRegexComplexityMultiply(tc.a, tc.b); got != tc.product {
			t.Fatal("native unsigned multiplication policy changed", tc, got)
		}
	}
	counters := make(map[ast.ASTHandle]uint32)
	var key ast.ASTHandle
	if stringRegexCounter(counters, key) != 0 || len(counters) != 1 {
		t.Fatal("reading a missing counter must insert zero")
	}
	incrementStringRegexCounter(counters, key)
	if counters[key] != 1 {
		t.Fatal("counter increment lost")
	}
	counters[key] = max
	incrementStringRegexCounter(counters, key)
	if counters[key] != 0 {
		t.Fatal("counter must wrap instead of saturating")
	}
}

func TestSMTStringRegexComplexityComplementAndByteRanges(t *testing.T) {
	m, theory, _, _ := stringLibraryFixture(t)
	pin := func(value ast.ASTHandle, err error) ast.ASTHandle {
		astTestMust(t, err)
		astTestPin(t, m, value)
		return value
	}
	a, b := stringRegexLiteral(t, m, "ab"), stringRegexLiteral(t, m, "xyz")
	concat := pin(m.MkRegexConcat(a, b))
	union := pin(m.MkRegexUnion(a, b))
	complement := pin(m.MkRegexComplement(concat))
	sort, err := m.GetSort(a)
	astTestMust(t, err)
	for _, tc := range []struct {
		regex                ast.ASTHandle
		normal, complemented uint32
	}{
		{stringRegexLiteral(t, m, ""), 0, 0},
		{stringRegexLiteral(t, m, "\x00\xff"), 2, 2},
		{concat, 5, 7}, {pin(m.MkRegexConcat(b, a)), 5, 8},
		{union, 5, 6}, {complement, 7, 7},
		{pin(m.MkRegexComplement(complement)), 7, 7},
		{pin(m.MkRegexStar(union)), 10, 12},
		{pin(m.MkRegexPlus(concat)), 10, 14},
		{pin(m.MkRegexRange(astTestString(t, m, "a"), astTestString(t, m, "z"))), 26, 26},
		{pin(m.MkRegexRange(astTestString(t, m, "z"), astTestString(t, m, "a"))), 4294967272, 4294967272},
		{pin(m.MkRegexFull(sort)), 1, 1},
		{pin(m.MkRegexFullChar(sort)), 1, 1},
		{pin(m.MkRegexOption(a)), 1, 1},
	} {
		for _, under := range []bool{false, true} {
			want := tc.normal
			if under {
				want = tc.complemented
			}
			got, err := theory.estimateStringRegexComplexity(tc.regex, under)
			astTestMust(t, err)
			if got != want {
				t.Fatal("native regex complexity changed", tc, under, got, want)
			}
		}
	}
	if _, err := theory.estimateStringRegexComplexity(astTestString(t, m, "not a regex"), false); err == nil {
		t.Fatal("non-regex argument accepted")
	}
}

func TestSMTStringRegexIntersectionDifficulty(t *testing.T) {
	a := automata.New(0, []uint32{2}, nil)
	b := automata.New(0, []uint32{4}, nil)
	got, err := estimateStringAutomataIntersectionDifficulty(a, b)
	astTestMust(t, err)
	if got != 15 {
		t.Fatal("intersection difficulty must use allocated state counts", got)
	}
	if _, err := estimateStringAutomataIntersectionDifficulty(nil, b); err == nil {
		t.Fatal("nil automaton accepted")
	}
	if _, err := estimateStringAutomataIntersectionDifficulty(a, nil); err == nil {
		t.Fatal("nil automaton accepted")
	}
}
