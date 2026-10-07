// Source: src/ast/ast.cpp:2561-2582.
package ast

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

type testSomeValueProc struct {
	value   ASTHandle
	failure error
	calls   uint32
	sort    ASTHandle
	hash    uint32
	manager *ASTManager
	nested  SomeValueProc
}

func (p *testSomeValueProc) SomeValue(s ASTHandle, hash uint32) (ASTHandle, error) {
	p.calls++
	p.sort = s
	p.hash = hash
	if p.nested != nil {
		r, e := p.manager.internal(s)
		if e != nil {
			return ASTHandle{}, e
		}
		if _, e = p.manager.getSomeValueWithProc(r, p.nested); !errors.Is(e, p.failure) {
			return ASTHandle{}, errors.New("nested error lost")
		}
		if p.manager.someValueProc != p {
			return ASTHandle{}, errors.New("outer callback not restored")
		}
	}
	return p.value, p.failure
}
func TestASTSomeValueProcScopedPriorityAndFailureRestoration(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	astTestPin(t, m, yes)
	no, e := m.MkFalse()
	astTestMust(t, e)
	astTestPin(t, m, no)
	hash, e := m.GetHash(b)
	astTestMust(t, e)
	override := testSomeValueProc{value: yes}
	got, e := m.GetSomeValueWithProc(b, &override)
	astTestMust(t, e)
	if got != yes || override.calls != 1 || override.sort != b || override.hash != hash || m.someValueProc != nil {
		t.Fatal("scoped callback before plugin", got, override)
	}
	failure := errors.New("callback failure")
	inner := testSomeValueProc{failure: failure}
	outer := testSomeValueProc{manager: m, nested: &inner, failure: failure}
	if _, e = m.GetSomeValueWithProc(b, &outer); !errors.Is(e, failure) {
		t.Fatal("callback error", e)
	}
	if inner.calls != 1 || m.someValueProc != nil {
		t.Fatal("nested callback escaped")
	}
	got, e = m.GetSomeValue(b)
	astTestMust(t, e)
	if got != no {
		t.Fatal("plugin after scoped failure", got)
	}
}
func TestASTSomeValueProcFallbackReservationFailureRestores(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "callback-refusal"), nil)
	astTestMust(t, e)
	astTestPin(t, m, s)
	callback := testSomeValueProc{}
	m.Budget().FailAfter(0)
	_, e = m.GetSomeValueWithProc(s, &callback)
	m.Budget().FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || callback.calls != 1 || m.someValueProc != nil {
		t.Fatal("fallback refusal/restoration", e, callback.calls)
	}
	value, e := m.GetSomeValue(s)
	astTestMust(t, e)
	astTestPin(t, m, value)
}
