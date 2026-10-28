package ast

// Source: src/ast/rewriter/bv_trailing.cpp:149-171.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestASTBitVectorNumeralScopeOriginalLifetime(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	input, e := m.BitVectorNumeralUnsigned(1, 8)
	astTestMust(t, e)
	out := astTestPin(t, m, input)
	var scope ASTBitVectorNumeralScope
	yes, one, width, e := m.ReadBitVectorNumeralScope(input, &scope)
	astTestMust(t, e)
	if !yes || !one || width != 8 || scope.frame == nil || scope.frame.consumed[0] {
		t.Fatal("copied c_val must remain live", yes, one, width)
	}
	if e := m.BitVectorNumeralScopeSourceReady(&scope); !errors.Is(e, ErrASTPending) {
		t.Fatal("live scope granted older DT", e)
	}
	copy := scope
	if e := m.EndBitVectorNumeralScope(&copy, nil); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied scope accepted", e)
	}
	result, e := m.BitVectorNumeralUnsignedInScope(1, 7, &scope)
	astTestMust(t, e)
	astTestMust(t, m.SetRef(out, result))
	if scope.frame.consumed[0] {
		t.Fatal("c_val ended before result publication")
	}
	astTestMust(t, m.EndBitVectorNumeralScope(&scope, nil))
	if !scope.frame.consumed[0] || !scope.frame.retired {
		t.Fatal("source footer did not retire c_val")
	}
	// A later call can reuse the embedded primary frame. A retired token
	// must not inspect or destroy that new invocation during owning cleanup.
	_, e = m.BitVectorNumeralUnsigned(7, 8)
	astTestMust(t, e)
	astTestMust(t, m.CleanupBitVectorNumeralScope(&scope))
	astTestMust(t, m.BitVectorNumeralScopeSourceReady(&scope))
	if e := m.EndBitVectorNumeralScope(&scope, nil); !errors.Is(e, ErrASTPending) {
		t.Fatal("source footer replayed", e)
	}
}

func TestASTBitVectorNumeralScopeStoppedAndForeignChildren(t *testing.T) {
	for _, captured := range []bool{false, true} {
		t.Run(fmt.Sprint(captured), func(t *testing.T) {
			m := astTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			input, e := m.BitVectorNumeralUnsigned(1, 8)
			astTestMust(t, e)
			astTestPin(t, m, input)
			var scope ASTBitVectorNumeralScope
			_, _, _, e = m.ReadBitVectorNumeralScope(input, &scope)
			astTestMust(t, e)
			if e := m.CleanupBitVectorNumeralScope(&scope); !errors.Is(e, ErrASTPending) {
				t.Fatal("cleanup before source exit", e)
			}
			var body error
			if captured {
				_, body = m.BitVectorNumeralUnsignedInScope(1, 0, &scope)
				if _, e := m.BitVectorNumeralUnsignedInScope(1, 8, &scope); !errors.Is(e, ErrASTPending) {
					t.Fatal("nested call site reused", e)
				}
			} else {
				_, body = m.BitVectorNumeralUnsigned(1, 0)
			}
			var exception ASTException
			if !errors.As(body, &exception) {
				t.Fatal("source zero-width exception", body)
			}
			if scope.frame.consumed[0] {
				t.Fatal("body ended older c_val")
			}
			if captured {
				body = m.EndBitVectorNumeralScope(&scope, body)
				if !errors.As(body, &exception) || !scope.frame.sourceDone || !scope.frame.consumed[0] || scope.retired {
					t.Fatal("captured source footer", body)
				}
				astTestMust(t, m.BitVectorNumeralScopeSourceReady(&scope))
			} else {
				// Without an exact child receipt, the footer cannot reach
				// older DT over an independently stopped call.
				if e := m.EndBitVectorNumeralScope(&scope, body); !errors.Is(e, ErrASTPending) || !errors.As(e, &exception) {
					t.Fatal("uncaptured child ignored", e)
				}
				if scope.frame.consumed[0] {
					t.Fatal("uncaptured child granted older source DT")
				}
				if e := m.BitVectorNumeralScopeSourceReady(&scope); !errors.Is(e, ErrASTPending) {
					t.Fatal("foreign source granted older DT", e)
				}
			}
			astTestMust(t, m.CleanupBitVectorNumeralScope(&scope))
			if !captured {
				if e := m.BitVectorNumeralScopeSourceReady(&scope); !errors.Is(e, ErrASTPending) {
					t.Fatal("owning cleanup resumed source DT", e)
				}
			}
		})
	}
}

