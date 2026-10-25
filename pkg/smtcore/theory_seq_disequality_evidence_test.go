package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTSequenceResidualDisequalityRetainsCancellationEvidence(t *testing.T) {
	m := astTestManager(t)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	text, err := m.MkString("")
	astTestMust(t, err)
	sort, err := m.GetSort(text)
	astTestMust(t, err)
	var words []ast.ASTHandle
	for _, name := range []string{"premise-left", "premise-right", "residual-left", "residual-right"} {
		word, err := m.MkConst(astTestSymbol(t, m, name), sort)
		astTestMust(t, err)
		words = append(words, word)
		astTestMust(t, state.Internalize(word, false))
	}
	left, err := m.MkSequenceConcat(words[0], words[2])
	astTestMust(t, err)
	right, err := m.MkSequenceConcat(words[1], words[3])
	astTestMust(t, err)
	original, err := m.MkEq(left, right)
	astTestMust(t, err)
	negative, err := m.MkNot(original)
	astTestMust(t, err)
	residual, err := m.MkEq(words[2], words[3])
	astTestMust(t, err)
	astTestMust(t, state.Internalize(original, true))
	astTestMust(t, state.Internalize(residual, true))
	originalLiteral, err := state.Literal(original)
	astTestMust(t, err)
	residualLiteral, err := state.Literal(residual)
	astTestMust(t, err)
	for attempt := 0; attempt < 2; attempt++ {
		astTestMust(t, state.PushScope())
		astTestMust(t, state.Assert(negative, ast.ASTHandle{}, 0))
		astTestMust(t, state.addEquality(state.context.enodes[words[0]], state.context.enodes[words[1]], NewSMTEqualityAxiom()))
		_, err = state.PropagateCore()
		astTestMust(t, err)
		progress, err := theory.checkReducedDisequations()
		astTestMust(t, err)
		value, err := state.Assignment(residualLiteral)
		astTestMust(t, err)
		if !progress || value != lbool.False {
			t.Fatal("residual disequality was not propagated with its cancellation evidence", progress, value)
		}
		data, err := state.BooleanData(residualLiteral.Var())
		astTestMust(t, err)
		owner, ok := data.Justification().Justification()
		if !ok {
			t.Fatal("residual disequality has no theory explanation")
		}
		extended, ok := owner.(*SMTExtTheoryPropagationJustification)
		if !ok || len(extended.Literals) != 1 || extended.Literals[0] != originalLiteral.Negated() || len(extended.Equalities) != 1 {
			t.Fatal("residual disequality lost literal or equality premises", owner)
		}
		pair := extended.Equalities[0]
		if pair.First.root != pair.Second.root || pair.First.owner != words[0] && pair.First.owner != words[1] || pair.Second.owner != words[0] && pair.Second.owner != words[1] {
			t.Fatal("residual disequality uses unrelated cancellation evidence")
		}
		if progress, err := theory.checkReducedDisequations(); err != nil || progress {
			t.Fatal("already propagated residual reported progress", progress, err)
		}
		astTestMust(t, state.PopScopes(1))
		if value, err := state.Assignment(residualLiteral); err != nil || value != lbool.Undef || state.context.enodes[words[0]].root == state.context.enodes[words[1]].root {
			t.Fatal("scoped residual propagation or cancellation equality survived pop", value, err)
		}
	}
}
