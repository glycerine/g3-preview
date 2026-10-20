package ast

import (
	"errors"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTCheckPredicateMemoizedDAGAndReset(t *testing.T) {
	m := astTestManager(t)
	a, b := proofTestBool(t, m, "functor-a"), proofTestBool(t, m, "functor-b")
	shared := proofTestApp(t, m, ASTOpAnd, a, b)
	root := proofTestApp(t, m, ASTOpOr, shared, shared)
	astTestPin(t, m, root)
	calls := make(map[ASTHandle]int)
	check, err := NewCheckPredicate(m, func(h ASTHandle) (bool, error) {
		calls[h]++
		return h == b, nil
	})
	astTestMust(t, err)
	t.Cleanup(func() {
		if !check.closed {
			astTestMust(t, check.Close())
		}
	})
	yes, err := check.Check(root)
	astTestMust(t, err)
	if !yes {
		t.Fatal("nested predicate was not propagated")
	}
	before := calls[root]
	yes, err = check.Check(root)
	astTestMust(t, err)
	if !yes || calls[root] != before {
		t.Fatal("memoized root was revisited")
	}
	astTestMust(t, check.Reset())
	_, err = check.Check(root)
	astTestMust(t, err)
	if calls[root] == before {
		t.Fatal("reset retained memoized visit")
	}
}

func TestASTContainsAppSlicesOwnershipAndGuards(t *testing.T) {
	m := astTestManager(t)
	a, b := proofTestBool(t, m, "contains-a"), proofTestBool(t, m, "contains-b")
	root := proofTestApp(t, m, ASTOpAnd, a, b)
	contains, err := NewContainsApp(m, b)
	astTestMust(t, err)
	yes, err := contains.Check(root)
	astTestMust(t, err)
	if !yes {
		t.Fatal("missing target")
	}
	yes, err = contains.CheckSlice([]ASTHandle{a, root})
	astTestMust(t, err)
	if !yes {
		t.Fatal("slice missing target")
	}
	if got, e := contains.Target(); e != nil || got != b {
		t.Fatalf("owned target: %v %v", got, e)
	}
	astTestMust(t, contains.Close())
	if _, err = contains.Check(root); !errors.Is(err, ErrASTHandle) {
		t.Fatalf("closed guard: %v", err)
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	variable, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	if _, err = NewContainsApp(m, variable); !errors.Is(err, ErrASTDomain) {
		t.Fatalf("non-application target guard: %v", err)
	}
}

func TestASTMapProcReconstructQuantifierAndReset(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	v, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	a := proofTestBool(t, m, "map-a")
	body := proofTestApp(t, m, ASTOpAnd, v, a)
	q, err := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "x")}, Body: body})
	astTestMust(t, err)
	astTestPin(t, m, q)
	p, err := NewASTMapProc(m)
	astTestMust(t, err)
	t.Cleanup(func() {
		if !p.closed {
			astTestMust(t, p.Close())
		}
	})
	astTestMust(t, p.VisitVar(v))
	astTestMust(t, p.Insert(a, v))
	astTestMust(t, p.Reconstruct(body))
	rewrittenBody, found, err := p.GetExpr(body)
	astTestMust(t, err)
	if !found || rewrittenBody == body {
		t.Fatal("application was not reconstructed")
	}
	astTestMust(t, p.VisitQuantifier(q))
	rewrittenQ, found, err := p.GetExpr(q)
	astTestMust(t, err)
	if !found || rewrittenQ == q {
		t.Fatal("quantifier was not reconstructed")
	}
	gotBody, err := m.GetQuantifierBody(rewrittenQ)
	astTestMust(t, err)
	if gotBody != rewrittenBody {
		t.Fatal("rewritten quantifier body")
	}
	astTestMust(t, p.Reset())
	if _, found, err = p.GetExpr(body); err != nil || found {
		t.Fatalf("reset mapping: found=%v err=%v", found, err)
	}
}
