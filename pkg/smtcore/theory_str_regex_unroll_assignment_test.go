package smtcore

import (
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTStringRegexUnrollAssignmentOneStepTruth(t *testing.T) {
	m, theory, state, rw := stringLibraryFixture(t)
	count := stringLibraryInteger(t, m, state, "regex-unroll-count")
	regex := stringRegexLiteral(t, m, "ab")
	unroll, err := theory.makeStringUnroll(regex, count)
	astTestMust(t, err)
	astTestMust(t, theory.assignStringRegexUnrolls([]ast.ASTHandle{unroll, unroll}))
	formula := theory.testers.unrollDefinitions[unroll]
	if formula.IsNull() || theory.freshID != 3 {
		t.Fatal("one-step unfolding or exact-set deduplication missing")
	}
	first := stringLibraryWitness(t, theory, "regex!tmp0")
	second := stringLibraryWitness(t, theory, "regex!tmp1")
	nextCount := stringLibraryWitness(t, theory, "unroll!tmp2")
	var nextUnroll ast.ASTHandle
	for _, ref := range theory.refs {
		term, err := m.RefGet(ref)
		astTestMust(t, err)
		op, own, err := theory.operation(term)
		astTestMust(t, err)
		if own && op == 43 && term != unroll {
			nextUnroll = term
		}
	}
	if nextUnroll.IsNull() {
		t.Fatal("unfolding bypassed actual unroll constructor")
	}
	for n := 0; n <= 3; n++ {
		for remaining := 0; remaining <= 2; remaining++ {
			for _, whole := range []string{"", "ab", "abab", "bad"} {
				for _, head := range []string{"ab", "bad"} {
					for _, tail := range []string{"", "ab", "bad"} {
						c, err := m.ArithmeticIntegerNumeralText(strconv.Itoa(n))
						astTestMust(t, err)
						r, err := m.ArithmeticIntegerNumeralText(strconv.Itoa(remaining))
						astTestMust(t, err)
						values := map[ast.ASTHandle]ast.ASTHandle{unroll: astTestString(t, m, whole), first: astTestString(t, m, head), second: astTestString(t, m, tail), nextUnroll: astTestString(t, m, tail), count: c, nextCount: r}
						body := whole == head+tail && head == "ab" && n == remaining+1
						want := ((n == 0) == (whole == "")) && ((n >= 1) == body)
						if got := stringAxiomTruth(t, m, rw, formula, values); got != want {
							t.Fatal("one-step unroll equivalences", n, remaining, whole, head, tail, got, want)
						}
					}
				}
			}
		}
	}
}

func TestSMTStringRegexUnrollCacheScopeReplayAndFresh(t *testing.T) {
	m, theory, state, _ := stringLibraryFixture(t)
	count := stringLibraryInteger(t, m, state, "regex-cache-count")
	unroll := stringUnrollTerm(t, theory, stringRegexLiteral(t, m, "x"), count)
	astTestMust(t, state.PushScope())
	astTestMust(t, theory.assignStringRegexUnrolls([]ast.ASTHandle{unroll}))
	formula := theory.testers.unrollDefinitions[unroll]
	fresh := theory.freshID
	countRoots := func() int {
		n := 0
		for _, ref := range theory.refs {
			term, err := m.RefGet(ref)
			astTestMust(t, err)
			if term == formula {
				n++
			}
		}
		return n
	}
	if countRoots() != 2 {
		t.Fatal("definition did not retain both explicit and assertion roots")
	}
	astTestMust(t, state.PopScopes(1))
	astTestMust(t, theory.assignStringRegexUnrolls([]ast.ASTHandle{unroll}))
	if theory.freshID != fresh || theory.testers.unrollDefinitions[unroll] != formula || countRoots() != 4 {
		t.Fatal("cached replay recreated witnesses or lost repeated native roots")
	}
	owner, err := theory.Fresh()
	astTestMust(t, err)
	defer func() { astTestMust(t, owner.Close()) }()
	if len(owner.testers.unrollDefinitions) != 0 {
		t.Fatal("Fresh copied unroll definitions")
	}
}

func TestSMTStringRegexUnrollNegativeLogicalBoundPrecedesCache(t *testing.T) {
	m, theory, state, _ := stringRegexStageFixture(t)
	count := stringLibraryInteger(t, m, state, "regex-negative-count")
	unroll := stringUnrollTerm(t, theory, stringRegexLiteral(t, m, "x"), count)
	astTestMust(t, state.PushScope())
	stringAssertArithmeticBound(t, m, state, count, smtArithOpGE, "0", false)
	astTestMust(t, theory.assignStringRegexUnrolls([]ast.ASTHandle{unroll}))
	cached := theory.testers.unrollDefinitions[unroll]
	astTestMust(t, state.PopScopes(1))
	stringAssertArithmeticBound(t, m, state, count, smtArithOpGE, "-2", false)
	stringCheckArithmeticBound(t, theory, count, true, "-2")
	fresh := theory.freshID
	astTestMust(t, theory.assignStringRegexUnrolls([]ast.ASTHandle{unroll}))
	last, err := m.RefGet(theory.refs[len(theory.refs)-1])
	astTestMust(t, err)
	zero, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	want, err := m.MkArithmeticGE(count, zero)
	astTestMust(t, err)
	if last != want || theory.freshID != fresh || theory.testers.unrollDefinitions[unroll] != cached {
		t.Fatal("negative logical endpoint did not override cache without replacing it")
	}
}
