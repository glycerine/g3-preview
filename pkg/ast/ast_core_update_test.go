// Licensed under the MIT license.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTQuantifierUpdateSourceOverloads(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkConst(astTestSymbol(t, m, "update-a"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	c, e := m.MkConst(astTestSymbol(t, m, "update-c"), b)
	astTestMust(t, e)
	astTestPin(t, m, c)
	pat, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{a})
	astTestMust(t, e)
	astTestPin(t, m, pat)
	qid, skid := astTestSymbol(t, m, "update-qid"), astTestSymbol(t, m, "update-skid")
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []util.Symbol{astTestSymbol(t, m, "bound")}, Body: a, Weight: 7, QID: qid, SkolemID: skid, NoPatterns: []ASTHandle{a}})
	astTestMust(t, e)
	astTestPin(t, m, q)
	// Each source overload returns the identical pointer before any allocation.
	m.BudgetField.FailAfter(0)
	r, e := m.UpdateQuantifierBody(q, a)
	astTestMust(t, e)
	if r != q {
		t.Fatal("unchanged body")
	}
	r, e = m.UpdateQuantifierWeight(q, 7)
	astTestMust(t, e)
	if r != q {
		t.Fatal("unchanged weight")
	}
	r, e = m.UpdateQuantifierKind(q, true, a)
	astTestMust(t, e)
	if r != q {
		t.Fatal("unchanged kind/body")
	}
	r, e = m.UpdateQuantifierPatterns(q, nil, a)
	astTestMust(t, e)
	if r != q {
		t.Fatal("unchanged patterns/body")
	}
	r, e = m.UpdateQuantifierAnnotations(q, nil, []ASTHandle{a}, a)
	astTestMust(t, e)
	if r != q {
		t.Fatal("unchanged annotation lists")
	}
	r, e = m.UpdateQuantifierKindPatterns(q, true, nil, a)
	astTestMust(t, e)
	if r != q {
		t.Fatal("unchanged kind/patterns/body")
	}
	m.BudgetField.FailAfter(-1)
	check := func(h ASTHandle, forall bool, body ASTHandle, weight int32, pc, npc uint32) {
		astTestPin(t, m, h)
		v := m.quantifierAt(h.slot)
		if v.forall != forall || m.external(v.body) != body || v.weight != weight || v.patternCount != pc || v.noPatternCount != npc || v.count != 1 || v.qid != qid || v.skolemID != skid {
			t.Fatal("source retained/replaced fields", *v)
		}
		s, e := m.GetDeclarationSort(h, 0)
		astTestMust(t, e)
		if s != b {
			t.Fatal("bound sort")
		}
	}
	r, e = m.UpdateQuantifierBody(q, c)
	astTestMust(t, e)
	check(r, true, c, 7, 0, 1)
	r, e = m.UpdateQuantifierWeight(q, -3)
	astTestMust(t, e)
	check(r, true, a, -3, 0, 1)
	r, e = m.UpdateQuantifierKind(q, false, c)
	astTestMust(t, e)
	check(r, false, c, 7, 0, 1)
	r, e = m.UpdateQuantifierPatterns(q, []ASTHandle{pat}, a)
	astTestMust(t, e)
	check(r, true, a, 7, 1, 0)
	patterned := r
	r, e = m.UpdateQuantifierPatterns(patterned, nil, c)
	astTestMust(t, e)
	check(r, true, c, 7, 0, 0)
	r, e = m.UpdateQuantifierAnnotations(patterned, nil, []ASTHandle{c}, a)
	astTestMust(t, e)
	check(r, true, a, 7, 0, 1)
	r, e = m.UpdateQuantifierKindPatterns(q, false, []ASTHandle{pat}, c)
	astTestMust(t, e)
	check(r, false, c, 7, 1, 0)
}

func TestASTQuantifierUpdateFailureAndEarlyIdentity(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkConst(astTestSymbol(t, m, "failure-body"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []util.Symbol{{}}, Body: a, Weight: 0})
	astTestMust(t, e)
	astTestPin(t, m, q)
	baseline := m.liveNodes
	var failures int
	succeeded := false
	for after := int64(0); after < 64; after++ {
		m.BudgetField.FailAfter(after)
		out, err := m.UpdateQuantifierWeight(q, 42)
		// Failed candidates must release physical node ownership immediately.
		if err != nil {
			failures++
			if m.liveNodes != baseline {
				t.Fatal("failed update retained candidate", after, m.liveNodes, baseline)
			}
		} else {
			if out == q {
				t.Fatal("weight is part of source identity")
			}
			succeeded = true
		}
		m.BudgetField.FailAfter(-1)
		if succeeded {
			astTestPin(t, m, out)
			break
		}
	}
	if !succeeded || failures == 0 {
		t.Fatal("missing constructor failure prefixes", failures)
	}
}
