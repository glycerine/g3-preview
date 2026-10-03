// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"testing"
)

// tst_expr_substitution registers a constructor/destructor, with no active
// assertions; its BV rewriter and output sequence remains an unmet consumer.
func TestASTExprSubstitutionRegisteredConstructorAndPolicies(t *testing.T) {
	m := astTestManager(t)
	base := m.allocator.Stats().Live
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	owner, e := m.ExprSubstitutionManager(s)
	astTestMust(t, e)
	if owner != m {
		t.Fatal(owner)
	}
	yes, e := m.ExprSubstitutionProofsEnabled(s)
	astTestMust(t, e)
	if yes {
		t.Fatal("default disabled manager")
	}
	yes, e = m.ExprSubstitutionUnsatCoreEnabled(s)
	astTestMust(t, e)
	if yes {
		t.Fatal("default cores")
	}
	yes, e = m.ExprSubstitutionEmpty(s)
	astTestMust(t, e)
	if !yes {
		t.Fatal("initial map")
	}
	astTestMust(t, m.DelExprSubstitution(&s))
	if m.allocator.Stats().Live != base {
		t.Fatal("owner before Close")
	}
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	s, e = m.NewExprSubstitutionCores(true)
	astTestMust(t, e)
	astTestMust(t, m.ToggleProofMode(ASTProofDisabled))
	yes, e = m.ExprSubstitutionProofsEnabled(s)
	astTestMust(t, e)
	if !yes {
		t.Fatal("policy snapshot")
	}
	yes, e = m.ExprSubstitutionUnsatCoreEnabled(s)
	astTestMust(t, e)
	if !yes {
		t.Fatal("cores snapshot")
	}
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTExprSubstitutionActualMapOutputsAndReferenceOrder(t *testing.T) {
	m := astTestManager(t)
	key := astDependencyExpr(t, m, 10000)
	a := astDependencyExpr(t, m, 10001)
	b := astDependencyExpr(t, m, 10002)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	astTestMust(t, m.ExprSubstitutionInsert(s, key, a, ASTHandle{}, ASTDependency{}))
	def, pr := b, b
	found, e := m.ExprSubstitutionFindProof(s, key, &def, &pr)
	astTestMust(t, e)
	if !found || def != a || pr != b {
		t.Fatal(found, def, pr)
	}
	astTestMust(t, m.ExprSubstitutionInsert(s, key, b, ASTHandle{}, ASTDependency{}))
	found, e = m.ExprSubstitutionFindProof(s, key, &def, &pr)
	astTestMust(t, e)
	if !found || def != b {
		t.Fatal(def)
	}
	astTestMust(t, m.ExprSubstitutionErase(s, key))
	def = a
	found, e = m.ExprSubstitutionFindProof(s, key, &def, &pr)
	astTestMust(t, e)
	if found || def != a || pr != b {
		t.Fatal("miss outputs must stay untouched", found, def, pr)
	}
	astTestMust(t, m.DelExprSubstitution(&s))
}

func TestASTExprSubstitutionBoundedIndependentReferenceModel(t *testing.T) {
	m := astTestManager(t)
	var keys, values [4]ASTHandle
	for i := range keys {
		keys[i] = astDependencyExpr(t, m, uint32(11000+i))
		values[i] = astDependencyExpr(t, m, uint32(11100+i))
	}
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	model := [4]int{-1, -1, -1, -1}
	seed := uint32(19)
	for step := 0; step < 500; step++ {
		seed = seed*1664525 + 1013904223
		k := int(seed>>8) % 4
		v := int(seed>>16) % 4
		switch seed % 7 {
		case 0:
			astTestMust(t, m.ExprSubstitutionReset(s))
			model = [4]int{-1, -1, -1, -1}
		case 1:
			astTestMust(t, m.ExprSubstitutionCleanup(s))
			model = [4]int{-1, -1, -1, -1}
		case 2:
			astTestMust(t, m.ExprSubstitutionErase(s, keys[k]))
			model[k] = -1
		default:
			astTestMust(t, m.ExprSubstitutionInsert(s, keys[k], values[v], ASTHandle{}, ASTDependency{}))
			model[k] = v
		}
		counts := [4]uint32{1, 1, 1, 1}
		for i, want := range model {
			var def, pr ASTHandle
			found, e := m.ExprSubstitutionFindProof(s, keys[i], &def, &pr)
			astTestMust(t, e)
			if found != (want >= 0) || found && def != values[want] {
				t.Fatal(step, i, want, found, def)
			}
			rc, e := m.GetRefCount(keys[i])
			astTestMust(t, e)
			expected := uint32(1)
			if want >= 0 {
				expected++
				counts[want]++
			}
			if rc != expected {
				t.Fatal("key model", step, i, rc, expected)
			}
		}
		for i, want := range counts {
			rc, e := m.GetRefCount(values[i])
			astTestMust(t, e)
			if rc != want {
				t.Fatal("value multiplicity", step, i, rc, want)
			}
		}
	}
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTExprSubstitutionAllPolicyMapsAndAliasedFindOutputs(t *testing.T) {
	for _, cores := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			m := astTestManager(t)
			astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
			key := astDependencyExpr(t, m, 11200)
			value := astDependencyExpr(t, m, 11201)
			other := astDependencyExpr(t, m, 11202)
			p, e := m.MkProofAsserted(value)
			astTestMust(t, e)
			astTestPin(t, m, p)
			dep, e := m.DependencyLeaf(other)
			astTestMust(t, e)
			astTestMust(t, m.DependencyInc(dep))
			s, e := m.NewExprSubstitutionPolicy(cores, proofs)
			astTestMust(t, e)
			astTestMust(t, m.ExprSubstitutionInsert(s, key, value, p, dep))
			alias := other
			gotdep := ASTDependency{}
			yes, e := m.ExprSubstitutionFindDependency(s, key, &alias, &alias, &gotdep)
			astTestMust(t, e)
			want := value
			if proofs {
				want = p
			}
			if !yes || alias != want || cores && gotdep != dep || !cores && !gotdep.IsNull() {
				t.Fatal(cores, proofs, yes, alias, gotdep)
			}
			astTestMust(t, m.ExprSubstitutionInsert(s, key, value, p, dep))
			rc, e := m.DependencyRefCount(dep)
			astTestMust(t, e)
			wantRC := uint32(1)
			if cores {
				wantRC = 2
			}
			if rc != wantRC {
				t.Fatal("self replacement dependency", rc, wantRC)
			}
			astTestMust(t, m.ExprSubstitutionErase(s, key))
			astTestMust(t, m.DelExprSubstitution(&s))
			astTestMust(t, m.DependencyDec(dep))
		}
	}
}
func TestASTExprSubstitutionNullDefinitionRepeatedNewBranch(t *testing.T) {
	m := astTestManager(t)
	key := astDependencyExpr(t, m, 11300)
	value := astDependencyExpr(t, m, 11301)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	astTestMust(t, m.ExprSubstitutionInsert(s, key, ASTHandle{}, ASTHandle{}, ASTDependency{}))
	found, e := m.ExprSubstitutionContains(s, key)
	astTestMust(t, e)
	if !found {
		t.Fatal("NULL value entry remains present")
	}
	astTestMust(t, m.ExprSubstitutionInsert(s, key, value, ASTHandle{}, ASTDependency{}))
	rc, e := m.GetRefCount(key)
	astTestMust(t, e)
	if rc != 3 {
		t.Fatal("source repeated key INC", rc)
	}
	astTestMust(t, m.DelExprSubstitution(&s))
	rc, e = m.GetRefCount(key)
	astTestMust(t, e)
	if rc != 2 {
		t.Fatal("source reset DEC only once", rc)
	}
	astTestMust(t, m.DecRef(key)) // Recover the explicitly unlisted source INC.
}
func TestASTExprSubstitutionNullOptionalOverwriteSourceProfiles(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	key := astDependencyExpr(t, m, 11600)
	value := astDependencyExpr(t, m, 11601)
	p1, e := m.MkProofAsserted(value)
	astTestMust(t, e)
	astTestPin(t, m, p1)
	p2, e := m.MkProofAsserted(astDependencyExpr(t, m, 11602))
	astTestMust(t, e)
	astTestPin(t, m, p2)
	d1, e := m.DependencyLeaf(value)
	astTestMust(t, e)
	astTestMust(t, m.DependencyInc(d1))
	d2, e := m.DependencyLeaf(value)
	astTestMust(t, e)
	astTestMust(t, m.DependencyInc(d2))
	s, e := m.NewExprSubstitutionPolicy(true, true)
	astTestMust(t, e)
	astTestMust(t, m.ExprSubstitutionInsert(s, key, ASTHandle{}, p1, d1))
	e = m.ExprSubstitutionInsert(s, key, value, p2, d2)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
	} else {
		astTestMust(t, e)
	}
	astTestMust(t, m.DelExprSubstitution(&s))
	rc, e := m.GetRefCount(key)
	astTestMust(t, e)
	if rc != 2 {
		t.Fatal("extra key INC remains unlisted", rc)
	}
	astTestMust(t, m.DecRef(key))
	rc, e = m.GetRefCount(p1)
	astTestMust(t, e)
	want := uint32(1)
	if !astSourceDebug {
		want = 2
	}
	if rc != want {
		t.Fatal("source optional overwrite did not DEC old proof", rc, want)
	}
	if !astSourceDebug {
		astTestMust(t, m.DecRef(p1))
		astTestMust(t, m.DependencyDec(d1))
	}
	astTestMust(t, m.DependencyDec(d1))
	astTestMust(t, m.DependencyDec(d2))
}
func TestASTExprSubstitutionWarmNoAllocationsUnderRefusal(t *testing.T) {
	m := astTestManager(t)
	key := astDependencyExpr(t, m, 11700)
	value := astDependencyExpr(t, m, 11701)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ScopedExprSubstitutionPop(scope, 1))
	m.budget.FailAfter(0)
	n := testing.AllocsPerRun(100, func() {
		if e := m.ScopedExprSubstitutionPush(scope); e != nil {
			panic(e)
		}
		if e := m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, ASTDependency{}); e != nil {
			panic(e)
		}
		var a, p ASTHandle
		if found, e := m.ScopedExprSubstitutionFindProof(scope, key, &a, &p); e != nil || !found {
			panic("find")
		}
		if e := m.ScopedExprSubstitutionPop(scope, 1); e != nil {
			panic(e)
		}
	})
	m.budget.FailAfter(-1)
	if n != 0 {
		t.Fatal(n)
	}
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
}