func TestASTBitVectorNumeralScopeEveryReachedReservation(t *testing.T) {
	for _, nested := range []bool{false, true} {
		success, failures, capturedChild := false, 0, false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			t.Run(fmt.Sprint(nested, "/", ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				var input ASTHandle
				if nested {
					input, e = m.BitVectorNumeralUnsigned(1, 8)
					astTestMust(t, e)
					astTestPin(t, m, input)
				} else {
					sort, e := m.BitVectorSort(8)
					astTestMust(t, e)
					input = bvTrailingColdTestNumeral(t, m, fid, sort)
				}
				var scope ASTBitVectorNumeralScope
				m.Budget().FailAfter(ordinal)
				matched, one, width, body := m.ReadBitVectorNumeralScope(input, &scope)
				if body == nil && nested {
					_, body = m.BitVectorNumeralUnsignedInScope(^uint32(0), 65, &scope)
				}
				body = m.EndBitVectorNumeralScope(&scope, body)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !errors.Is(body, arena.ErrInjectedFailure) {
						var f umemory.Failure
						if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
							t.Fatal("unexpected refusal", body)
						}
					}
					if scope.frame != nil {
						f := scope.frame
						if f.retired {
							t.Fatal("body failure retired its caller")
						}
						found := false
						for n := m.bitVectorPlugin.numeralFrames; n != nil; n = n.next {
							if n == f {
								found = true
							}
						}
						if !found {
							t.Fatal("older scope disappeared beneath its stopped child")
						}
						if f.normalizationChild.frame != nil {
							capturedChild = true
							if f.sourceCause != nil && f.consumed[0] {
								t.Fatal("c_val consumed past stopped child DT")
							}
						}
					}
					if _, e := m.BitVectorNumeralUnsignedInScope(1, 8, &scope); !errors.Is(e, ErrASTPending) {
						t.Fatal("stopped scope retried", e)
					}
				} else {
					success = true
					if !matched || one != nested || width != 8 || !scope.retired {
						t.Fatal("scope result", matched, one, width)
					}
				}
				astTestMust(t, m.CleanupBitVectorNumeralScope(&scope))
				astTestMust(t, m.CleanupBitVectorNumeralScope(&scope))
			})
			if success {
				break
			}
		}
		if !success || failures == 0 || nested && !capturedChild {
			t.Fatal("scope allocation horizon", nested, success, failures, capturedChild)
		}
	}
}

func TestASTBitVectorNumeralScopeErasedGuard(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	before := m.Stats()
	used := m.Budget().Used()
	m.Budget().FailAfter(0)
	var scope ASTBitVectorNumeralScope
	yes, one, width, e := m.ReadBitVectorNumeralScope(x, &scope)
	astTestMust(t, e)
	astTestMust(t, m.EndBitVectorNumeralScope(&scope, nil))
	astTestMust(t, m.CleanupBitVectorNumeralScope(&scope))
	m.Budget().FailAfter(-1)
	if yes || one || width != 0 || scope.frame != nil || used != m.Budget().Used() || before != m.Stats() {
		t.Fatal("nonnumeral guard allocated or read numerical metadata")
	}
}

func TestASTBitVectorNumeralScopeStoredOnePredicate(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	for _, literal := range []string{"1", "0", "-1", "257", "340282366920938463463374607431768211457"} {
		input := bvRawOffsetTestNumeral(t, m, fid, sort, literal, 8)
		var scope ASTBitVectorNumeralScope
		yes, one, width, e := m.ReadBitVectorNumeralScope(input, &scope)
		astTestMust(t, e)
		text, e := m.numerals.String(&scope.frame.values[0])
		astTestMust(t, e)
		if !yes || one != (literal == "1") || width != 8 || text != literal {
			t.Fatal("stored c_val was normalized", literal, yes, one, width, text)
		}
		astTestMust(t, m.EndBitVectorNumeralScope(&scope, nil))
		astTestMust(t, m.CleanupBitVectorNumeralScope(&scope))
	}
}
