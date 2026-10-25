package smtcore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTAssertionConflictProofAllocationFrontiers(t *testing.T) {
	succeeded := false
	for stop := int64(0); stop < 128 && !succeeded; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m := astTestManager(t)
			astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
			k := smtTestKernel(t, m)
			p := smtRelevancyBool(t, m, "stopped-conflict-prefix")
			astTestMust(t, k.Assert(p, ast.ASTHandle{}))
			falsity, err := m.MkFalse()
			astTestMust(t, err)
			astTestMust(t, k.Assert(falsity, ast.ASTHandle{}))
			m.Budget().FailAfter(stop)
			err = k.internalizePending()
			m.Budget().FailAfter(-1)
			if err == nil {
				succeeded = true
			} else {
				if !errors.Is(err, arena.ErrInjectedFailure) || k.Inconsistent() {
					t.Fatal("stopped proof owner partially published a conflict", err)
				}
				astTestMust(t, k.internalizePending())
			}
			if _, found := k.internalizer.BoolVar(p); found || k.assertedHead != 0 || !k.Inconsistent() {
				t.Fatal("proof construction or retry internalized the pending prefix")
			}
		})
		if t.Failed() {
			t.Fatalf("proof allocation frontier %d", stop)
		}
	}
	if !succeeded {
		t.Fatal("no successful conflict proof allocation frontier")
	}
}

func TestSMTAssertionConflictSkipsPrefixAndPreservesProofScopes(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		m := astTestManager(t)
		if proofs {
			astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
		}
		k := smtTestKernel(t, m)
		astTestMust(t, k.Push())
		p := smtRelevancyBool(t, m, "early-conflict-prefix")
		astTestMust(t, k.Assert(p, ast.ASTHandle{}))
		falsity, err := m.MkFalse()
		astTestMust(t, err)
		astTestMust(t, k.Assert(falsity, ast.ASTHandle{}))
		expected, err := k.AssertionInconsistencyProof()
		astTestMust(t, err)
		astTestMust(t, k.internalizePending())
		if _, found := k.internalizer.BoolVar(p); found || k.assertedHead != 0 || !k.Inconsistent() {
			t.Fatal("source early-conflict path internalized or committed pending formulas")
		}
		proof, err := k.Proof()
		astTestMust(t, err)
		if proof != expected || proofs == proof.IsNull() {
			t.Fatal("preprocessing conflict did not publish its original proof")
		}
		conflicts := k.Statistics().NumConflicts
		astTestMust(t, k.internalizePending())
		if k.Statistics().NumConflicts != conflicts {
			t.Fatal("existing Boolean conflict was overwritten")
		}
		status, err := k.Check()
		astTestMust(t, err)
		if status != lbool.False {
			t.Fatal("direct assertion conflict did not return UNSAT", status)
		}
		if proofs {
			proof, err = k.Proof()
			astTestMust(t, err)
			fact, err := m.GetProofFact(proof)
			astTestMust(t, err)
			if !m.IsFalse(fact) {
				t.Fatal("search lost the false conflict proof")
			}
		}
		astTestMust(t, k.Push())
		if k.assertedHead != len(k.assertions) {
			t.Fatal("inconsistent scope push did not commit its assertion cursor")
		}
		astTestMust(t, k.Pop(1))
		if !k.Inconsistent() {
			t.Fatal("inner pop discarded the outer conflict")
		}
		astTestMust(t, k.Pop(1))
		status, err = k.Check()
		astTestMust(t, err)
		if status != lbool.True || k.Inconsistent() || len(k.assertions) != 0 {
			t.Fatal("outer pop did not restore a satisfiable context", status)
		}
	}
}

func TestSMTAssertionConflictInitialCancellationAndAxiomFallback(t *testing.T) {
	m := astTestManager(t)
	k := smtTestKernel(t, m)
	p := smtRelevancyBool(t, m, "axiom-conflict-prefix")
	astTestMust(t, k.Assert(p, ast.ASTHandle{}))
	np, err := m.MkNot(p)
	astTestMust(t, err)
	astTestMust(t, k.Assert(np, ast.ASTHandle{}))
	// A canceled local simplifier may establish the source eager flag while
	// its contradictory original vector remains published without a false row.
	k.assertedInconsistent = true
	m.Limit.Cancel()
	before := m.Limit.Count()
	astTestMust(t, k.internalizePending())
	if k.Inconsistent() || m.Limit.Count() != before+1 || k.assertedHead != 0 {
		t.Fatal("initial get_cancel_flag did not precede the inconsistent-manager branch")
	}
	m.Limit.ResetCancel()
	m.Budget().FailAfter(0)
	err = k.internalizePending()
	m.Budget().FailAfter(-1)
	astTestMust(t, err)
	conflict, found := k.internalizer.Conflict()
	if !found || conflict.Kind() != SMTBooleanJustificationAxiom {
		t.Fatal("missing false row/proof did not select the source axiom conflict")
	}
	if _, found := k.internalizer.BoolVar(p); found {
		t.Fatal("axiom fallback traversed the original contradictory vector")
	}
}

type smtCancelFirstAssertionTheory struct {
	*smtInternalizerTestTheory
	cancelFirst bool
}

func (t *smtCancelFirstAssertionTheory) InternalizeAtom(h ast.ASTHandle, gate bool) (bool, error) {
	ok, err := t.smtInternalizerTestTheory.InternalizeAtom(h, gate)
	if err == nil && t.cancelFirst {
		t.cancelFirst = false
		t.state.manager.Limit.Cancel()
	}
	return ok, err
}

func TestSMTAssertionCancellationCommitsOnlyCompletedPrefix(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	k := smtTestKernel(t, m)
	base, err := NewSMTTheoryBase(family)
	astTestMust(t, err)
	theory := &smtCancelFirstAssertionTheory{smtInternalizerTestTheory: &smtInternalizerTestTheory{SMTTheoryBase: base}, cancelFirst: true}
	astTestMust(t, k.RegisterTheory(theory))
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	var atoms []ast.ASTHandle
	for _, name := range []string{"cancel-first-atom", "cancel-second-atom"} {
		info, err := m.NewFuncDeclInfo(family, 9002, nil, ast.ASTFuncFlags{}, false)
		astTestMust(t, err)
		d, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, name), nil, b, info)
		astTestMust(t, err)
		astTestMust(t, m.DelFuncDeclInfo(&info))
		a, err := m.MkApp(d, nil)
		astTestMust(t, err)
		astTestMust(t, k.Assert(a, ast.ASTHandle{}))
		atoms = append(atoms, a)
	}
	astTestMust(t, k.internalizePending())
	if k.assertedHead != 1 || theory.atoms != 1 {
		t.Fatal("canceled assertion walk did not commit just its completed prefix", k.assertedHead, theory.atoms)
	}
	if _, found := k.internalizer.BoolVar(atoms[1]); found {
		t.Fatal("cancellation did not stop before the second atom")
	}
	m.Limit.ResetCancel()
	astTestMust(t, k.internalizePending())
	if k.assertedHead != 2 || theory.atoms != 2 {
		t.Fatal("retry did not resume at the uncommitted suffix", k.assertedHead, theory.atoms)
	}
}
