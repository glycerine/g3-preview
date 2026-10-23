package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func smtAssertedNNFFixture(t *testing.T, proofs bool) (*ast.ASTManager, *SMTSolver) {
	t.Helper()
	m, config := smtModelGeneratorFixture(t)
	if proofs {
		astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	}
	solver, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	return m, solver
}

func TestSMTAssertedNNFSkolemizesExistentialAndNegativeUniversal(t *testing.T) {
	for _, negative := range []bool{false, true} {
		m, solver := smtAssertedNNFFixture(t, false)
		b, err := m.MkBoolSort()
		astTestMust(t, err)
		x, err := m.MkVar(0, b)
		astTestMust(t, err)
		p := smtQuickPredicate(t, m, "nnf-witness-p", b, b)
		body, err := m.MkApp(p, []ast.ASTHandle{x})
		astTestMust(t, err)
		q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: negative, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{astTestSymbol(t, m, "nnf-witness-x")}, Body: body})
		astTestMust(t, err)
		if negative {
			q, err = m.MkNot(q)
			astTestMust(t, err)
		}
		astTestMust(t, solver.Assert(q))
		status, err := solver.CheckSat(nil)
		astTestMust(t, err)
		if status != lbool.True {
			t.Fatal("Skolem witness formula was not satisfiable", negative, status)
		}
		groundPredicate := false
		for _, node := range solver.kernel.internalizer.enodeStack {
			d, err := m.GetDecl(node.owner)
			astTestMust(t, err)
			if d == p {
				groundPredicate = true
			}
		}
		if !groundPredicate || !solver.kernel.internalizer.quantifiers.Empty() {
			t.Fatal("existential polarity did not produce a ground witness", negative)
		}
	}
}

func TestSMTAssertedNNFExistentialConflictPreservesProof(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		m, solver := smtAssertedNNFFixture(t, proofs)
		b, err := m.MkBoolSort()
		astTestMust(t, err)
		x, err := m.MkVar(0, b)
		astTestMust(t, err)
		p := smtQuickPredicate(t, m, "nnf-conflict-p", b, b)
		body, err := m.MkApp(p, []ast.ASTHandle{x})
		astTestMust(t, err)
		exists, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: false, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{astTestSymbol(t, m, "nnf-exists-x")}, Body: body})
		astTestMust(t, err)
		astTestMust(t, solver.Assert(exists))
		negated, err := m.MkNot(body)
		astTestMust(t, err)
		forall, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{astTestSymbol(t, m, "nnf-forall-x")}, Body: negated})
		astTestMust(t, err)
		astTestMust(t, solver.Assert(forall))
		status, err := solver.CheckSat(nil)
		astTestMust(t, err)
		if status != lbool.False {
			t.Fatal("existential witness did not conflict with the universal", proofs, status)
		}
		if proofs {
			proof, err := solver.Proof()
			astTestMust(t, err)
			fact, err := m.GetProofFact(proof)
			astTestMust(t, err)
			if !m.IsFalse(fact) {
				t.Fatal("Skolemized contradiction lost its proof")
			}
		}
	}
}

func TestSMTAssertedNNFDependentSkolemFunction(t *testing.T) {
	m, solver := smtAssertedNNFFixture(t, false)
	one, err := m.ArithmeticIntegerNumeralText("1")
	astTestMust(t, err)
	integer, err := m.GetSort(one)
	astTestMust(t, err)
	x, err := m.MkVar(1, integer)
	astTestMust(t, err)
	y, err := m.MkVar(0, integer)
	astTestMust(t, err)
	next, err := m.MkArithmeticAdd(x, one)
	astTestMust(t, err)
	body, err := m.MkEq(y, next)
	astTestMust(t, err)
	exists, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: false, Sorts: []ast.ASTHandle{integer}, Names: []symbol.Symbol{astTestSymbol(t, m, "nnf-dependent-y")}, Body: body})
	astTestMust(t, err)
	forall, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{integer}, Names: []symbol.Symbol{astTestSymbol(t, m, "nnf-dependent-x")}, Body: exists})
	astTestMust(t, err)
	astTestMust(t, solver.Assert(forall))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("dependent Skolem function was not modeled", status)
	}
	for _, q := range solver.kernel.internalizer.quantifiers.quantifiers {
		body, err := m.GetQuantifierBody(q)
		astTestMust(t, err)
		contains, err := m.HasQuantifiers(body)
		astTestMust(t, err)
		if contains {
			t.Fatal("nested existential reached the model finder")
		}
	}
}

