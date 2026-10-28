package ast

// Source: src/ast/expr2var.cpp:25-92; src/ast/expr2var.h:32-77.

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func expr2VarAtom(t *testing.T, m *ASTManager, name string, sort ASTHandle) ASTHandle {
	t.Helper()
	h, err := m.MkConst(astTestSymbol(t, m, name), sort)
	astTestMust(t, err)
	astTestPin(t, m, h)
	return h
}

func TestExpr2VarSourceMappingOwnershipInverseAndReset(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	astTestPin(t, m, sort)
	a := expr2VarAtom(t, m, "expr2var-a", sort)
	b := expr2VarAtom(t, m, "expr2var-b", sort)
	sentinel := expr2VarAtom(t, m, "expr2var-sentinel", sort)
	refs := func(h ASTHandle) uint32 {
		t.Helper()
		n, e := m.GetRefCount(h)
		astTestMust(t, e)
		return n
	}
	aRefs, bRefs := refs(a), refs(b)
	x, err := NewExpr2Var(m)
	astTestMust(t, err)
	astTestMust(t, x.Insert(a, 2))
	astTestMust(t, x.Insert(b, 0))
	if refs(a) != aRefs+1 || refs(b) != bRefs+1 {
		t.Fatal("insert must pin each expression key")
	}
	if got, e := x.ToVar(a); e != nil || got != 2 {
		t.Fatal("source lookup", got, e)
	}
	if got, e := x.ToVar(sort); e != nil || got != math.MaxUint32 {
		t.Fatal("source absent sentinel", got, e)
	}
	if yes, e := x.IsVar(b); e != nil || !yes {
		t.Fatal("source contains", yes, e)
	}
	if manager, e := x.Manager(); e != nil || manager != m {
		t.Fatal("source manager accessor", manager, e)
	}
	entries, e := x.Entries()
	astTestMust(t, e)
	seen := make(map[ASTHandle]uint32, len(entries))
	for _, entry := range entries {
		seen[entry.Expr] = entry.Variable
	}
	if len(seen) != 2 || seen[a] != 2 || seen[b] != 0 {
		t.Fatal("source mapping iterator snapshot", seen)
	}
	if yes, e := x.InterpretedVars(); e != nil || yes {
		t.Fatal("uninterpreted constants changed interpreted flag", yes, e)
	}
	if e := x.Insert(a, 9); !errors.Is(e, ErrASTDomain) || refs(a) != aRefs+1 {
		t.Fatal("duplicate insertion guard", e, refs(a))
	}

	inv, err := x.Inverse()
	astTestMust(t, err)
	if len(inv) != 3 {
		t.Fatal("source sparse inverse size", len(inv))
	}
	for index, want := range map[int]ASTHandle{0: b, 2: a} {
		got, e := m.RefGet(inv[index])
		astTestMust(t, e)
		if got != want {
			t.Fatal("source inverse value", index, got, want)
		}
	}
	if inv[1] != (ASTRef{}) {
		t.Fatal("source inverse hole", inv[1])
	}
	for i := range inv {
		if inv[i] != (ASTRef{}) {
			astTestMust(t, m.DelRef(&inv[i]))
		}
	}
	// Source is_var tests map membership rather than the UINT_MAX lookup
	// sentinel, so a UINT_MAX-valued entry remains present.
	astTestMust(t, x.Insert(sentinel, math.MaxUint32))
	if yes, e := x.IsVar(sentinel); e != nil || !yes {
		t.Fatal("source membership at sentinel value", yes, e)
	}

	astTestMust(t, x.Reset())
	if refs(a) != aRefs || refs(b) != bRefs {
		t.Fatal("reset must release expression keys")
	}
	if recent, e := x.Recent(); e != nil || len(recent) != 0 {
		t.Fatal("reset recent expressions", recent, e)
	}
	if yes, e := x.InterpretedVars(); e != nil || yes {
		t.Fatal("reset interpreted flag", yes, e)
	}
	astTestMust(t, x.Close())
	astTestMust(t, x.Close())
	if _, e := x.ToVar(a); !errors.Is(e, ErrASTHandle) {
		t.Fatal("closed owner guard", e)
	}
}

