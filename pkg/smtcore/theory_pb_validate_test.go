package smtcore

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func smtPBExpectDebugGuard(t *testing.T, err error) {
	t.Helper()
	if smtcoreSourceDebug {
		if !errors.Is(err, ast.ErrASTDomain) {
			t.Fatal("source debug assertion did not refuse", err)
		}
	} else if err != nil {
		t.Fatal("erased source debug call touched its operands", err)
	}
}

func TestSMTPBDebugWatchAndAssignmentValidators(t *testing.T) {
	_, s, pb, atom, inputs := smtPBWatchFixture(t, []int64{4, 3, 2, 2, 1}, 6, false)
	smtPBWatchAssign(t, s, atom)
	c := pb.constraints[0]
	astTestMust(t, pb.validateWatch(c))
	sum := new(big.Rat).Set(&c.watchSum)
	c.watchSum.Add(&c.watchSum, big.NewRat(1, 1))
	smtPBExpectDebugGuard(t, pb.validateWatch(c))
	c.watchSum.Set(sum)
	maximum := new(big.Rat).Set(&c.maxWatch)
	c.maxWatch.Add(&c.maxWatch, big.NewRat(1, 1))
	smtPBExpectDebugGuard(t, pb.validateWatch(c))
	c.maxWatch.Set(maximum)
	astTestMust(t, s.AssignLiteral(inputs[0].Negated(), NewSMTBooleanAxiom()))
	premises := []SMTLiteral{inputs[0].Negated(), atom}
	astTestMust(t, pb.validateAssign(c, premises, inputs[1]))
	smtPBExpectDebugGuard(t, pb.validateAssign(c, []SMTLiteral{inputs[0], atom}, inputs[1]))
	smtPBExpectDebugGuard(t, pb.validateAssign(c, []SMTLiteral{atom}, inputs[1]))
	smtPBExpectDebugGuard(t, pb.validateAssign(c, premises, inputs[0]))
	_, err := s.PropagateCore()
	astTestMust(t, err)
	if s.Inconsistent() {
		t.Fatal("valid weighted propagation rejected after source checks")
	}
}

func TestSMTPBDebugVariableSumsAndFinalCheckRelevance(t *testing.T) {
	for _, equality := range []bool{false, true} {
		_, s, pb, atom, inputs := smtPBWatchFixture(t, []int64{1, 1}, 2, equality)
		c := pb.constraints[0]
		astTestMust(t, pb.validateFinalConstraint(c)) // undefined predicate skips
		expression, err := s.BoolExpression(atom.Var())
		astTestMust(t, err)
		astTestMust(t, s.context.SetRelevant(expression, true))
		astTestMust(t, s.SetAssignment(atom.Var(), lbool.True))
		smtPBExpectDebugGuard(t, pb.validateFinalConstraint(c))
		astTestMust(t, s.context.SetRelevant(expression, false))
		astTestMust(t, pb.validateFinalConstraint(c)) // irrelevant predicate skips
		astTestMust(t, s.context.SetRelevant(expression, true))
		for _, l := range inputs {
			astTestMust(t, s.SetAssignment(l.Var(), lbool.True))
		}
		astTestMust(t, pb.validateVariableSums(c)) // delayed counters are conservative
		c.minSum.SetInt64(3)
		smtPBExpectDebugGuard(t, pb.validateVariableSums(c))
		c.minSum.SetInt64(0)
		c.maxSum.SetInt64(1)
		smtPBExpectDebugGuard(t, pb.validateVariableSums(c))
		c.maxSum.SetInt64(2)
		_, err = pb.FinalCheck()
		astTestMust(t, err)
		astTestMust(t, s.SetAssignment(atom.Var(), lbool.False))
		_, err = pb.FinalCheck()
		smtPBExpectDebugGuard(t, err)
	}
	if !smtcoreSourceDebug {
		var absent *SMTPseudoBooleanTheory
		astTestMust(t, absent.validateWatch(nil))
		astTestMust(t, absent.validateAssign(nil, nil, SMTNullLiteral))
		astTestMust(t, absent.validateVariableSums(nil))
		astTestMust(t, absent.validateFinalConstraint(nil))
	}
}
