// Source: src/util/rational.h:238-240,421-423; src/util/mpq.h:573-581.
package util

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalServiceComparisonInvocationSourceBranches(t *testing.T) {
	s, budget := serviceNew(t)
	for _, row := range []struct {
		a, b          string
		less, greater bool
	}{{"0", "3", true, false}, {"-7", "5", true, false}, {"7", "5", false, true}, {"1/2", "2/3", true, false}, {"-1/2", "-2/3", false, true}, {"1/2", "1/2", false, false}} {
		a := serviceValue(t, s, row.a)
		b := serviceValue(t, s, row.b)
		var calls [2]RationalServiceInvocation
		got, e := s.LessWithInvocation(&a, &b, &calls[0])
		serviceMust(t, e)
		greater, e := s.GreaterWithInvocation(&a, &b, &calls[1])
		serviceMust(t, e)
		if got != row.less || greater != row.greater {
			t.Fatal("comparison result", row, got, greater)
		}
		for i := range calls {
			serviceMust(t, calls[i].SourceReady())
			serviceMust(t, calls[i].Cleanup(false))
			serviceMust(t, calls[i].Cleanup(true))
			serviceMust(t, calls[i].EndContext())
		}
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&b))
	}
	a := serviceValue(t, s, "-1")
	b := serviceValue(t, s, "1/2")
	var c RationalServiceInvocation
	before := budget.Used()
	budget.FailAfter(0)
	got, e := s.LessWithInvocation(&a, &b, &c)
	budget.FailAfter(-1)
	serviceMust(t, e)
	if !got || budget.Used() != before || c.compareGeneration != 0 {
		t.Fatal("sign shortcut entered cross products")
	}
	serviceMust(t, c.Cleanup(false))
	serviceMust(t, c.Cleanup(true))
	serviceMust(t, c.EndContext())
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&b))
}

func TestRationalServiceComparisonInvocationEveryActualReservation(t *testing.T) {
	for _, greater := range []bool{false, true} {
		t.Run(fmt.Sprint(greater), func(t *testing.T) {
			complete, failures := false, 0
			for ordinal := int64(0); ordinal < 100 && !complete; ordinal++ {
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					s, budget := serviceNew(t)
					n := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 2048), big.NewInt(17))
					d := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 1024), big.NewInt(3))
					a := serviceValue(t, s, n.String()+"/"+d.String())
					b := serviceValue(t, s, "7/3")
					beforeA, beforeB := a, b
					// An older completed comparison must not adopt the next call's
					// still-active raw cross-products during its owning cleanup.
					var older RationalServiceInvocation
					_, e := s.LessWithInvocation(&a, &b, &older)
					serviceMust(t, e)
					var c RationalServiceInvocation
					budget.FailAfter(ordinal)
					var got bool
					var body error
					if greater {
						got, body = s.GreaterWithInvocation(&a, &b, &c)
					} else {
						got, body = s.LessWithInvocation(&a, &b, &c)
					}
					budget.FailAfter(-1)
					if a != beforeA || b != beforeB {
						t.Fatal("comparison modified operands")
					}
					qopsAuditLiveOwnership(t, s.Manager, []*Int{&a.Raw.Num, &a.Raw.Den, &b.Raw.Num, &b.Raw.Den})
					if want := s.HeaderBytes + s.CacheBytes + s.Manager.HeaderBytes + integerActualOwnedBytes(s.Manager.Z); budget.Used() != want {
						t.Fatal("actual comparison graph", budget.Used(), want)
					}
					if body != nil {
						failures++
						if !numericPhysicalReservationRefusal(body) || c.compareGeneration == 0 {
							t.Fatal("comparison prefix", body, c.compareGeneration)
						}
						if _, e := s.LessWithInvocation(&a, &b, &c); !errors.Is(e, arena.ErrInvalidSlot) {
							t.Fatal("stopped call reused", e)
						}
					} else {
						complete = true
						if got != greater {
							t.Fatal("large comparison result", got)
						}
					}
					bodyHistory, sourceHistory := c.bodyCause, c.sourceCause
					rawBefore := s.Manager.RawArithmetic
					usedBefore := budget.Used()
					serviceMust(t, older.Cleanup(false))
					serviceMust(t, older.Cleanup(true))
					if s.Manager.RawArithmetic.CompareActive != rawBefore.CompareActive || s.Manager.RawArithmetic.CompareOne != rawBefore.CompareOne || s.Manager.RawArithmetic.CompareTwo != rawBefore.CompareTwo || budget.Used() != usedBefore {
						t.Fatal("older invocation adopted newer comparison storage")
					}
					serviceMust(t, older.EndContext())
					serviceMust(t, c.Cleanup(false))
					serviceMust(t, c.Cleanup(true))
					if c.bodyCause != bodyHistory || c.sourceCause != sourceHistory {
						t.Fatal("history changed")
					}
					if s.Manager.RawArithmetic.CompareActive {
						t.Fatal("actual comparison locals leaked")
					}
					serviceMust(t, c.EndContext())
					serviceMust(t, s.Del(&a))
					serviceMust(t, s.Del(&b))
				})
			}
			if !complete || failures == 0 {
				t.Fatal("comparison horizon", complete, failures)
			}
		})
	}
}
