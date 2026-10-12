package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestDynAckSchedulePairPriorityGCAndCumulativeBudget(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkSort(symbol.Numerical(901), ast.ASTSortInfo{})
	astTestMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		astTestMust(t, m.IncRef(h))
		t.Cleanup(func() { astTestMust(t, m.DecRef(h)) })
		return h
	}
	arg := func(id uint32) ast.ASTHandle {
		t.Helper()
		h, e := m.MkConst(symbol.Numerical(id), sort)
		astTestMust(t, e)
		return pin(h)
	}
	argA, argB, argC := arg(902), arg(903), arg(904)
	decl, e := m.MkFuncDecl(symbol.Numerical(905), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	app := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		result, e := m.MkApp(decl, []ast.ASTHandle{h})
		astTestMust(t, e)
		return pin(result)
	}
	a, b, c := app(argA), app(argB), app(argC)
	base := map[ast.ASTHandle]uint32{}
	for _, h := range []ast.ASTHandle{a, b, c} {
		base[h], e = m.GetRefCount(h)
		astTestMust(t, e)
	}
	pairs, e := newDynAckPairState(m, 2, .75)
	astTestMust(t, e)
	triples, e := newDynAckTripleState(m, 2)
	astTestMust(t, e)
	t.Cleanup(func() {
		astTestMust(t, pairs.close())
		astTestMust(t, triples.close())
	})
	s, e := newDynAckSchedule(pairs, triples, 2, 1)
	astTestMust(t, e)
	for i := 0; i < 3; i++ {
		astTestMust(t, pairs.congruenceEvent(a, b))
	}
	for i := 0; i < 2; i++ {
		astTestMust(t, triples.eqEvent(a, b, c))
	}
	if got, e := s.propagate(2, dynAckDisabled); e != nil || len(got) != 0 || s.propagations != 0 {
		t.Fatal("disabled propagation must leave counters and queues intact", got, e)
	}
	got, e := s.propagate(1, dynAckRoot)
	astTestMust(t, e)
	if len(got) != 1 || !got[0].isPair || got[0].pair != (dynAckPair{a, b}) || s.instances != 1 {
		t.Fatal("pair candidate must precede triple under one-instance budget", got, s.instances)
	}
	astTestMust(t, pairs.markInstantiated(got[0].pair))
	got, e = s.propagate(1, dynAckRoot)
	astTestMust(t, e)
	if len(got) != 0 || s.propagations != 2 {
		t.Fatal("instance budget is cumulative across propagations", got, s.propagations)
	}
	got, e = s.propagate(2, dynAckRoot) // third propagation runs pair GC first
	astTestMust(t, e)
	if len(got) != 1 || got[0].isPair || got[0].triple != (dynAckTriple{a, b, c}) || s.propagations != 0 || len(pairs.apps) != 0 {
		t.Fatal("GC must clear pair roots then leave triple candidate for next budget slot", got, s.propagations, pairs.apps)
	}
	astTestMust(t, triples.markInstantiated(got[0].triple))
	astTestMust(t, s.initSearch())
	if s.instances != 0 || s.propagations != 0 || len(pairs.instantiated) != 1 || len(triples.instantiated) != 1 {
		t.Fatal("search init must reset counters while retaining clause state", s.instances, s.propagations, pairs.instantiated, triples.instantiated)
	}
	astTestMust(t, s.reset())
	if len(pairs.instantiated) != 0 || len(triples.instantiated) != 0 {
		t.Fatal("manager reset must clear both instantiated sets")
	}
	for i := 0; i < 2; i++ {
		astTestMust(t, pairs.congruenceEvent(a, b))
		astTestMust(t, triples.eqEvent(a, b, c))
	}
	got, e = s.propagate(2, dynAckRoot)
	astTestMust(t, e)
	if len(got) != 2 || !got[0].isPair || got[0].pair != (dynAckPair{a, b}) || got[1].isPair || got[1].triple != (dynAckTriple{a, b, c}) {
		t.Fatal("source drains pair queue before triple queue", got)
	}
	astTestMust(t, s.reset())
	half, e := newDynAckSchedule(pairs, triples, 100, .5)
	astTestMust(t, e)
	for i := 0; i < 2; i++ {
		astTestMust(t, pairs.congruenceEvent(a, b))
		astTestMust(t, triples.eqEvent(a, b, c))
	}
	for _, tc := range []struct {
		conflicts uint32
		wantPair  bool
		want      int
	}{{1, false, 0}, {2, true, 1}, {4, false, 1}} {
		got, e = half.propagate(tc.conflicts, dynAckRoot)
		astTestMust(t, e)
		if len(got) != tc.want || (tc.want != 0 && got[0].isPair != tc.wantPair) {
			t.Fatal("fractional source instance budget", tc.conflicts, got)
		}
	}
	astTestMust(t, half.reset())
	for h, want := range base {
		refs, e := m.GetRefCount(h)
		astTestMust(t, e)
		if refs != want {
			t.Fatal("scheduled app root leaked", h, refs, want)
		}
	}
}

