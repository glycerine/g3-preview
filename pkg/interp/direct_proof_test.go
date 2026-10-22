package interp

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func directProofFixture(t *testing.T) (*ast.ASTManager, *Manager, *Base, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	m, manager := interpProofManagerFixture(t)
	boolean, err := manager.BoolType()
	interpMust(t, err)
	p, err := manager.MakeVariable("direct-a", boolean)
	interpMust(t, err)
	p = interpPin(t, m, p)
	q, err := manager.MakeVariable("direct-b", boolean)
	interpMust(t, err)
	q = interpPin(t, m, q)
	base, err := NewBase(manager, []ast.ASTHandle{p, q}, nil, nil)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, base.Close()) })
	return m, manager, base, p, q
}

func TestDirectProofTermSidesAndAssumptions(t *testing.T) {
	_, manager, base, p, q := directProofFixture(t)
	mixed, err := manager.Make(OperatorAnd, p, q)
	interpMust(t, err)
	direct, err := NewDirectProofInterpolator(base, DownwardScopeRange(0), true)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, direct.Close()) })
	for _, test := range []struct {
		term ast.ASTHandle
		want TermSide
	}{{p, TermA}, {q, TermB}, {mixed, TermMixed}} {
		if got, e := direct.TermSide(test.term); e != nil || got != test.want {
			t.Fatalf("term side = %v, want %v: %v", got, test.want, e)
		}
	}
	inA, err := direct.MakeAssumption(0, []ast.ASTHandle{p, q, mixed})
	interpMust(t, err)
	if op, e := manager.Operator(inA); e != nil || op != OperatorOr {
		t.Fatal("A assumption", op, e)
	}
	args, err := manager.Args(inA)
	interpMust(t, err)
	if len(args) != 2 || args[0] != q || args[1] != mixed {
		t.Fatal("A assumption clause", args)
	}
	inB, err := direct.MakeAssumption(1, []ast.ASTHandle{p, q, mixed})
	interpMust(t, err)
	if truth, e := manager.IsTrue(inB); e != nil || !truth {
		t.Fatal("B assumption", truth, e)
	}
}

func TestDirectProofHypothesisReflexivityAndContra(t *testing.T) {
	_, manager, base, p, q := directProofFixture(t)
	direct, err := NewDirectProofInterpolator(base, DownwardScopeRange(0), false)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, direct.Close()) })
	hypA, err := direct.MakeHypothesis(p)
	interpMust(t, err)
	if no, e := manager.IsFalse(hypA); e != nil || !no {
		t.Fatal("A hypothesis", no, e)
	}
	hypB, err := direct.MakeHypothesis(q)
	interpMust(t, err)
	if yes, e := manager.IsTrue(hypB); e != nil || !yes {
		t.Fatal("B hypothesis", yes, e)
	}
	mixed, err := manager.Make(OperatorEqual, p, q)
	interpMust(t, err)
	hypMixed, err := direct.MakeHypothesis(mixed)
	interpMust(t, err)
	hypMixedAgain, err := direct.MakeHypothesis(mixed)
	interpMust(t, err)
	if hypMixed != hypMixedAgain {
		t.Fatal("mixed hypothesis did not reuse placeholders", hypMixed, hypMixedAgain)
	}
	if op, e := manager.Operator(hypMixed); e != nil || op != OperatorAnd {
		t.Fatal("mixed hypothesis", op, e)
	}
	reflA, err := direct.MakeReflexivity(p)
	interpMust(t, err)
	if no, _ := manager.IsFalse(reflA); !no {
		t.Fatal("A reflexivity")
	}
	reflB, err := direct.MakeReflexivity(q)
	interpMust(t, err)
	if yes, _ := manager.IsTrue(reflB); !yes {
		t.Fatal("B reflexivity")
	}
	reflMixed, err := direct.MakeReflexivity(mixed)
	interpMust(t, err)
	if op, e := manager.Operator(reflMixed); e != nil || op != OperatorAnd {
		t.Fatal("mixed reflexivity", op, e)
	}
	contra, err := direct.MakeContra(hypMixed, []ast.ASTHandle{mixed})
	interpMust(t, err)
	if contra != hypMixed {
		t.Fatal("contra changed proof term", contra, hypMixed)
	}
}

func TestDirectProofArithmeticHypothesisAndGuards(t *testing.T) {
	m, manager := interpProofManagerFixture(t)
	integer, err := manager.IntType()
	interpMust(t, err)
	x, err := manager.MakeVariable("direct-x", integer)
	interpMust(t, err)
	x = interpPin(t, m, x)
	y, err := manager.MakeVariable("direct-y", integer)
	interpMust(t, err)
	y = interpPin(t, m, y)
	zero, err := manager.MakeIntText("0")
	interpMust(t, err)
	xFrame, err := manager.Make(OperatorLE, x, zero)
	interpMust(t, err)
	xFrame = interpPin(t, m, xFrame)
	yFrame, err := manager.Make(OperatorLE, y, zero)
	interpMust(t, err)
	yFrame = interpPin(t, m, yFrame)
	base, err := NewBase(manager, []ast.ASTHandle{xFrame, yFrame}, nil, nil)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, base.Close()) })
	direct, err := NewDirectProofInterpolator(base, DownwardScopeRange(0), false)
	interpMust(t, err)
	mixed, err := manager.Make(OperatorLE, x, y)
	interpMust(t, err)
	hypothesis, err := direct.MakeHypothesis(mixed)
	interpMust(t, err)
	if op, e := manager.Operator(hypothesis); e != nil || op != OperatorAnd {
		t.Fatal("mixed arithmetic hypothesis", op, e)
	}
	copy := *direct
	if err = copy.Close(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copy guard", err)
	}
	interpMust(t, direct.Close())
	if _, err = direct.TermSide(mixed); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed guard", err)
	}
}