func TestSMTAssertedNNFDefinitionNamesFollowUserScopes(t *testing.T) {
	t.Run("ordinary", func(t *testing.T) { smtAssertedNNFNamesScopeCase(t, false) })
	t.Run("proofs", func(t *testing.T) { smtAssertedNNFNamesScopeCase(t, true) })
}

func smtAssertedNNFNamesScopeCase(t *testing.T, proofs bool) {
	m, solver := smtAssertedNNFFixture(t, proofs)
	// Exercise NNF naming before value propagation can replace q by true.
	// This source preprocessor field has no public parameter accessor.
	solver.kernel.preprocessState.PropagateValues = false
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	x, err := m.MkVar(0, b)
	astTestMust(t, err)
	p := smtQuickPredicate(t, m, "nnf-name-p", b, b)
	f := smtQuickPredicate(t, m, "nnf-name-f", b, b)
	body, err := m.MkApp(p, []ast.ASTHandle{x})
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{astTestSymbol(t, m, "nnf-name-x")}, Body: body})
	astTestMust(t, err)
	application, err := m.MkApp(f, []ast.ASTHandle{q})
	astTestMust(t, err)
	yes, err := m.MkTrue()
	astTestMust(t, err)
	positive, err := m.MkApp(f, []ast.ASTHandle{yes})
	astTestMust(t, err)
	negative, err := m.MkNot(positive)
	astTestMust(t, err)
	for _, h := range []ast.ASTHandle{q, application, negative} {
		ref, err := m.NewRef(h)
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
	}
	for attempt := 0; attempt < 2; attempt++ {
		astTestMust(t, solver.Push())
		for _, h := range []ast.ASTHandle{application, q, negative} {
			astTestMust(t, solver.Assert(h))
		}
		status, err := solver.CheckSat(nil)
		astTestMust(t, err)
		count, err := solver.kernel.definedNames.NumNames()
		astTestMust(t, err)
		if status != lbool.False || count == 0 {
			t.Fatal("NNF name definition did not preserve the quantified argument", attempt, status, count)
		}
		if proofs {
			proof, err := solver.Proof()
			astTestMust(t, err)
			fact, err := m.GetProofFact(proof)
			astTestMust(t, err)
			if !m.IsFalse(fact) {
				t.Fatal("NNF name definition lost its contradiction proof")
			}
		}
		astTestMust(t, solver.Pop(1))
		count, err = solver.kernel.definedNames.NumNames()
		astTestMust(t, err)
		if count != 0 {
			t.Fatal("popped NNF names survived their defining assertions", count)
		}
	}
}

func TestSMTAssertedNNFUsesGlobalModuleOptions(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	astTestMust(t, config.Globals.SourceSetName("nnf.mode", "full"))
	solver, err := NewSMTSolver(m, nil, "QF_UF", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	smtDisableAutoConfig(t, solver)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	a := smtRelevancyBool(t, m, "nnf-global-a")
	c := smtRelevancyBool(t, m, "nnf-global-b")
	conjunction, err := m.MkAnd(a, c)
	astTestMust(t, err)
	f := smtQuickPredicate(t, m, "nnf-global-f", b, b)
	application, err := m.MkApp(f, []ast.ASTHandle{conjunction})
	astTestMust(t, err)
	astTestMust(t, solver.Assert(application))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	count, err := solver.kernel.definedNames.NumNames()
	astTestMust(t, err)
	if status != lbool.True || count == 0 {
		t.Fatal("global full NNF mode did not name the ground Boolean argument", status, count)
	}
}
