// Source: src/math/polynomial/upolynomial.cpp:1734-1764;
// src/util/memory_manager.cpp:274-285. Returning-transport lifecycle fixture.
package polynomial

import (
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialEvalSignBinaryActualCallbackOwnerAndTupleRefusal(t *testing.T) {
	var c *AlgebraicRootCore
	var p *UPolynomialManager
	armed := false
	calls := 0
	transport := func(f util.Failure) error {
		if !armed {
			return f
		}
		calls++
		q := &p.binaryEvaluation
		before := *q
		z := p.config.Integer
		used := z.Budget().Used()
		physical := z.RuntimeOwner().Stats()
		r := p.binaryEvaluationReceipt()
		copied := *q
		ctx := q.context()
		ctx.binaryEvaluation = &copied
		if (ctx.roots(z)) != nil {
			t.Error("copied frame acquired numeric leaf")
		}
		stale := q.context()
		stale.binaryEvaluationInvocation--
		if (stale.roots(z)) != nil {
			t.Error("stale invocation acquired numeric leaf")
		}
		owner := *p
		owner.self = &owner
		owner.binaryEvaluation.owner = &owner
		if (owner.binaryEvaluation.context().roots(z)) != nil {
			t.Error("copied owner acquired numeric leaf")
		}
		foreign := r
		foreign.frame = &copied
		serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&foreign, true))
		for _, abort := range []bool{false, true} {
			if e := p.cleanupBinaryEvaluationWithReceipt(&r, abort); e != ErrUPolynomialCleanupPending {
				t.Errorf("executing exact cleanup %v", e)
			}
		}
		if e := q.finish(nil); e != ErrUPolynomialCleanupPending {
			t.Errorf("executing source finalize %v", e)
		}
		if e := p.SourceFinalize(); e != ErrUPolynomialCleanupPending {
			t.Errorf("UP source owner %v", e)
		}
		if e := p.Close(); e != ErrUPolynomialCleanupPending {
			t.Errorf("UP owning owner %v", e)
		}
		if e := z.SourceFinalize(); e != ErrPolynomialPending {
			t.Errorf("Integer source owner %v", e)
		}
		if e := z.CloseChecked(); e != ErrPolynomialPending {
			t.Errorf("Integer own owner %v", e)
		}
		if e := c.config.Rationals.SourceFinalize(); e != ErrPolynomialPending {
			t.Errorf("Rational source owner %v", e)
		}
		if e := c.config.Rationals.CloseChecked(); e != ErrPolynomialPending {
			t.Errorf("Rational own owner %v", e)
		}
		if !reflect.DeepEqual(before, *q) || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
			t.Error("refused callback changed exact charged frame/physical graph")
		}
		return f
	}
	var rt *util.Runtime
	c, p, rt = algebraicNegTLSFixture(t, transport)
	poly := algebraicPolynomial(t, p, 1, math.MaxInt32, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var b util.BinaryRational
	serviceMust(t, c.config.Binary.SetInt32(&b, math.MaxInt32))
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	_, body := p.EvalSignBinary(v, &b)
	armed = false
	if body == nil || calls == 0 {
		t.Fatal("actual allocating Horner transport not reached", body, calls)
	}
	q := &p.binaryEvaluation
	extra := []*util.Int{&(*b.BorrowNumerator()), &q.values[0], &q.values[1]}
	if q.addMul.Frame != nil {
		extra = append(extra, &q.addMul.Frame.Tmp)
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, extra, []util.Block{held})
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, p.cleanupBinaryEvaluation(false))
	serviceMust(t, c.config.Binary.Del(&b))
	serviceMust(t, p.cleanupBinaryEvaluation(true))
	serviceMust(t, rt.Deallocate(&held))
}
