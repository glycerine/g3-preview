package api

// Source: src/api/api_solver.cpp public assertion, scope, check, model,
// unsat-core, proof and reason-unknown operations.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type smtAPIFixture struct {
	t *testing.T
	c *SMTContext
	m *ast.ASTManager
}

func newSMTAPIFixture(t *testing.T, logic string, configure func(*SMTContext)) *smtAPIFixture {
	t.Helper()
	c, err := NewSMTContext()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, c.Close()) })
	if configure != nil {
		configure(c)
	}
	astTestMust(t, c.SetLogic(logic))
	m, err := c.Manager()
	astTestMust(t, err)
	return &smtAPIFixture{t: t, c: c, m: m}
}

func (f *smtAPIFixture) pin(h ast.ASTHandle, err error) ast.ASTHandle {
	f.t.Helper()
	astTestMust(f.t, err)
	ref, err := f.m.NewRef(h)
	astTestMust(f.t, err)
	f.t.Cleanup(func() { astTestMust(f.t, f.m.DelRef(&ref)) })
	return h
}

func (f *smtAPIFixture) arithmeticVariable(name string, integer bool) ast.ASTHandle {
	f.t.Helper()
	family, err := f.m.ArithmeticFamilyID()
	astTestMust(f.t, err)
	kind := int32(0)
	if integer {
		kind = 1
	}
	sort := f.pin(f.m.MkFamilySort(family, kind, nil))
	symbol, err := f.m.Symbols.Intern(name)
	astTestMust(f.t, err)
	return f.pin(f.m.MkConst(symbol, sort))
}

func TestSMTContextEndToEndModelScopesAndAssumptions(t *testing.T) {
	f := newSMTAPIFixture(t, "QF_LIA", nil)
	x := f.arithmeticVariable("api-x", true)
	four := f.pin(f.m.ArithmeticIntegerNumeralText("4"))
	equalFour := f.pin(f.m.MkEq(x, four))
	astTestMust(t, f.c.Assert(equalFour))
	assertions, err := f.c.Assertions()
	astTestMust(t, err)
	if len(assertions) != 1 || assertions[0] != equalFour {
		t.Fatal("assertion order", assertions)
	}
	status, err := f.c.CheckSat()
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("base status", status)
	}
	md, err := f.c.Model()
	astTestMust(t, err)
	declaration, err := f.m.GetDecl(x)
	astTestMust(t, err)
	value, err := md.ConstantInterpretation(declaration)
	astTestMust(t, err)
	text, integer, matched, err := f.m.ArithmeticNumeralText(value)
	astTestMust(t, err)
	if !matched || !integer || text != "4" {
		t.Fatal("model value", text, integer, matched)
	}
	astTestMust(t, md.DecRef())
	stats, err := f.c.Statistics()
	astTestMust(t, err)
	if stats.Size() == 0 {
		t.Fatal("missing solver statistics")
	}

	astTestMust(t, f.c.Push())
	level, err := f.c.ScopeLevel()
	astTestMust(t, err)
	if level != 1 {
		t.Fatal("scope level", level)
	}
	five := f.pin(f.m.ArithmeticIntegerNumeralText("5"))
	equalFive := f.pin(f.m.MkEq(x, five))
	astTestMust(t, f.c.Assert(equalFive))
	status, err = f.c.CheckSat()
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("scoped contradiction", status)
	}
	astTestMust(t, f.c.Pop(1))
	level, err = f.c.ScopeLevel()
	astTestMust(t, err)
	if level != 0 {
		t.Fatal("popped scope level", level)
	}
	boolean := f.pin(f.m.MkBoolSort())
	assumptionName, err := f.m.Symbols.Intern("api-assumption")
	astTestMust(t, err)
	assumption := f.pin(f.m.MkConst(assumptionName, boolean))
	status, err = f.c.CheckSat(assumption)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("assumption recovery", status)
	}
}

func TestSMTContextUnsatCoreAndProofResults(t *testing.T) {
	t.Run("core", func(t *testing.T) {
		f := newSMTAPIFixture(t, "QF_UF", func(c *SMTContext) {
			astTestMust(t, c.SetBoolOption("produce-unsat-cores", true))
		})
		boolean := f.pin(f.m.MkBoolSort())
		symbol, err := f.m.Symbols.Intern("api-core-p")
		astTestMust(t, err)
		p := f.pin(f.m.MkConst(symbol, boolean))
		notP := f.pin(f.m.MkNot(p))
		astTestMust(t, f.c.AssertTracked(p, "positive"))
		astTestMust(t, f.c.AssertTracked(notP, "negative"))
		status, err := f.c.CheckSat()
		astTestMust(t, err)
		if status != lbool.False {
			t.Fatal(status)
		}
		core, err := f.c.UnsatCore()
		astTestMust(t, err)
		if len(core) != 2 || core[0] == core[1] {
			t.Fatal("tracked core", core)
		}
	})
	t.Run("proof", func(t *testing.T) {
		f := newSMTAPIFixture(t, "QF_UF", func(c *SMTContext) {
			astTestMust(t, c.SetBoolOption("produce-proofs", true))
		})
		boolean := f.pin(f.m.MkBoolSort())
		symbol, err := f.m.Symbols.Intern("api-proof-p")
		astTestMust(t, err)
		p := f.pin(f.m.MkConst(symbol, boolean))
		notP := f.pin(f.m.MkNot(p))
		astTestMust(t, f.c.Assert(p))
		astTestMust(t, f.c.Assert(notP))
		status, err := f.c.CheckSat()
		astTestMust(t, err)
		if status != lbool.False {
			t.Fatal(status)
		}
		proof, err := f.c.Proof()
		astTestMust(t, err)
		fact, err := f.m.GetProofFact(proof)
		astTestMust(t, err)
		if !f.m.IsFalse(fact) {
			t.Fatal("proof does not derive false")
		}
	})
}

func TestSMTContextUnknownReason(t *testing.T) {
	f := newSMTAPIFixture(t, "ALL", func(c *SMTContext) {
		astTestMust(t, c.SetBoolOption("auto_config", false))
		astTestMust(t, c.SetUIntOption("smt.arith.solver", 0))
	})
	x := f.arithmeticVariable("api-unknown-x", false)
	zero := f.pin(f.m.ArithmeticRealNumeralText("0"))
	lower := f.pin(f.m.MkArithmeticGE(x, zero))
	astTestMust(t, f.c.Assert(lower))
	astTestMust(t, f.c.Push())
	status, err := f.c.CheckSat()
	astTestMust(t, err)
	if status != lbool.Undef {
		t.Fatal("disabled arithmetic status", status)
	}
	reason, err := f.c.ReasonUnknown()
	astTestMust(t, err)
	if reason != "smt tactic failed to show goal to be sat/unsat (incomplete (theory no arithmetic))" {
		t.Fatal("unknown reason", reason)
	}
}

func TestSMTContextCopiesShareClose(t *testing.T) {
	c, err := NewSMTContext()
	astTestMust(t, err)
	copy := *c
	astTestMust(t, copy.Close())
	if _, err = c.CheckSat(); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("shared close did not retire aliases", err)
	}
}
