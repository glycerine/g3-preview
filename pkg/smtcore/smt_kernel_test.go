package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func smtTestKernel(t *testing.T, manager *ast.ASTManager) *SMTKernel {
	t.Helper()
	kernel, err := NewSMTKernel(manager, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, kernel.Close()) })
	return kernel
}

func TestSMTKernelAssertionCheckAndDecisionClause(t *testing.T) {
	m := astTestManager(t)
	atom := smtRelevancyBool(t, m, "kernel-atom")
	kernel := smtTestKernel(t, m)
	astTestMust(t, kernel.Assert(atom, ast.ASTHandle{}))
	result, err := kernel.Check()
	astTestMust(t, err)
	if result != lbool.True || len(kernel.Assertions()) != 1 {
		t.Fatal("unit assertion check", result, kernel.Assertions())
	}

	a := smtRelevancyBool(t, m, "kernel-or-a")
	b := smtRelevancyBool(t, m, "kernel-or-b")
	disjunction, err := m.MkOr(a, b)
	astTestMust(t, err)
	other := smtTestKernel(t, m)
	astTestMust(t, other.Assert(disjunction, ast.ASTHandle{}))
	result, err = other.Check()
	astTestMust(t, err)
	if result != lbool.True || other.Statistics().NumDecisions == 0 {
		t.Fatal("decision-requiring clause", result, other.Statistics())
	}
	internalizer, err := other.Internalizer()
	astTestMust(t, err)
	for _, expression := range []ast.ASTHandle{a, b} {
		literal, literalErr := internalizer.Literal(expression)
		astTestMust(t, literalErr)
		value, assignmentErr := internalizer.Assignment(literal)
		astTestMust(t, assignmentErr)
		if value != lbool.Undef {
			t.Fatal("search decision leaked after check", expression, value)
		}
	}

	falseExpression, err := m.MkFalse()
	astTestMust(t, err)
	unsat := smtTestKernel(t, m)
	astTestMust(t, unsat.Assert(falseExpression, ast.ASTHandle{}))
	result, err = unsat.Check()
	astTestMust(t, err)
	if result != lbool.False || !unsat.Inconsistent() {
		t.Fatal("false assertion check", result, unsat.Inconsistent())
	}
}

func TestSMTKernelConflictLearningProvesBooleanUnsat(t *testing.T) {
	m := astTestManager(t)
	a := smtRelevancyBool(t, m, "kernel-unsat-a")
	b := smtRelevancyBool(t, m, "kernel-unsat-b")
	notA, err := m.MkNot(a)
	astTestMust(t, err)
	notB, err := m.MkNot(b)
	astTestMust(t, err)
	clauses := make([]ast.ASTHandle, 4)
	clauses[0], err = m.MkOr(a, b)
	astTestMust(t, err)
	clauses[1], err = m.MkOr(notA, b)
	astTestMust(t, err)
	clauses[2], err = m.MkOr(a, notB)
	astTestMust(t, err)
	clauses[3], err = m.MkOr(notA, notB)
	astTestMust(t, err)
	formula, err := m.MkAndArgs(clauses)
	astTestMust(t, err)
	kernel := smtTestKernel(t, m)
	astTestMust(t, kernel.Assert(formula, ast.ASTHandle{}))
	result, err := kernel.Check()
	astTestMust(t, err)
	if result != lbool.False || kernel.Statistics().NumDecisions == 0 || kernel.Statistics().NumConflicts < 2 {
		t.Fatal("search did not learn from contradictory branches", result, kernel.Statistics())
	}
}

func TestSMTKernelFlattensNegatedDisjunctionAssertion(t *testing.T) {
	m := astTestManager(t)
	a := smtRelevancyBool(t, m, "kernel-not-or-a")
	b := smtRelevancyBool(t, m, "kernel-not-or-b")
	disjunction, err := m.MkOr(a, b)
	astTestMust(t, err)
	formula, err := m.MkNot(disjunction)
	astTestMust(t, err)
	kernel := smtTestKernel(t, m)
	astTestMust(t, kernel.Assert(formula, ast.ASTHandle{}))
	result, err := kernel.Check()
	astTestMust(t, err)
	if result != lbool.True || kernel.Statistics().NumDecisions != 0 {
		t.Fatal("negated disjunction assertion", result, kernel.Statistics())
	}
	state, err := kernel.Internalizer()
	astTestMust(t, err)
	for _, expression := range []ast.ASTHandle{a, b} {
		literal, literalErr := state.Literal(expression)
		astTestMust(t, literalErr)
		value, assignmentErr := state.Assignment(literal)
		astTestMust(t, assignmentErr)
		if value != lbool.False {
			t.Fatal("negated disjunction child", expression, value)
		}
	}
}

