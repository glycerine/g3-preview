package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTSequenceLengthCoherenceRequiresSolutionRoot(t *testing.T) {
	f := newSMTSequenceInterfaceFixture(t)
	family, err := f.m.ArithmeticFamilyID()
	astTestMust(t, err)
	arithmetic, err := NewSMTLinearArithmeticTheory(family)
	astTestMust(t, err)
	astTestMust(t, f.state.RegisterTheory(arithmetic))
	length, err := f.m.MkSequenceLength(f.x)
	astTestMust(t, err)
	minimum, err := f.m.ArithmeticIntegerNumeralText("2")
	astTestMust(t, err)
	lower, err := f.m.MkArithmeticGE(length, minimum)
	astTestMust(t, err)
	astTestMust(t, f.state.Internalize(lower, true))
	variable, found := f.state.BoolVar(lower)
	if !found {
		t.Fatal("length bound was not internalized")
	}
	astTestMust(t, arithmetic.Assign(variable, true))
	astTestMust(t, arithmetic.Propagate())
	for attempt := 0; attempt < 2; attempt++ {
		astTestMust(t, f.state.PushScope())
		astTestMust(t, f.state.PushScope())
		astTestMust(t, f.theory.solutions.Update(f.x, f.y, nil))
		before := f.state.searchProgress()
		axioms := len(f.theory.axioms.values)
		progress, err := f.theory.addLengthLowerBoundDecomposition()
		astTestMust(t, err)
		if progress || f.state.searchProgress() != before || len(f.theory.axioms.values) != axioms {
			t.Fatal("already solved sequence acquired a lower-bound decomposition")
		}
		astTestMust(t, f.state.PopScopes(1))
		progress, err = f.theory.addLengthLowerBoundDecomposition()
		astTestMust(t, err)
		if !progress || len(f.theory.axioms.values) <= axioms {
			t.Fatal("restored solution root did not acquire a lower-bound decomposition")
		}
		astTestMust(t, f.state.PopScopes(1))
	}
}

func TestSMTSequenceFixedLengthSkolemEligibilityAndReplay(t *testing.T) {
	for _, name := range []string{"seq.tail", "seq.first", "seq.idx.left", "seq.idx.right", "seq.pre", "seq.post", "seq.left", "seq.right"} {
		t.Run(name, func(t *testing.T) {
			f := newSMTSequenceInterfaceFixture(t)
			zero := astTestInteger(t, f.m, 0)
			arguments := []ast.ASTHandle{f.x, zero}
			if name == "seq.first" {
				arguments = []ast.ASTHandle{f.x}
			}
			if name == "seq.idx.left" || name == "seq.idx.right" || name == "seq.left" || name == "seq.right" {
				arguments = []ast.ASTHandle{f.x, f.y}
			}
			symbol, err := f.m.Symbols.Intern(name)
			astTestMust(t, err)
			word, err := f.m.MkSequenceSkolem(symbol, arguments, f.sort)
			astTestMust(t, err)
			family, err := f.m.ArithmeticFamilyID()
			astTestMust(t, err)
			arithmetic, err := NewSMTLinearArithmeticTheory(family)
			astTestMust(t, err)
			astTestMust(t, f.state.RegisterTheory(arithmetic))
			length, err := f.m.MkSequenceLength(word)
			astTestMust(t, err)
			two := astTestInteger(t, f.m, 2)
			lower, err := f.m.MkArithmeticGE(length, two)
			astTestMust(t, err)
			upper, err := f.m.MkArithmeticLE(length, two)
			astTestMust(t, err)
			for _, bound := range []ast.ASTHandle{lower, upper} {
				astTestMust(t, f.state.Internalize(bound, true))
				variable, found := f.state.BoolVar(bound)
				if !found {
					t.Fatal("Skolem length bound was not internalized")
				}
				astTestMust(t, arithmetic.Assign(variable, true))
			}
			for attempt := 0; attempt < 2; attempt++ {
				astTestMust(t, f.state.PushScope())
				before := len(f.theory.axioms.values)
				progress, err := f.theory.addFixedLengthDecomposition()
				astTestMust(t, err)
				want := name == "seq.pre" || name == "seq.post" || name == "seq.left" || name == "seq.right"
				if progress != want || (len(f.theory.axioms.values) > before) != want {
					t.Fatal("fixed-length Skolem eligibility", progress, "want", want)
				}
				progress, err = f.theory.addFixedLengthDecomposition()
				astTestMust(t, err)
				if progress {
					t.Fatal("fixed-length decomposition repeated in the same scope")
				}
				astTestMust(t, f.state.PopScopes(1))
			}
		})
	}
}
