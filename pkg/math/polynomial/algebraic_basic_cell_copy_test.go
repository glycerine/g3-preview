// Source: algebraic_numbers.cpp:424-438; algebraic_numbers.h:372.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicBasicCellCopyAccessorSwitch(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source accessor assertion erased")
	}
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt64(&q, 3))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	serviceMust(t, c.config.Rationals.SetInt64(&q, 5))
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	old := a
	if e := c.SetTake(&a, &q); !errors.Is(e, arena.ErrClosed) || a != old {
		t.Error("reused basic setter skipped accessor switch", e)
	}
	serviceMust(t, c.CleanupValue(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}
func TestAlgebraicBasicCellCopyActualTransport(t *testing.T) {
	var c *AlgebraicRootCore
	var a Algebraic
	armed := false
	calls := 0
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		old := a
		z := c.config.Rationals.Integers()
		used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
		if e := c.Del(&a); !errors.Is(e, ErrAlgebraicPending) {
			t.Error("basic copy callback admitted caller deletion", e)
			return f
		}
		for _, call := range []func() error{c.CleanupRetained, c.Close} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("basic copy callback acquired core", e)
			}
		}
		if a != old || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
			t.Error("refused callback changed ownership")
		}
		return f
	}
	c, _, rt := algebraicNegTLSFixture(t, transport)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt64(&q, 3))
	serviceMust(t, c.SetRational(&a, &q))
	serviceMust(t, c.config.Rationals.SetString(&q, "1267650600228229401496703205377/18446744073709551619"))
	beforeQ := q
	pad := algebraicCellSourceTLSBoundary(t, c)
	armed = true
	body := c.SetTake(&a, &q)
	armed = false
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&pad))
	if calls == 0 || body == nil {
		t.Fatal("actual rational SET transport missing", calls, body)
	}
	if q != beforeQ || !c.config.Rationals.Integers().MatchesCopyAllocationInvocation(&c.rationalCopy) {
		t.Fatal("reused setter changed its source or lost the accepted copy", q != beforeQ)
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.CleanupValue(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}
