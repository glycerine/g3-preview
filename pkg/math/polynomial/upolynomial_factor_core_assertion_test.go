package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"testing"
)

func TestUPolynomialFactorPrimitiveLeadingAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			_, s := preparationPhysicalFixture(t)
			m := s.manager
			p, out := upolySourceNew(t, m, []int32{-1, -2}), upolySourceNew(t, m, []int32{99})
			d, e := m.descriptor(p)
			serviceMust(t, e)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			ok, e := s.factorSqfPrimitive(&d.state, fs, 1, DefaultUPolynomialFactorParams())
			if polynomialSourceDebug && action == "T" {
				if ok || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || fs.count != 0 {
					t.Fatal("negative-leading Throw", ok, e)
				}
				return
			}
			serviceMust(t, e)
			if !ok || fs.count != 1 || fs.totalDegree != 1 || fs.totalFactors != 1 {
				t.Fatal("linear Continue dispatch", ok, fs.count)
			}
			serviceMust(t, fs.Multiply(out))
			upolyGCDExpect(t, m, out, []int64{-1, -2})
			upolyGCDExpect(t, m, p, []int64{-1, -2})
		})
	}
}
func TestUPolynomialFactorQuadraticSizeAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			_, s := preparationPhysicalFixture(t)
			m := s.manager
			p := upolySourceNew(t, m, []int32{-1, 1})
			d, e := m.descriptor(p)
			serviceMust(t, e)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			// Qualified direct size-invariant adapter. No parent/frontend frame exists
			// or is manufactured: Continue reaches the actual first guarded p[2] read.
			e = s.factorQuadratic(&d.state, fs, 1)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("quadratic size Throw", e)
				}
			} else if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("quadratic first source read", e)
			}
			if fs.count != 0 || s.frontend.quadratic.constructed != 0 || s.frontend.quadratic.invocation != 0 {
				t.Fatal("size guard fabricated numerical frame")
			}
			upolyGCDExpect(t, m, p, []int64{-1, 1})
		})
	}
}
func TestUPolynomialFactorPrimitiveEmptyClosedGate(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	p := upolySourceNew(t, m, nil)
	d, e := m.descriptor(p)
	serviceMust(t, e)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	ok, e := s.factorSqfPrimitive(&d.state, fs, 1, DefaultUPolynomialFactorParams())
	if ok || !errors.Is(e, arena.ErrClosed) || fs.count != 0 {
		t.Fatal("closed entry gate crossed empty dispatch", ok, e)
	}
}
