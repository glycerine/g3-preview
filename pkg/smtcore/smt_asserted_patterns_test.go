package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTAssertedPatternInferenceGuardsAndProofs(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, mode := range []string{"enabled", "no-ematching", "no-preprocessing"} {
			t.Run(fmt.Sprint(proofs, "/", mode), func(t *testing.T) {
				m, solver := smtAssertedNNFFixture(t, proofs)
				u, err := m.MkUninterpretedSort(astTestSymbol(t, m, "infer-U"), nil)
				astTestMust(t, err)
				x, err := m.MkVar(0, u)
				astTestMust(t, err)
				b, err := m.MkBoolSort()
				astTestMust(t, err)
				p := smtQuickPredicate(t, m, "infer-p", u, b)
				body, err := m.MkApp(p, []ast.ASTHandle{x})
				astTestMust(t, err)
				q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{u}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: body, Weight: 1})
				astTestMust(t, err)
				astTestMust(t, solver.Assert(q))
				k := solver.kernel
				k.smtState.Ematching = mode != "no-ematching"
				k.preprocess = mode != "no-preprocessing"
				astTestMust(t, k.inferPendingPatterns())
				row := k.assertions[0]
				n, err := m.GetNumPatterns(row.Formula)
				astTestMust(t, err)
				want := uint32(0)
				if mode == "enabled" {
					want = 1
				}
				if n != want {
					t.Fatal("inferred pattern count", n, want)
				}
				if n != 0 {
					pattern, err := m.GetPattern(row.Formula, 0)
					astTestMust(t, err)
					term, err := m.GetArg(pattern, 0)
					astTestMust(t, err)
					if term != body {
						t.Fatal("wrong inferred trigger")
					}
				}
				if proofs {
					fact, err := m.GetProofFact(row.Proof)
					astTestMust(t, err)
					if fact != row.Formula {
						t.Fatal("inference lost the assertion proof")
					}
				}
				// A repeated pass preserves an already inferred pattern.
				before := row.Formula
				astTestMust(t, k.inferPendingPatterns())
				if k.assertions[0].Formula != before {
					t.Fatal("inferred trigger changed on repeated pass")
				}
			})
		}
	}
}