func TestExpr2VarSourceRecentScopesAndCopiedOwnerGuards(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	astTestPin(t, m, sort)
	a := expr2VarAtom(t, m, "expr2var-scope-a", sort)
	b := expr2VarAtom(t, m, "expr2var-scope-b", sort)
	c := expr2VarAtom(t, m, "expr2var-scope-c", sort)
	x, err := NewExpr2Var(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, x.Close()) }()
	astTestMust(t, x.Insert(a, 0))
	astTestMust(t, x.Push())
	astTestMust(t, x.Insert(b, 1))
	astTestMust(t, x.Push())
	astTestMust(t, x.Insert(c, 2))
	if recent, e := x.Recent(); e != nil || len(recent) != 3 || recent[0] != a || recent[1] != b || recent[2] != c {
		t.Fatal("source recent insertion order", recent, e)
	}
	if e := x.ResetRecent(); !errors.Is(e, ErrASTDomain) {
		t.Fatal("source active-scope assertion", e)
	}
	astTestMust(t, x.Pop(1))
	if yes, e := x.IsVar(c); e != nil || yes {
		t.Fatal("inner scope key survived", yes, e)
	}
	if yes, e := x.IsVar(b); e != nil || !yes {
		t.Fatal("outer scope key removed", yes, e)
	}
	astTestMust(t, x.Pop(1))
	if yes, e := x.IsVar(b); e != nil || yes {
		t.Fatal("outer scope key survived", yes, e)
	}
	if yes, e := x.IsVar(a); e != nil || !yes {
		t.Fatal("pre-scope key removed", yes, e)
	}
	if e := x.Pop(1); !errors.Is(e, ErrASTDomain) {
		t.Fatal("scope underflow guard", e)
	}
	astTestMust(t, x.ResetRecent())
	if recent, e := x.Recent(); e != nil || len(recent) != 0 {
		t.Fatal("source reset_recent", recent, e)
	}
	if yes, e := x.IsVar(a); e != nil || !yes {
		t.Fatal("reset_recent changed mapping", yes, e)
	}
	copyOwner := *x
	if e := copyOwner.Push(); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied owner guard", e)
	}
	if e := x.Display(nil); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("nil display writer guard", e)
	}
}

func TestExpr2VarSourceInterpretedTraceAndDisplay(t *testing.T) {
	c, globals, _, budget := dynAckSourceFixture(t)
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, err)
	t.Cleanup(func() { _, e := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, e) })
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	atom := expr2VarAtom(t, m, "expr2var-trace-a", sort)
	interpreted, err := m.MkNot(atom)
	astTestMust(t, err)
	astTestPin(t, m, interpreted)
	x, err := NewExpr2Var(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, x.Close()) })
	astTestMust(t, x.Insert(atom, 4))
	if trace.Len() != 0 {
		t.Fatal("uninterpreted constant traced", trace.String())
	}
	tags.Enable("expr2var")
	astTestMust(t, x.Insert(interpreted, 7))
	if yes, e := x.InterpretedVars(); e != nil || !yes {
		t.Fatal("interpreted flag", yes, e)
	}
	if tags.ShouldTrace("expr2var") {
		want := "-------- [expr2var] insert src/ast/expr2var.cpp:29 ---------\ninterpreted:\n(not expr2var-trace-a)\n------------------------------------------------\n"
		if got := trace.String(); got != want {
			t.Fatalf("source interpreted trace: got %q, want %q", got, want)
		}
	} else if trace.Len() != 0 {
		t.Fatal("erased source trace wrote", trace.String())
	}
	var display bytes.Buffer
	astTestMust(t, x.Display(&display))
	got := display.String()
	if !strings.Contains(got, "expr2var-trace-a -> 4\n") || !strings.Contains(got, "(not expr2var-trace-a) -> 7\n") || strings.Count(got, " -> ") != 2 {
		t.Fatalf("source display entries: %q", got)
	}
}
