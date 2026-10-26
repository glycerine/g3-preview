package smtcore

import (
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTStringConcatUnrollSplitTruthAndFreshOrder(t *testing.T) {
	m, theory, state, rw := stringLibraryFixture(t)
	left := stringGroundingWord(t, theory, "concat-unroll-left")
	right := stringGroundingWord(t, theory, "concat-unroll-right")
	concat := stringDependencyConcat(t, theory, left, right)
	count := stringLibraryInteger(t, m, state, "concat-unroll-count")
	unroll, err := theory.makeStringUnroll(stringRegexLiteral(t, m, "ab"), count)
	astTestMust(t, err)
	astTestMust(t, theory.processStringConcatUnroll(concat, unroll))
	formula := theory.testers.concatUnrollDefinitions[[2]ast.ASTHandle{concat, unroll}]
	if formula.IsNull() || theory.freshID != 7 {
		t.Fatal("concat split lost five word/two count witness order")
	}
	var parts [5]ast.ASTHandle
	for i := range parts {
		parts[i] = stringLibraryWitness(t, theory, "regex!tmp"+strconv.Itoa(i))
	}
	firstCount := stringLibraryWitness(t, theory, "unroll!tmp5")
	secondCount := stringLibraryWitness(t, theory, "unroll!tmp6")
	var firstUnroll, secondUnroll ast.ASTHandle
	for _, ref := range theory.refs {
		term, err := m.RefGet(ref)
		astTestMust(t, err)
		op, own, err := theory.operation(term)
		astTestMust(t, err)
		if !own || op != 43 || term == unroll {
			continue
		}
		bound, err := m.GetArg(term, 1)
		astTestMust(t, err)
		if bound == firstCount {
			firstUnroll = term
		} else if bound == secondCount {
			secondUnroll = term
		}
	}
	if firstUnroll.IsNull() || secondUnroll.IsNull() {
		t.Fatal("concat split bypassed actual residual unroll constructors")
	}
	for n := 0; n <= 3; n++ {
		for c1 := 0; c1 <= 1; c1++ {
			for c2 := 0; c2 <= 1; c2++ {
				for _, l := range []string{"", "a"} {
					for _, r := range []string{"", "bab"} {
						for _, boundary := range []string{"ab", "bad"} {
							number, err := m.ArithmeticIntegerNumeralText(strconv.Itoa(n))
							astTestMust(t, err)
							first, err := m.ArithmeticIntegerNumeralText(strconv.Itoa(c1))
							astTestMust(t, err)
							second, err := m.ArithmeticIntegerNumeralText(strconv.Itoa(c2))
							astTestMust(t, err)
							values := map[ast.ASTHandle]ast.ASTHandle{left: astTestString(t, m, l), right: astTestString(t, m, r), count: number, firstCount: first, secondCount: second, firstUnroll: astTestString(t, m, ""), secondUnroll: astTestString(t, m, "ab")}
							for i, text := range []string{"", "a", "b", "ab", boundary} {
								values[parts[i]] = astTestString(t, m, text)
							}
							body := l == "a" && r == "bab" && boundary == "ab" && c1+c2 == n-1
							want := ((n == 0) == (l == "" && r == "")) && ((n >= 1) == body)
							if got := stringAxiomTruth(t, m, rw, formula, values); got != want {
								t.Fatal("concat/unroll split", n, c1, c2, l, r, boundary, got, want)
							}
						}
					}
				}
			}
		}
	}
}

func TestSMTStringConcatUnrollCacheScopeRootsAndFresh(t *testing.T) {
	m, theory, state, _ := stringLibraryFixture(t)
	left := stringGroundingWord(t, theory, "concat-cache-left")
	right := stringGroundingWord(t, theory, "concat-cache-right")
	concat := stringDependencyConcat(t, theory, left, right)
	count := stringLibraryInteger(t, m, state, "concat-cache-count")
	unroll := stringUnrollTerm(t, theory, stringRegexLiteral(t, m, "x"), count)
	astTestMust(t, state.PushScope())
	astTestMust(t, theory.processStringConcatUnroll(concat, unroll))
	key := [2]ast.ASTHandle{concat, unroll}
	formula := theory.testers.concatUnrollDefinitions[key]
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
		t.Fatal("initial concat/unroll definition lost explicit/assertion roots")
	}
	fresh := theory.freshID
	astTestMust(t, state.PopScopes(1))
	astTestMust(t, theory.processStringConcatUnroll(concat, unroll))
	if theory.freshID != fresh || theory.testers.concatUnrollDefinitions[key] != formula || countRoots() != 3 {
		t.Fatal("cache replay should add only its assertion root")
	}
	owner, err := theory.Fresh()
	astTestMust(t, err)
	defer func() { astTestMust(t, owner.Close()) }()
	if len(owner.testers.concatUnrollDefinitions) != 0 {
		t.Fatal("Fresh copied concat/unroll definitions")
	}
}

func TestSMTStringConcatUnrollRangeBoundaryAssertionAndCacheIdentity(t *testing.T) {
	m, theory, state, rw := stringLibraryFixture(t)
	left := stringGroundingWord(t, theory, "concat-range-left")
	right := stringGroundingWord(t, theory, "concat-range-right")
	concat := stringDependencyConcat(t, theory, left, right)
	count := stringLibraryInteger(t, m, state, "concat-range-count")
	regex, err := m.MkRegexRange(astTestString(t, m, "a"), astTestString(t, m, "b"))
	astTestMust(t, err)
	unroll := stringUnrollTerm(t, theory, regex, count)
	before := len(theory.refs)
	astTestMust(t, theory.processStringConcatUnroll(concat, unroll))
	boundary := stringLibraryWitness(t, theory, "regex!tmp4")
	var assertion ast.ASTHandle
	for _, ref := range theory.refs[before:] {
		term, err := m.RefGet(ref)
		astTestMust(t, err)
		kind, err := m.GetDeclKind(term)
		astTestMust(t, err)
		if kind == ast.ASTOpOr {
			assertion = term
		}
	}
	if assertion.IsNull() {
		t.Fatal("range constraint was hidden inside positive-count guard")
	}
	for _, text := range []string{"", "a", "b", "c"} {
		if got := stringAxiomTruth(t, m, rw, assertion, map[ast.ASTHandle]ast.ASTHandle{boundary: astTestString(t, m, text)}); got != (text == "a" || text == "b") {
			t.Fatal("unconditional boundary range", text, got)
		}
	}
	other := stringDependencyConcat(t, theory, right, left)
	astTestMust(t, theory.processStringConcatUnroll(other, unroll))
	if len(theory.testers.concatUnrollDefinitions) != 2 || theory.freshID != 14 {
		t.Fatal("ordered concat cache keys were incorrectly treated as symmetric")
	}
}
