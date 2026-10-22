package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTDelayedUnitBackjumpGuard(t *testing.T) {
	m := astTestManager(t)
	kernel, err := NewSMTKernel(m, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, kernel.Close()) })
	kernel.smtState.DelayUnits = true
	kernel.smtState.DelayUnitsThreshold = 2
	quantifier := smtRelevancyBool(t, m, "delayed-unit-quantifier-sentinel")
	kernel.internalizer.quantifiers = &SMTQuantifierRegistry{manager: m, quantifiers: []ast.ASTHandle{quantifier}}
	resolution := SMTConflictResolution{
		Resolved: true, ConflictLevel: 4, NewScopeLevel: 0,
		Lemma: []SMTLiteral{NewSMTLiteral(SMTFirstBoolVar, false)},
	}
	level, err := kernel.learnedBackjumpLevel(resolution, 0)
	astTestMust(t, err)
	if level != 3 {
		t.Fatal("delayed-unit backjump", level)
	}

	kernel.internalizer.learnedUnits = make([]smtPersistentLearnedUnit, 2)
	level, err = kernel.learnedBackjumpLevel(resolution, 0)
	astTestMust(t, err)
	if level != 0 {
		t.Fatal("delayed-unit threshold ignored", level)
	}
	kernel.internalizer.learnedUnits = nil

	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	level, err = kernel.learnedBackjumpLevel(resolution, 0)
	astTestMust(t, err)
	if level != 0 {
		t.Fatal("proof-producing search delayed a learned unit", level)
	}
}
