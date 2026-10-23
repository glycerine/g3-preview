package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTModelCheckerInversePreservesGenerationBeforeFingerprintCanonicalization(t *testing.T) {
	for _, fromInverse := range []bool{false, true} {
		name := "unrestricted-context-lookup"
		if fromInverse {
			name = "recovered-inverse-term"
		}
		t.Run(name, func(t *testing.T) {
			m, state, registry, _ := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{})
			seven, err := m.ArithmeticIntegerNumeralText("7")
			astTestMust(t, err)
			three, err := m.ArithmeticIntegerNumeralText("3")
			astTestMust(t, err)
			integer, err := m.GetSort(seven)
			astTestMust(t, err)
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			predicate := smtQuickPredicate(t, m, "inverse-generation-p", integer, boolean)
			x, err := m.MkVar(0, integer)
			astTestMust(t, err)
			body, err := m.MkApp(predicate, []ast.ASTHandle{x})
			astTestMust(t, err)
			q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{integer}, Names: []symbol.Symbol{astTestSymbol(t, m, "inverse-generation-x")}, Body: body})
			astTestMust(t, err)
			astTestMust(t, state.Internalize(q, true))
			original, err := NewSMTEnode(m, seven, state.context.enodes, SMTEnodeOptions{Generation: 8})
			astTestMust(t, err)
			astTestMust(t, state.context.RegisterEnode(original, true))
			representative, err := NewSMTEnode(m, three, state.context.enodes, SMTEnodeOptions{Generation: 1})
			astTestMust(t, err)
			astTestMust(t, state.context.RegisterEnode(representative, true))
			original.root = representative
			checker, err := NewSMTModelChecker(state, 1)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, checker.Close()) })
			checker.valueTerms = map[ast.ASTHandle]ast.ASTHandle{seven: three}
			inserted, err := checker.enqueueCounterexample(q, []ast.ASTHandle{seven}, fromInverse)
			astTestMust(t, err)
			if !inserted || len(registry.schedule.newEntries) != 1 {
				t.Fatal("counterexample was not queued")
			}
			entry := registry.schedule.newEntries[0]
			want := uint32(1)
			if fromInverse {
				want = 8
			}
			if entry.Generation != want {
				t.Fatal("inverse generation replaced before fingerprint construction", entry.Generation, want)
			}
			if len(entry.Fingerprint.args) != 1 || entry.Fingerprint.args[0] != representative {
				t.Fatal("fingerprint canonicalization changed")
			}
		})
	}
}
