package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTConflictResolverBuildsFirstUIPLemmaAndBackjumpLevel(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	a := smtRelevancyBool(t, m, "resolve-a")
	b := smtRelevancyBool(t, m, "resolve-b")
	c := smtRelevancyBool(t, m, "resolve-c")
	for _, expression := range []ast.ASTHandle{a, b, c} {
		astTestMust(t, state.Internalize(expression, true))
	}
	av, _ := state.BoolVar(a)
	bv, _ := state.BoolVar(b)
	cv, _ := state.BoolVar(c)
	la, lb, lc := NewSMTLiteral(av, false), NewSMTLiteral(bv, false), NewSMTLiteral(cv, false)
	astTestMust(t, state.PushScope())
	astTestMust(t, state.AssignLiteral(la, NewSMTBooleanAxiom()))
	astTestMust(t, state.PushScope())
	astTestMust(t, state.AssignLiteral(lb, NewSMTBooleanAxiom()))
	reason, err := NewSMTClause(m, []SMTLiteral{lc, la.Negated()}, SMTClauseAux, nil, nil, false, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, reason.Close()) })
	astTestMust(t, state.AssignLiteral(lc, NewSMTBooleanClause(reason)))
	conflict, err := NewSMTClause(m, []SMTLiteral{lc.Negated(), lb.Negated()}, SMTClauseAux, nil, nil, false, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, conflict.Close()) })
	resolver, err := NewSMTConflictResolver(state)
	astTestMust(t, err)
	got, err := resolver.Resolve(NewSMTBooleanClause(conflict), SMTNullLiteral, 0)
	astTestMust(t, err)
	want := []SMTLiteral{lb.Negated(), la.Negated()}
	if !got.Resolved || got.ConflictLevel != 2 || got.NewScopeLevel != 1 || len(got.Lemma) != len(want) {
		t.Fatal("resolution metadata", got)
	}
	for i := range want {
		if got.Lemma[i] != want[i] {
			t.Fatal("lemma", got.Lemma, want)
		}
	}
}

func TestSMTInternalizerInstallsLearnedClauseInCurrentScope(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	variables := make([]SMTLiteral, 3)
	for index := range variables {
		expression := smtRelevancyBool(t, m, "learned-clause-"+string(rune('a'+index)))
		astTestMust(t, state.Internalize(expression, true))
		variable, found := state.BoolVar(expression)
		if !found {
			t.Fatal("missing learned-clause variable", index)
		}
		variables[index] = NewSMTLiteral(variable, false)
	}
	astTestMust(t, state.PushScope())
	astTestMust(t, state.AddLearnedClause(variables))
	clauses := state.Clauses()
	if len(clauses) != 1 || state.clauses[0].clause.Kind() != SMTClauseLearned {
		t.Fatal("learned clause kind", clauses)
	}
	astTestMust(t, state.PopScopes(1))
	if len(state.Clauses()) != 0 {
		t.Fatal("learned clause survived its logical scope")
	}
}
