package ast

// Source: src/ast/array_decl_plugin.cpp:543-558;
// src/ast/ast.cpp:2566-2593.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestASTArrayOriginalRecursiveFullInterpretation(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	u, e := m.MkUninterpretedSort(astTestSymbol(t, m, "array-uninterpreted-domain"), nil)
	astTestMust(t, e)
	astTestPin(t, m, u)
	makeSort := func(domain []ASTHandle, r ASTHandle) ASTHandle {
		s, e := m.MkArraySort(SliceArguments(domain), r)
		astTestMust(t, e)
		astTestPin(t, m, s)
		return s
	}
	full := makeSort([]ASTHandle{b, b}, b)
	badDomain := makeSort([]ASTHandle{b, u}, b)
	badRange := makeSort([]ASTHandle{b}, u)
	for _, row := range []struct {
		s    ASTHandle
		want bool
	}{{b, true}, {u, false}, {full, true}, {badDomain, false}, {badRange, false}, {makeSort([]ASTHandle{full}, full), true}, {makeSort([]ASTHandle{badDomain}, b), false}, {makeSort([]ASTHandle{b}, badRange), false}} {
		got, e := m.IsFullyInterpreted(row.s)
		astTestMust(t, e)
		if got != row.want {
			t.Fatal("recursive array interpretation", row.s, got, row.want)
		}
	}
}

type arrayValueProc struct {
	sorts       []ASTHandle
	values      map[ASTHandle]ASTHandle
	failureSort ASTHandle
	failure     error
}

func (p *arrayValueProc) SomeValue(s ASTHandle, _ uint32) (ASTHandle, error) {
	p.sorts = append(p.sorts, s)
	if s == p.failureSort {
		return ASTHandle{}, p.failure
	}
	return p.values[s], nil
}

func TestASTArrayOriginalSomeValueRecursiveOverrideAndFallback(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	inner, e := m.MkArraySort(SliceArguments([]ASTHandle{b}), b)
	astTestMust(t, e)
	astTestPin(t, m, inner)
	outer, e := m.MkArraySort(SliceArguments([]ASTHandle{b, b}), inner)
	astTestMust(t, e)
	astTestPin(t, m, outer)
	expectedInner, e := m.MkConstArray(inner, no)
	astTestMust(t, e)
	astTestPin(t, m, expectedInner)
	expectedOuter, e := m.MkConstArray(outer, expectedInner)
	astTestMust(t, e)
	astTestPin(t, m, expectedOuter)
	got, e := m.GetSomeValue(outer)
	astTestMust(t, e)
	if got != expectedOuter {
		t.Fatal("recursive constant-array value", got, expectedOuter)
	}
	changedInner, e := m.MkConstArray(inner, yes)
	astTestMust(t, e)
	astTestPin(t, m, changedInner)
	changedOuter, e := m.MkConstArray(outer, changedInner)
	astTestMust(t, e)
	astTestPin(t, m, changedOuter)
	proc := arrayValueProc{values: map[ASTHandle]ASTHandle{b: yes}}
	got, e = m.GetSomeValueWithProc(outer, &proc)
	astTestMust(t, e)
	if got != changedOuter || len(proc.sorts) != 3 || proc.sorts[0] != outer || proc.sorts[1] != inner || proc.sorts[2] != b || m.someValueProc != nil {
		t.Fatal("recursive range override", got, proc.sorts)
	}
	proc = arrayValueProc{values: map[ASTHandle]ASTHandle{outer: expectedOuter}}
	got, e = m.GetSomeValueWithProc(outer, &proc)
	astTestMust(t, e)
	if got != expectedOuter || len(proc.sorts) != 1 {
		t.Fatal("outer override must erase recursion")
	}
	failure := errors.New("range override refusal")
	proc = arrayValueProc{failureSort: inner, failure: failure}
	if _, e = m.GetSomeValueWithProc(outer, &proc); !errors.Is(e, failure) || m.someValueProc != nil {
		t.Fatal("recursive callback refusal restoration", e)
	}
	u, e := m.MkUninterpretedSort(astTestSymbol(t, m, "array-model-value-range"), nil)
	astTestMust(t, e)
	astTestPin(t, m, u)
	s, e := m.MkArraySort(SliceArguments([]ASTHandle{b}), u)
	astTestMust(t, e)
	astTestPin(t, m, s)
	uv, e := m.GetSomeValue(u)
	astTestMust(t, e)
	astTestPin(t, m, uv)
	expected, e := m.MkConstArray(s, uv)
	astTestMust(t, e)
	astTestPin(t, m, expected)
	got, e = m.GetSomeValue(s)
	astTestMust(t, e)
	if got != expected {
		t.Fatal("range model-value fallback")
	}
}

func TestASTArraySomeValueRecursiveReservationRefusalRestoresProc(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	inner, e := m.MkArraySort(SliceArguments([]ASTHandle{b}), b)
	astTestMust(t, e)
	astTestPin(t, m, inner)
	outer, e := m.MkArraySort(SliceArguments([]ASTHandle{b}), inner)
	astTestMust(t, e)
	astTestPin(t, m, outer)
	want, e := m.GetSomeValue(outer)
	astTestMust(t, e)
	astTestPin(t, m, want)
	parameters := m.liveParameters
	for fault := int64(0); fault < 128; fault++ {
		proc := arrayValueProc{}
		m.Budget().FailAfter(fault)
		got, e := m.GetSomeValueWithProc(outer, &proc)
		m.Budget().FailAfter(-1)
		if m.someValueProc != nil || m.liveParameters != parameters {
			t.Fatal("recursive callback/parameter leak", fault)
		}
		if e == nil {
			if got != want {
				t.Fatal("recursive value reuse")
			}
			return
		}
		if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("recursive value refusal", fault, e)
		}
	}
	t.Fatal("recursive value refusal campaign never succeeded")
}
