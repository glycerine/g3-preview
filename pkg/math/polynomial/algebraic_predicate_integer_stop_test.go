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

func TestAlgebraicPredicateIntegerHornerStopRetainsCandidate(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 4096 && !reached; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			var c *AlgebraicRootCore
			var p *UPolynomialManager
			var runtime *umemory.Runtime
			var lane umemory.Block
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || c == nil || !c.predicate.arithmeticExecuting || !p.integerEvaluation.executing {
					return f
				}
				z := c.config.Rationals.Integers()
				ref := p.integerEvaluation.values[0].Cell
				if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
					return f
				}
				armed = false
				reached = true
				for _, e := range []error{c.Close(), c.CleanupRetained(), c.cleanupPredicate(), c.finishPredicate(nil)} {
					if !errors.Is(e, ErrAlgebraicPending) {
						t.Fatal("integer Horner callback obtained lifecycle", e)
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
			root := algebraicIntegerPredicateRoot(t, c)
			v := algebraicPredicateBeforeIntegerEvaluation(t, c, &root)
			z := c.config.Rationals.Integers()
			original := c.predicate.candidate
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent")
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			armed = true
			_, body := c.evalPredicateInteger(v)
			armed = false
			serviceMust(t, runtime.SetMaxSize(0))
			_ = c.finishPredicate(body)
			s := &c.predicate
			if reached {
				if !errors.Is(body, arena.ErrBounds) || s.childStop == nil || !p.matchesIntegerEvaluationReceipt(&s.integerEvaluation) || !p.integerEvaluation.active || s.stage != 0 || !s.active || s.candidate != original || z.Cell(original.Cell).Released {
					t.Fatal("integer Horner stop destroyed older candidate", body, s.stage)
				}
				if _, e := c.IsInt(&root); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("stopped integer predicate was replaced", e)
				}
				before, used, stats := *s, z.Budget().Used(), runtime.Stats()
				if e := c.finishPredicate(nil); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				if !reflect.DeepEqual(before, *s) || used != z.Budget().Used() || stats != runtime.Stats() {
					t.Fatal("predicate source finalizer replayed")
				}
			}
			contextTLSFinish(t, runtime, c.CleanupRetained)
			if reached && (s.active || s.stage != 0 || s.ownStage != 5 || p.integerEvaluation.active) {
				t.Fatal("integer owning cleanup changed source cursor")
			}
			contextTLSFinish(t, runtime, func() error { return c.CleanupValue(&root) })
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
		t.Fatal("did not reach actual nested integer Horner source stop")
	}
}

func TestAlgebraicPredicateIntegerSetterSourceStopRetainsCandidate(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 1024 && !reached; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			var c *AlgebraicRootCore
			var runtime *umemory.Runtime
			var lane umemory.Block
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || c == nil || !c.integerSet.running || !c.localActive || c.localIntegerSet != &c.integerSet {
					return f
				}
				z := c.config.Rationals.Integers()
				ref := c.integerSet.value.BorrowNumerator().Cell
				if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released || !z.IsFailureRef(c.localQ.BorrowNumerator().Cell) {
					return f
				}
				armed = false
				reached = true
				for _, e := range []error{c.Close(), c.CleanupRetained(), c.cleanupIntegerSet()} {
					if !errors.Is(e, ErrAlgebraicPending) {
						t.Fatal("setter callback obtained lifecycle", e)
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
			c, _, runtime = algebraicNegTLSFixture(t, transport)
			root := algebraicIntegerPredicateRoot(t, c)
			v := algebraicPredicateBeforeIntegerEvaluation(t, c, &root)
			sign, e := c.evalPredicateInteger(v)
			serviceMust(t, e)
			if sign != 0 {
				t.Fatal("integer fixture not root")
			}
			z := c.config.Rationals.Integers()
			candidate := c.predicate.candidate
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent")
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			armed = true
			body := c.setIntegerWithInvocation(&root, &c.predicate.candidate, &c.predicate.setter)
			armed = false
			serviceMust(t, runtime.SetMaxSize(0))
			if c.matchesIntegerSetter(&c.predicate.setter) && c.integerSet.active {
				c.predicate.childStop = c.integerSet.sourceStop
			}
			_ = c.finishPredicate(body)
			s := &c.integerSet
			if reached {
				if !errors.Is(body, arena.ErrBounds) || !numericPhysicalReservationRefusal(body) || !s.active || s.sourceCursor != 0 || s.sourceStop == nil || !z.Cell(s.value.BorrowNumerator().Cell).Released || c.predicate.stage != 0 || c.predicate.candidate != candidate || z.Cell(candidate.Cell).Released {
					t.Fatal("setter source stop crossed candidate", body, s.sourceCursor)
				}
				if _, e := c.IsInt(&root); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("stopped integer predicate was replaced", e)
				}
				before, used, stats := *s, z.Budget().Used(), runtime.Stats()
				if e := c.finishIntegerSet(nil); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				if !reflect.DeepEqual(before, *s) || used != z.Budget().Used() || stats != runtime.Stats() {
					t.Fatal("setter source destructor replayed")
				}
				if e := c.cleanupPredicate(); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("candidate cleaned before setter", e)
				}
			}
			contextTLSFinish(t, runtime, c.CleanupRetained)
			if reached && (s.active || s.sourceCursor != 0 || s.ownCursor != 2 || c.predicate.active || c.predicate.stage != 0) {
				t.Fatal("setter owning cleanup changed source cursor")
			}
			contextTLSFinish(t, runtime, func() error { return c.CleanupValue(&root) })
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
		t.Fatal("did not reach actual scoped integer setter source stop")
	}
}
