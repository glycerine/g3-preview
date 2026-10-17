package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestAlgebraicInPlaceRefinementOwnsActualHornerAndRetries(t *testing.T) {
	for _, precision := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "precision"}[precision], func(t *testing.T) {
			c, p := algebraicCellSourceFixtureProfile(t, true)
			root := algebraicTinyBinaryBoundsRoot(t, c)
			run := func() error {
				if precision {
					return c.RefineToPrecision(&root, 0)
				}
				return c.Refine(&root)
			}
			pad := algebraicCellSourceTLSBoundary(t, c)
			body := run()
			if body == nil || !c.refinement.evaluationPending(c) || !c.refinement.pending(c) {
				t.Fatal("missing actual in-place Horner child", body)
			}
			if c.bounds.refinement.owner != nil {
				t.Fatal("in-place refinement borrowed the bounds owner")
			}
			if e := run(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("pending in-place refinement replaced", e)
			}
			runtime := c.config.Rationals.Integers().RuntimeOwner()
			serviceMust(t, runtime.SetMaxSize(0))
			serviceMust(t, runtime.Deallocate(&pad))
			serviceMust(t, c.CleanupRetained())
			if c.refinement.pending(c) {
				t.Fatal("core cleanup did not acknowledge in-place child")
			}
			p.config.Checkpoint = func() bool { return true }
			p.config.Cooperation = func() error { return nil }
			serviceMust(t, run())
			var upper unumeral.BinaryRational
			serviceMust(t, c.GetUpper(&root, &upper))
			want, k := int32(3), uint32(65)
			if precision {
				want, k = 1, 63
			}
			if upper.BorrowNumerator().Val != want || upper.K() != k {
				t.Fatal("independent refined upper bound", upper, want, k)
			}
			serviceMust(t, c.config.Binary.Del(&upper))
			serviceMust(t, c.Del(&root))
		})
	}
}

func TestAlgebraicInPlacePrecisionRefinementExecutingCheckpoint(t *testing.T) {
	c, p, _ := algebraicCompareFixture(t)
	root := algebraicWideDyadicRoot(t, c)
	p.config.Cooperation = func() error { return nil }
	checkpoints := 0
	p.config.Checkpoint = func() bool {
		checkpoints++
		if !c.refinement.executing || c.refinement.live != 1<<boundRefinementWidth {
			t.Fatal("actual in-place width span missing")
		}
		for _, call := range []func() error{c.CleanupRetained, c.Close, func() error { return c.Refine(&root) }, func() error { return c.RefineToPrecision(&root, 2) }, func() error { return c.Del(&root) }} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("callback obtained executing owner", e)
			}
		}
		return false
	}
	body := c.RefineToPrecision(&root, 2)
	if !errors.Is(body, ErrUPolynomialCanceled) || checkpoints != 1 || c.refinement.executing || c.refinement.live != 0 {
		t.Fatal("source cancellation/width lifetime", body, checkpoints)
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicInPlaceRefinementOwningManagerCloseCallback(t *testing.T) {
	cfg, _, _ := algebraicManagerFixture(t)
	m, e := NewAlgebraicManager(cfg)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.Close()) })
	c := m.AlgebraicRootCore
	root := algebraicWideDyadicRoot(t, c)
	p := c.config.Polynomials
	p.config.Cooperation = func() error { return nil }
	p.config.Checkpoint = func() bool {
		if e := m.Close(); !errors.Is(e, ErrAlgebraicPending) {
			t.Fatal("owner Close entered executing refinement", e)
		}
		if m.finishing || m.closed || m.cleanupCause != nil {
			t.Fatal("refused Close changed owning manager lifecycle")
		}
		return false
	}
	if e := m.RefineToPrecision(&root, 1); !errors.Is(e, ErrUPolynomialCanceled) {
		t.Fatal(e)
	}
	serviceMust(t, m.Del(&root))
}

func TestAlgebraicInPlaceRefinementPreservesOlderHorner(t *testing.T) {
	c, p := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicTinyBinaryBoundsRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	a := unumeral.NewBinaryIntPower(1, 64)
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := p.EvalSignBinary(v, &a)
	if body == nil || !p.binaryEvaluation.active {
		t.Fatal("missing actual older evaluation", body)
	}
	older := p.binaryEvaluationReceipt()
	runtime := c.config.Rationals.Integers().RuntimeOwner()
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	body = c.RefineToPrecision(&root, 0)
	if !errors.Is(body, ErrUPolynomialCleanupPending) || c.refinement.evaluation.owner != nil {
		t.Fatal("in-place refinement acquired refused older entry", body)
	}
	state, used, stats := p.binaryEvaluation, p.config.Integer.Budget().Used(), runtime.Stats()
	serviceMust(t, c.CleanupRetained())
	if !reflect.DeepEqual(state, p.binaryEvaluation) || used != p.config.Integer.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("core cleanup consumed older Horner")
	}
	serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&older, false))
	serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&older, true))
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicInPlaceRefinementPublishesBeforeCancellationReturn(t *testing.T) {
	c, p, _ := algebraicCompareFixture(t)
	root := algebraicWideDyadicRoot(t, c)
	checkpoints := 0
	p.config.Cooperation = func() error { return nil }
	p.config.Checkpoint = func() bool { checkpoints++; return checkpoints < 2 }
	body := c.RefineToPrecision(&root, 8)
	if !errors.Is(body, ErrUPolynomialCanceled) || checkpoints != 2 {
		t.Fatal(body, checkpoints)
	}
	var lower unumeral.BinaryRational
	serviceMust(t, c.GetLower(&root, &lower))
	text, e := c.config.Rationals.Integers().String(lower.BorrowNumerator())
	serviceMust(t, e)
	if text != "36893488147419103233" || lower.K() != 65 {
		t.Fatal("completed source bisection was lost on error", text, lower.K())
	}
	if c.refinement.executing || c.refinement.pending(c) {
		t.Fatal("completed source destruction retained in-place span")
	}
	serviceMust(t, c.config.Binary.Del(&lower))
	serviceMust(t, c.Del(&root))
}
