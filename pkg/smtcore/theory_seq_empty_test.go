package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTEmptySequenceTheoryScopesAndModelFactory(t *testing.T) {
	m := astTestManager(t)
	_, err := m.MkString("")
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTEmptySequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	if theory.Name() != "seq-empty" || theory.UseDisequalities() {
		t.Fatal("empty sequence theory metadata", theory.Name(), theory.UseDisequalities())
	}
	status, err := theory.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("unused sequence theory", status)
	}
	astTestMust(t, state.PushScope())
	if internalized, e := theory.InternalizeTerm(ast.ASTHandle{}); e != nil || internalized {
		t.Fatal("empty sequence term", internalized, e)
	}
	status, err = theory.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckGiveUp {
		t.Fatal("used sequence theory", status)
	}
	astTestMust(t, state.PopScopes(1))
	status, err = theory.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("sequence usage did not roll back", status)
	}
	fresh, err := theory.Fresh()
	astTestMust(t, err)
	if fresh == theory || fresh.FamilyID() != family {
		t.Fatal("fresh sequence theory", fresh)
	}
}
