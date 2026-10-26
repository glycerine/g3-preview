package smtcore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTDummyTheoryNumeralsTriggerScopedIncompleteness(t *testing.T) {
	for _, integer := range []bool{false, true} {
		t.Run(fmt.Sprint(integer), func(t *testing.T) {
			m := astTestManager(t)
			family, err := m.RegisterArithmeticPlugin()
			astTestMust(t, err)
			dummy, err := NewSMTDummyTheory(family, "no arithmetic")
			astTestMust(t, err)
			_, _, state := smtInternalizerFixture(t, m, nil)
			astTestMust(t, state.RegisterTheory(dummy))
			value, err := m.ArithmeticRealNumeralText("3")
			if integer {
				value, err = m.ArithmeticIntegerNumeralText("3")
			}
			astTestMust(t, err)
			for repeat := 0; repeat < 2; repeat++ {
				astTestMust(t, state.PushScope())
				astTestMust(t, state.Internalize(value, false))
				astTestMust(t, state.Internalize(value, false))
				status, err := dummy.FinalCheck()
				astTestMust(t, err)
				if status != SMTFinalCheckGiveUp {
					t.Fatal("numeral escaped disabled theory", status)
				}
				astTestMust(t, state.PopScopes(1))
				status, err = dummy.FinalCheck()
				astTestMust(t, err)
				if status != SMTFinalCheckDone {
					t.Fatal("numeral incompleteness survived pop", status)
				}
			}
		})
	}
}

func TestSMTDummyTheoryTracksExpressionsScopesResetAndFresh(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	dummy, err := NewSMTDummyTheory(family, "no arithmetic")
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(dummy))
	if dummy.Name() != "no arithmetic" || dummy.BuildModels() || dummy.UseDisequalities() {
		t.Fatal("dummy metadata", dummy.Name(), dummy.BuildModels(), dummy.UseDisequalities())
	}
	status, err := dummy.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("initial final check", status)
	}
	astTestMust(t, state.PushScope())
	if internalized, err := dummy.InternalizeTerm(ast.ASTHandle{}); err != nil || internalized {
		t.Fatal("dummy term", internalized, err)
	}
	status, err = dummy.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckGiveUp {
		t.Fatal("expression final check", status)
	}
	astTestMust(t, state.PopScopes(1))
	status, err = dummy.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("scope rollback final check", status)
	}
	if err = dummy.NewEquality(0, 1); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("unreachable equality callback", err)
	}
	if err = dummy.NewDisequality(0, 1); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("unreachable disequality callback", err)
	}
	fresh, err := dummy.Fresh()
	astTestMust(t, err)
	if fresh == dummy || fresh.Name() != dummy.Name() || fresh.FamilyID() != family {
		t.Fatal("fresh dummy", fresh)
	}
	astTestMust(t, dummy.Reset())
	status, err = dummy.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckGiveUp {
		t.Fatal("reset final check", status)
	}
}
