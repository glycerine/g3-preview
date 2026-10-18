package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialSequenceSturmProtectsBorrowedNumericOwners(t *testing.T) {
	c, m, _ := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	s, e := m.NewSequence()
	serviceMust(t, e)
	z, n := m.config.Integer, m.config.Integer.Naturals()
	inner, e := m.NewSequence()
	serviceMust(t, e)
	beforeInteger, beforeNatural := z.CallerBindings(), n.CallerBindings()
	probed := false
	executing := func(binding *unumeral.NumericCallerBinding) bool {
		for b := binding; b != nil; b = b.Previous() {
			if b.Guard() != nil && b.Guard().NumericExecuting() {
				return true
			}
		}
		return false
	}
	m.config.Checkpoint = func() bool {
		if probed {
			return true
		}
		probed = true
		// Avoid destroying the fixture when running this regression before the fix.
		if !executing(z.CallerBindings()) || !executing(n.CallerBindings()) {
			t.Error("Sturm checkpoint has no executing numeric-owner protection")
			return false
		}
		used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
		for _, e := range []error{z.SourceFinalize(), z.CloseChecked(), c.config.Rationals.SourceFinalize(), c.config.Rationals.CloseChecked()} {
			if !errors.Is(e, ErrPolynomialPending) {
				t.Error("checkpoint acquired borrowed numeric owner", e)
			}
		}
		for _, e := range []error{n.SourceFinalize(), n.CloseChecked()} {
			if !errors.Is(e, arena.ErrInvalidSlot) {
				t.Error("checkpoint acquired borrowed natural owner", e)
			}
		}
		if used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
			t.Error("numeric-owner refusals changed physical graph")
		}
		integerHead, naturalHead := z.CallerBindings(), n.CallerBindings()
		if e := m.SturmSequence(v, inner); e != nil {
			t.Error("nested Sturm call", e)
		}
		if z.CallerBindings() != integerHead || n.CallerBindings() != naturalHead || !executing(z.CallerBindings()) || !executing(n.CallerBindings()) {
			t.Error("nested Sturm call did not restore outer numeric guards")
		}
		return true
	}
	m.config.Cooperation = func() error { return nil }
	body := m.SturmSequence(v, s)
	if body != nil && !errors.Is(body, ErrUPolynomialCanceled) {
		t.Error(body)
	}
	if !probed {
		t.Error("actual remainder checkpoint was not reached")
	}
	if z.CallerBindings() != beforeInteger || n.CallerBindings() != beforeNatural || executing(z.CallerBindings()) || executing(n.CallerBindings()) {
		t.Error("exited Sturm call retained executing numeric guard")
	}
	serviceMust(t, inner.Close())
	serviceMust(t, s.Close())
	serviceMust(t, c.Del(&root))
}