func TestDynAckScheduleSourceEventGates(t *testing.T) {
	if dynAckDisabled != 0 || dynAckRoot != 1 || dynAckConflictResolution != 2 {
		t.Fatal("source DACK strategy enumerator values changed")
	}
	m := astTestManager(t)
	sort, e := m.MkSort(symbol.Numerical(911), ast.ASTSortInfo{})
	astTestMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		astTestMust(t, m.IncRef(h))
		t.Cleanup(func() { astTestMust(t, m.DecRef(h)) })
		return h
	}
	decl, e := m.MkFuncDecl(symbol.Numerical(912), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	app := func(id uint32) ast.ASTHandle {
		t.Helper()
		arg, e := m.MkConst(symbol.Numerical(id), sort)
		astTestMust(t, e)
		arg = pin(arg)
		h, e := m.MkApp(decl, []ast.ASTHandle{arg})
		astTestMust(t, e)
		return pin(h)
	}
	a, b, c := app(913), app(914), app(915)
	pairs, e := newDynAckPairState(m, 2, .75)
	astTestMust(t, e)
	triples, e := newDynAckTripleState(m, 2)
	astTestMust(t, e)
	t.Cleanup(func() {
		astTestMust(t, pairs.close())
		astTestMust(t, triples.close())
	})
	s, e := newDynAckSchedule(pairs, triples, 10, 1)
	astTestMust(t, e)
	astTestMust(t, s.usedCongruence(a, a, dynAckDisabled))
	astTestMust(t, s.usedCongruence(a, a, dynAckRoot))
	astTestMust(t, s.conflictCongruence(a, a, dynAckConflictResolution))
	astTestMust(t, s.usedEquality(a, a, a, false))
	if len(pairs.apps) != 0 || len(triples.apps) != 0 {
		t.Fatal("disabled event gates must run before app preconditions")
	}
	if e := s.usedCongruence(a, a, dynAckConflictResolution); e != ast.ErrASTDomain {
		t.Fatal("enabled congruence must enforce source distinct-app precondition", e)
	}
	astTestMust(t, s.conflictCongruence(a, b, dynAckRoot))
	astTestMust(t, s.usedCongruence(a, b, dynAckConflictResolution))
	astTestMust(t, s.usedEquality(a, b, c, true))
	if pairs.counts[dynAckPair{a, b}] != 2 || len(pairs.toInstantiate) != 1 || triples.counts[dynAckTriple{a, b, c}] != 1 {
		t.Fatal("root and conflict-resolution events share pair counts; eq uses independent triple counts", pairs.counts, triples.counts)
	}
	astTestMust(t, s.usedEquality(a, b, c, true))
	got, e := s.propagate(2, dynAckRoot)
	astTestMust(t, e)
	if len(got) != 2 || !got[0].isPair || got[1].isPair {
		t.Fatal("enabled event gates must feed pair then triple propagation", got)
	}
	if e := s.conflictCongruence(a, b, dynAckStrategy(3)); e != ast.ErrASTDomain {
		t.Fatal("invalid strategy must be refused", e)
	}
	astTestMust(t, s.reset())
}
