package smt2

import (
	"errors"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMT2LocalReferenceStacksFrameShrinkAndReset(t *testing.T) {
	m := localLookupManager(t)
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	term, err := m.MkVar(0, sort)
	if err != nil {
		t.Fatal(err)
	}
	var env smt2LocalEnv
	name := usymbol.Numerical(31)
	env.insert(name, smt2Local{term: term, level: 1})
	var exprs localExprStack
	var sorts localSortStack
	baseline := m.Stats().LiveReferences
	for i := 0; i < 2; i++ {
		found, err := env.pushLocal(m, nil, name, 1, &exprs)
		if err != nil || !found {
			t.Fatal("local expression", found, err)
		}
		if err := sorts.push(m, sort); err != nil {
			t.Fatal(err)
		}
	}
	if m.Stats().LiveReferences != baseline+4 {
		t.Fatal("parser stacks did not own both expression and sort slots")
	}
	if got, err := exprs.top(m); err != nil || got != term {
		t.Fatal("expression stack top", got, err)
	}
	if got, err := sorts.top(m); err != nil || got != sort {
		t.Fatal("sort stack top", got, err)
	}
	if err := sorts.shrink(m, 1); err != nil {
		t.Fatal(err)
	}
	if err := exprs.shrink(m, 1); err != nil {
		t.Fatal(err)
	}
	if len(sorts.refs) != 1 || len(exprs.refs) != 1 || m.Stats().LiveReferences != baseline+2 {
		t.Fatal("frame shrink discarded the wrong reference slots")
	}
	if err := sorts.reset(m); err != nil {
		t.Fatal(err)
	}
	if err := exprs.reset(m); err != nil {
		t.Fatal(err)
	}
	if len(sorts.refs) != 0 || len(exprs.refs) != 0 || m.Stats().LiveReferences != baseline {
		t.Fatal("parser reset retained reference slots")
	}
	if _, err := exprs.top(m); err != errLocalExprStack {
		t.Fatal("empty expression stack", err)
	}
	if err := exprs.shrink(m, -1); err != errLocalExprStack {
		t.Fatal("invalid expression frame position", err)
	}
	if err := sorts.shrink(m, 1); err != errLocalSortStack {
		t.Fatal("invalid sort frame position", err)
	}
}

func TestSMT2QualifiedLocalExpressionOwnerAndFrameCleanup(t *testing.T) {
	m := localLookupManager(t)
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	term, err := m.MkVar(0, sort)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.IncRef(term); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.DecRef(term); err != nil {
			t.Fatal(err)
		}
	})
	var env smt2LocalEnv
	name := usymbol.Numerical(32)
	env.insert(name, smt2Local{term: term, level: 1})
	var exprs localExprStack
	var sorts localSortStack
	baseline := m.Stats().LiveReferences
	if err := sorts.push(m, sort); err != nil {
		t.Fatal(err)
	}
	found, err := env.pushQualifiedLocal(m, nil, name, 1, &sorts, true, 0, &exprs)
	if err != nil || !found {
		t.Fatal("qualified local", found, err)
	}
	if len(sorts.refs) != 0 || len(exprs.refs) != 1 || m.Stats().LiveReferences != baseline+1 {
		t.Fatal("qualified local did not transfer only its expression reference")
	}
	if got, err := exprs.top(m); err != nil || got != term {
		t.Fatal("qualified expression top", got, err)
	}
	if err := exprs.shrink(m, 0); err != nil {
		t.Fatal(err)
	}
	if m.Stats().LiveReferences != baseline {
		t.Fatal("qualified expression survived frame cleanup")
	}
	if err := sorts.push(m, sort); err != nil {
		t.Fatal(err)
	}
	found, err = env.pushQualifiedLocal(m, nil, name, 1, &sorts, true, 1, &exprs)
	if !found || !errors.Is(err, errLocalIndexed) || len(exprs.refs) != 0 || len(sorts.refs) != 0 || m.Stats().LiveReferences != baseline {
		t.Fatal("indexed local refusal retained a parser reference", found, err)
	}
}
