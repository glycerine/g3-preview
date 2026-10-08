// Source: src/ast/arith_decl_plugin.cpp:400-449.
package ast

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTArithmeticNumeralInvocationEveryActualReservation(t *testing.T) {
	for _, tc := range []struct {
		text    string
		integer bool
	}{
		{"15", true}, {"18446744073709551617", true},
		{"18446744073709551617/18446744073709551619", false},
	} {
		complete, sawCachePending := false, false
		for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", tc.text, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				v, e := m.numerals.NewValue()
				astTestMust(t, e)
				defer func() { astTestMust(t, m.numerals.Del(&v)) }()
				astTestMust(t, m.numerals.SetString(&v, tc.text))
				if tc.text == "15" {
					astTestMust(t, m.numerals.SetInt64(&v, 0))
				}
				var older ASTArithmeticNumeralInvocation
				old, e := m.ArithmeticNumeralWithInvocation(&v, tc.integer, &older)
				astTestMust(t, e)
				oldRef := astTestPin(t, m, old)
				// Use a distinct uncached value so the younger call reaches a new frame.
				if tc.text == "15" {
					astTestMust(t, m.numerals.SetInt64(&v, 15))
				} else {
					astTestMust(t, m.numerals.NegAssign(&v))
				}
				var call ASTArithmeticNumeralInvocation
				m.Budget().FailAfter(ordinal)
				h, body := m.ArithmeticNumeralWithInvocation(&v, tc.integer, &call)
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					astTestMust(t, m.ArithmeticNumeralInvocationSourceReady(&call))
					astTestPin(t, m, h)
					r, e := m.internal(h)
					astTestMust(t, e)
					info, e := m.funcInfo(m.declAt(m.appAt(r.slot).decl.slot).info)
					astTestMust(t, e)
					token, e := m.infoParameter(info, 0)
					astTestMust(t, e)
					p, e := m.parameter(token)
					astTestMust(t, e)
					equal, e := m.numerals.Equal(&p.Rational, &v)
					astTestMust(t, e)
					if !equal {
						t.Fatal("factory changed value")
					}
				} else {
					var f util.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != util.FailureMemout) {
						t.Fatal(body)
					}
				}
				if call.frame == nil || call.invocation == 0 {
					t.Fatal("missing actual frame")
				}
				frame := call.frame
				active, invocation, source := frame.active, frame.invocation, frame.sourceCause
				astTestMust(t, m.CleanupArithmeticNumeralInvocation(&older))
				if frame.active != active || frame.invocation != invocation || frame.sourceCause != source {
					t.Fatal("older cleanup adopted younger frame")
				}
				if _, e = m.RefGet(oldRef); e != nil {
					t.Fatal("older returned value lost", e)
				}
				copy := call
				if e = m.CleanupArithmeticNumeralInvocation(&copy); e != ErrASTHandle {
					t.Fatal("copied invocation", e)
				}
				savedBody, savedSource := call.bodyCause, call.sourceCause
				pending := m.arithmeticPlugin.cachePending
				if pending.slot != 0 {
					sawCachePending = true
				}
				astTestMust(t, m.CleanupArithmeticNumeralInvocation(&call))
				astTestMust(t, m.CleanupArithmeticNumeralInvocation(&call))
				if frame.active || !call.retired {
					t.Fatal("exact child not joined")
				}
				if savedBody != call.bodyCause || savedSource != call.sourceCause {
					t.Fatal("history changed")
				}
				if pending != m.arithmeticPlugin.cachePending {
					t.Fatal("lexical cleanup adopted plugin cache pin")
				}
				if body != nil {
					if _, e = m.ArithmeticNumeralWithInvocation(&v, tc.integer, &call); e != ErrASTPending {
						t.Fatal("stopped receipt reused", e)
					}
				}
			})
		}
		if !complete {
			t.Fatal("campaign incomplete", tc.text)
		}
		if tc.text == "15" && !sawCachePending {
			t.Fatal("cache INC/resize frontier not reached")
		}
	}
}

func TestASTArithmeticNumeralInvocationCacheAndGuardHaveNoChild(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	v, e := m.numerals.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, m.numerals.Del(&v)) }()
	astTestMust(t, m.numerals.SetInt64(&v, 15))
	var call ASTArithmeticNumeralInvocation
	h, e := m.ArithmeticNumeralWithInvocation(&v, true, &call)
	astTestMust(t, e)
	astTestMust(t, m.CleanupArithmeticNumeralInvocation(&call))
	m.Budget().FailAfter(0)
	again, e := m.ArithmeticNumeralWithInvocation(&v, true, &call)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if h != again || call.frame != nil || call.invocation != 0 {
		t.Fatal("cache hit invented lexical child")
	}
	astTestMust(t, m.CleanupArithmeticNumeralInvocation(&call))
	astTestMust(t, m.numerals.SetString(&v, "1/3"))
	var rejected ASTArithmeticNumeralInvocation
	if _, e = m.ArithmeticNumeralWithInvocation(&v, true, &rejected); e == nil {
		t.Fatal("fraction accepted")
	}
	if rejected.frame != nil {
		t.Fatal("guard invented parameter")
	}
	astTestMust(t, m.CleanupArithmeticNumeralInvocation(&rejected))
	foreign := astTestManager(t)
	if e = foreign.CleanupArithmeticNumeralInvocation(&call); e != ErrASTHandle {
		t.Fatal("foreign context", e)
	}
}

func TestASTArithmeticNumeralInvocationCapturesExecutingCopy(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ASTManager
	var call ASTArithmeticNumeralInvocation
	active, calls := false, 0
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, Transport: func(f util.Failure) error {
		if active {
			calls++
			if !call.active || call.frame != &m.arithmeticPlugin.numeral || call.invocation == 0 || call.frame.invocation != call.invocation {
				t.Fatal("missing executing child capture")
			}
			if e := m.CleanupArithmeticNumeralInvocation(&call); e != ErrASTPending {
				t.Fatal("executing child cleanup", e)
			}
			if e := m.ArithmeticNumeralInvocationSourceReady(&call); e != ErrASTPending {
				t.Fatal("executing source readiness", e)
			}
			if e := m.Close(); e != ErrASTPending {
				t.Fatal("executing owner cleanup", e)
			}
		}
		return f
	}})
	astTestMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: b})
	astTestMust(t, e)
	m, e = NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	v, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetString(&v, "18446744073709551617/18446744073709551619"))
	astTestMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+1))
	active = true
	_, body := m.ArithmeticNumeralWithInvocation(&v, false, &call)
	active = false
	if body == nil || calls == 0 || call.active || m.formatExecuting {
		t.Fatal("copy transport not reached or restored", body, calls)
	}
	astTestMust(t, rt.SetMaxCount(0))
	savedBody, savedSource := call.bodyCause, call.sourceCause
	astTestMust(t, m.CleanupArithmeticNumeralInvocation(&call))
	if savedBody != call.bodyCause || savedSource != call.sourceCause {
		t.Fatal("history lost")
	}
	astTestMust(t, q.Del(&v))
	astTestMust(t, m.Close())
	astTestMust(t, q.Close())
	astTestMust(t, sy.Close())
	astTestMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("executing copy graph retained", b.Used())
	}
}
