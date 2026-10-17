package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

// Returning physical OOM transport plus actual signed-host deallocation guard.
// The accepted Horner receipt must stop the older scoped rational destructor.
func TestAlgebraicPredicateRationalHornerStopRetainsCandidate(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 4096 && !reached; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			var c *AlgebraicRootCore
			var p *UPolynomialManager
			var runtime *umemory.Runtime
			var lane umemory.Block
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || c == nil || !c.predicateQ.executing || !p.rationalEvaluation.executing {
					return f
				}
				z := c.config.Rationals.Integers()
				ref := p.rationalEvaluation.values[1].Cell
				if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
					return f
				}
				armed = false
				reached = true
				for _, e := range []error{c.CleanupRetained(), c.Close(), c.cleanupPredicateQ(), c.finishPredicateQ(nil), c.finishPredicate(nil)} {
					if !errors.Is(e, ErrAlgebraicPending) {
						t.Fatal("Horner callback obtained outer lifecycle", e)
					}
				}
				serviceMust(t, runtime.SetMaxSize(0))
				contextTLSNearMinimum(t, runtime)
				var e error
				lane, e = runtime.Allocate(8)
				serviceMust(t, e)
				delta := int64(-1000000)
				if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				return f
			}
			c, p, runtime = algebraicNegTLSFixture(t, transport)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			algebraicPredicateBeforeQ(t, c, &root)
			serviceMust(t, c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp))
			z := c.config.Rationals.Integers()
			originalNum, originalDen := *c.localQ.BorrowNumerator(), *c.localQ.BorrowDenominator()
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent")
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			armed = true
			_, body := c.evalPredicateRational(v)
			armed = false
			serviceMust(t, runtime.SetMaxSize(0))
			s := &c.predicateQ
			cause := c.finishPredicateQ(body)
			_ = c.finishPredicate(cause)
			if reached {
				if !errors.Is(body, arena.ErrBounds) || s.childStop == nil || !p.matchesRationalEvaluationReceipt(&s.evaluation) || !p.rationalEvaluation.active || s.sourceCursor != 0 || s.sourceStop != nil || !s.active || c.predicate.stage != 0 || c.predicate.restored {
					t.Fatal("Horner source stop lost", body, s.sourceCursor)
				}
				if originalNum != *c.localQ.BorrowNumerator() || originalDen != *c.localQ.BorrowDenominator() || z.Cell(originalNum.Cell).Released || z.Cell(originalDen.Cell).Released {
					t.Fatal("older rational destroyed before Horner child")
				}
				state, used, stats := *s, z.Budget().Used(), runtime.Stats()
				if e := c.finishPredicateQ(nil); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				if !reflect.DeepEqual(state, *s) || used != z.Budget().Used() || stats != runtime.Stats() {
					t.Fatal("source rational finalizer replayed after child stop")
				}
			}
			contextTLSFinish(t, runtime, c.CleanupRetained)
			if reached && (s.active || s.sourceCursor != 0 || s.ownCursor != 2 || s.childStop == nil || p.rationalEvaluation.active) {
				t.Fatal("own cleanup replayed source or left child active")
			}
			contextTLSFinish(t, runtime, func() error { return c.Del(&root) })
			for _, block := range []*umemory.Block{&lane, &pad} {
				contextTLSFinish(t, runtime, func() error {
					if *block == (umemory.Block{}) {
						return nil
					}
					return runtime.Deallocate(block)
				})
			}
		})
	}
	if !reached {
		t.Fatal("did not reach actual nested rational Horner destructor stop")
	}
}

func TestAlgebraicPredicateRationalHornerDoesNotAdoptOlderEvaluation(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	p := c.config.Polynomials
	root := algebraicLargeDyadicPromotionRoot(t, c)
	algebraicPredicateBeforeQ(t, c, &root)
	serviceMust(t, c.setPredicateQParts(&c.predicate.candidate, &c.isRationalTmp))
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := c.config.Rationals.Integers()
	runtime := z.RuntimeOwner()
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := p.EvalSignRational(v, &c.localQ)
	if body == nil || !p.rationalEvaluation.active {
		t.Fatal("older public evaluation did not retain actual allocation", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	prior := p.rationalEvaluation
	_, body = c.evalPredicateRational(v)
	if body == nil || c.predicateQ.evaluation.owner != nil || c.predicateQ.childStop != nil || !reflect.DeepEqual(prior, p.rationalEvaluation) {
		t.Fatal("refused candidate call acquired older Horner", body)
	}
	_ = c.finishPredicate(c.finishPredicateQ(body))
	serviceMust(t, c.CleanupRetained())
	if !reflect.DeepEqual(prior, p.rationalEvaluation) {
		t.Fatal("candidate owning cleanup consumed older Horner")
	}
	serviceMust(t, p.CleanupRationalEvaluation())
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicPredicateRationalHornerErasedCallsIssueNoReceipt(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	p := c.config.Polynomials
	constant, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(constant, []unumeral.Int{unumeral.NewInt(-7)}))
	v, e := p.wholeView(constant)
	serviceMust(t, e)
	for _, tc := range []struct {
		v    UPolynomialView
		want int
	}{{UPolynomialView{}, 0}, {v, -1}} {
		var receipt upolyRationalEvaluationInvocationReceipt
		got, e := p.evalSignRationalWithReceipt(tc.v, nil, &receipt)
		serviceMust(t, e)
		if got != tc.want || receipt.owner != nil {
			t.Fatal("zero/constant evaluation entered Horner scope", got, receipt.owner)
		}
	}
	serviceMust(t, p.DelPolynomial(&constant))
}