func TestSMTKernelFlattenedAssertionsOwnDerivedProofs(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	a := smtRelevancyBool(t, m, "kernel-proof-a")
	b := smtRelevancyBool(t, m, "kernel-proof-b")
	formula, err := m.MkAnd(a, b)
	astTestMust(t, err)
	kernel := smtTestKernel(t, m)
	astTestMust(t, kernel.Assert(formula, ast.ASTHandle{}))
	assertions := kernel.Assertions()
	if len(assertions) != 2 {
		t.Fatal("flattened proof assertions", assertions)
	}
	for _, assertion := range assertions {
		andElim, proofErr := m.IsProofKind(assertion.Proof, ast.ASTPrAndElim)
		astTestMust(t, proofErr)
		if !andElim {
			t.Fatal("missing conjunction-elimination proof", assertion.Proof)
		}
	}
	result, err := kernel.Check()
	astTestMust(t, err)
	if result != lbool.True {
		t.Fatal("proved conjunction", result)
	}
}

func TestSMTKernelAssumptionsValidateCheckAndRollback(t *testing.T) {
	m := astTestManager(t)
	a := smtRelevancyBool(t, m, "kernel-assumption-a")
	b := smtRelevancyBool(t, m, "kernel-assumption-b")
	clause, err := m.MkOr(a, b)
	astTestMust(t, err)
	notA, err := m.MkNot(a)
	astTestMust(t, err)
	notB, err := m.MkNot(b)
	astTestMust(t, err)
	kernel := smtTestKernel(t, m)
	astTestMust(t, kernel.Assert(clause, ast.ASTHandle{}))
	result, err := kernel.CheckAssumptions([]ast.ASTHandle{notA, notB})
	astTestMust(t, err)
	if result != lbool.False {
		t.Fatal("contradictory assumptions", result)
	}
	core := kernel.UnsatCore()
	if len(core) != 2 || core[0] != notA || core[1] != notB {
		t.Fatal("assumption unsat core", core, notA, notB)
	}
	state, err := kernel.Internalizer()
	astTestMust(t, err)
	for _, expression := range []ast.ASTHandle{a, b} {
		literal, literalErr := state.Literal(expression)
		astTestMust(t, literalErr)
		value, assignmentErr := state.Assignment(literal)
		astTestMust(t, assignmentErr)
		data, dataErr := state.BooleanData(literal.Var())
		astTestMust(t, dataErr)
		if value != lbool.Undef || data.Assumption {
			t.Fatal("assumption state leaked", expression, value, data.Assumption)
		}
	}
	result, err = kernel.Check()
	astTestMust(t, err)
	if result != lbool.True {
		t.Fatal("base formula after assumption rollback", result)
	}
	if len(kernel.UnsatCore()) != 0 {
		t.Fatal("ordinary check did not clear unsat core")
	}
	if _, err = kernel.CheckAssumptions([]ast.ASTHandle{clause}); err == nil {
		t.Fatal("compound assumption accepted")
	}
}

func TestSMTKernelUnsatCoreIncludesContradictingAssumptionLiteral(t *testing.T) {
	m := astTestManager(t)
	a := smtRelevancyBool(t, m, "kernel-core-a")
	notA, err := m.MkNot(a)
	astTestMust(t, err)
	kernel := smtTestKernel(t, m)
	astTestMust(t, kernel.Assert(a, ast.ASTHandle{}))
	result, err := kernel.CheckAssumptions([]ast.ASTHandle{notA})
	astTestMust(t, err)
	core := kernel.UnsatCore()
	if result != lbool.False || len(core) != 1 || core[0] != notA {
		t.Fatal("directly contradicting assumption core", result, core)
	}
}

func TestSMTKernelScopesRestoreAssertionsAndContext(t *testing.T) {
	m := astTestManager(t)
	atom := smtRelevancyBool(t, m, "kernel-scope-atom")
	negation, err := m.MkNot(atom)
	astTestMust(t, err)
	kernel := smtTestKernel(t, m)
	astTestMust(t, kernel.Assert(atom, ast.ASTHandle{}))
	astTestMust(t, kernel.Push())
	astTestMust(t, kernel.Assert(negation, ast.ASTHandle{}))
	result, err := kernel.Check()
	astTestMust(t, err)
	if result != lbool.False || kernel.ScopeLevel() != 1 {
		t.Fatal("scoped contradiction", result, kernel.ScopeLevel())
	}
	astTestMust(t, kernel.Pop(1))
	if kernel.ScopeLevel() != 0 || len(kernel.Assertions()) != 1 || kernel.Inconsistent() {
		t.Fatal("scope rollback", kernel.ScopeLevel(), kernel.Assertions(), kernel.Inconsistent())
	}
	result, err = kernel.Check()
	astTestMust(t, err)
	if result != lbool.True {
		t.Fatal("base assertion after rollback", result)
	}
}
