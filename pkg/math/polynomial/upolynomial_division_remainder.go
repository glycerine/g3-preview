// Source: src/math/polynomial/upolynomial.cpp:516-567.
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func (m *UPolynomialManager) remState(a, b UPolynomialView, d *uint32, r *upolyVectorState) error {
	return m.remStateWithSourceContext(a, b, d, r, integerAllocationInvocationContext{})
}
func (m *UPolynomialManager) remStateWithSourceContext(a, b UPolynomialView, d *uint32, r *upolyVectorState, context integerAllocationInvocationContext) (cause error) {
	if d == nil {
		return arena.ErrBounds
	}
	if polynomialSourceDebug && (m.alias(a, r) || m.alias(b, r)) {
		return ErrUPolynomialAssertion
	}
	*d = 0
	if polynomialSourceDebug && b.count == 0 {
		return ErrUPolynomialAssertion
	}
	if b.count == 1 {
		return m.reset(r)
	}
	if e := m.setViewWithSourceContext(r, a, context); e != nil {
		return e
	}
	if a.count <= 1 {
		return nil
	}
	bn, e := m.read(b, b.count-1)
	if e != nil {
		return e
	}
	if polynomialSourceDebug {
		zero, e := m.modular.IsZero(bn)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	var local util.Int
	am := &local
	var frame *upolyFactorRemainderScope
	if owner := context.factorFrontendFrame(m.config.Integer); owner != nil {
		frame = &owner.remainder
		if frame.constructed || frame.executing || frame.invocation == ^uint64(0) {
			return ErrUPolynomialCleanupPending
		}
		*frame = upolyFactorRemainderScope{owner: owner, invocation: frame.invocation + 1, constructed: true, executing: true}
		am = &frame.am
		defer func() { frame.executing = false; cause = frame.finish(cause) }()
	} else {
		defer m.modular.Del(am)
	}
	for {
		if e = m.checkpoint(); e != nil {
			return e
		}
		n := r.size
		if n < b.count {
			return nil
		}
		delta := n - b.count
		field, e := m.modular.Field()
		if e != nil {
			return e
		}
		bn, e = m.read(b, b.count-1)
		if e != nil {
			return e
		}
		if field {
			if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Div(&m.values(r)[n-1], bn, am); e != nil {
				return e
			}
		} else {
			*d++
			if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Copy(am, &m.values(r)[n-1]); e != nil {
				return e
			}
			for i := uint32(0); i < n-1; i++ {
				bn, e = m.read(b, b.count-1)
				if e != nil {
					return e
				}
				if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).Mul(&m.values(r)[i], bn, &m.values(r)[i]); e != nil {
					return e
				}
			}
		}
		for i := uint32(0); i < b.count-1; i++ {
			x, e := m.read(b, i)
			if e != nil {
				return e
			}
			dst := &m.values(r)[i+delta]
			var child *util.IntegerAddMulInvocationReceipt
			if frame != nil {
				child = &frame.addMul
			}
			if e = m.modular.WithAllocationContext((context).utility(m.modular.Integers())).SubMul(dst, am, x, dst, child); e != nil {
				return e
			}
		}
		if e = m.setSize(r, n-1); e != nil {
			return e
		}
	}
}
func (m *UPolynomialManager) RemViews(a, b UPolynomialView, r UPolynomial, d *uint32) error {
	out, e := m.descriptor(r)
	if e != nil {
		return e
	}
	return m.remState(a, b, d, &out.state)
}
func (m *UPolynomialManager) Rem(a, b, r UPolynomial, d *uint32) error {
	av, e := m.wholeView(a)
	if e != nil {
		return e
	}
	bv, e := m.wholeView(b)
	if e != nil {
		return e
	}
	return m.RemViews(av, bv, r, d)
}
func (m *UPolynomialManager) signedRemState(a, b UPolynomialView, r *upolyVectorState) error {
	if polynomialSourceDebug && (m.alias(a, r) || m.alias(b, r)) {
		return ErrUPolynomialAssertion
	}
	var d uint32
	if e := m.remState(a, b, &d, r); e != nil {
		return e
	}
	negate := d%2 == 0
	if !negate && b.count > 0 {
		x, e := m.read(b, b.count-1)
		if e != nil {
			return e
		}
		negate, e = m.modular.IsPos(x)
		if e != nil {
			return e
		}
	}
	if negate {
		return m.NegViewInPlace(m.stateView(r))
	}
	return nil
}
func (m *UPolynomialManager) SignedRemViews(a, b UPolynomialView, r UPolynomial) error {
	out, e := m.descriptor(r)
	if e != nil {
		return e
	}
	return m.signedRemState(a, b, &out.state)
}
func (m *UPolynomialManager) SignedRem(a, b, r UPolynomial) error {
	av, e := m.wholeView(a)
	if e != nil {
		return e
	}
	bv, e := m.wholeView(b)
	if e != nil {
		return e
	}
	return m.SignedRemViews(av, bv, r)
}
