// Licensed under the MIT license.
// algebraic_numbers.cpp:541–684, real univariate factor/isolation pipeline.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

func (m *AlgebraicRootCore) checkpoint() error {
	if !m.config.Checkpoint() {
		return ErrUPolynomialCanceled
	}
	return m.config.Cooperation()
}
func (m *AlgebraicRootCore) ensureIsolation() error {
	var e error
	if m.factors == nil {
		m.factors, e = m.config.Polynomials.NewFactors()
		if e != nil {
			return e
		}
	}
	if m.roots == nil {
		m.roots, e = numericBinaryRationalManager_NewVector(m.config.Binary)
		if e != nil {
			return e
		}
	}
	if m.lowers == nil {
		m.lowers, e = numericBinaryRationalManager_NewVector(m.config.Binary)
		if e != nil {
			return e
		}
	}
	if m.uppers == nil {
		m.uppers, e = numericBinaryRationalManager_NewVector(m.config.Binary)
		if e != nil {
			return e
		}
	}
	return nil
}
func (m *AlgebraicRootCore) factorPolynomial(v UPolynomialView) (bool, error) {
	if m.factor {
		return m.config.Factors.Factor(v, m.factors, m.factorParams)
	}
	if m.tmp3.slot == 0 {
		var e error
		m.tmp3, e = m.config.Polynomials.NewPolynomial()
		if e != nil {
			return false, e
		}
	}
	if e := m.config.Polynomials.SquareFreeView(v, m.tmp3); e != nil {
		return false, e
	}
	w, e := m.config.Polynomials.wholeView(m.tmp3)
	if e != nil {
		return false, e
	}
	return false, m.factors.PushCopy(w, 1)
}
func (m *AlgebraicRootCore) pushNew(v *AlgebraicVector, a *Algebraic) error {
	if e := v.pushTake(a); e != nil {
		if a.slot != 0 {
			m.pending = *a
			l := &m.locations[a.slot-1]
			if l.published {
				l.published = false
				m.callers--
			}
			*a = Algebraic{}
		}
		return e
	}
	return nil
}
func (m *AlgebraicRootCore) IsolateRoots(v UPolynomialView, out *AlgebraicVector) (err error) {
	if err = m.check(); err != nil {
		return err
	}
	if out == nil || out.core != m {
		return ErrAlgebraicOwner
	}
	if err = out.check(); err != nil {
		return err
	}
	if v.count == 0 {
		return nil
	}
	if m.pending.slot != 0 {
		return ErrAlgebraicPending
	}
	if err = m.ensureIsolation(); err != nil {
		return err
	}
	if err = m.factors.Reset(); err != nil {
		return err
	}
	zero, err := m.config.Polynomials.HasZeroRoots(v)
	if err != nil {
		return err
	}
	if zero {
		a := Algebraic{}
		if err = out.pushTake(&a); err != nil {
			return err
		}
		if m.tmp2.slot == 0 {
			m.tmp2, err = m.config.Polynomials.NewPolynomial()
			if err != nil {
				return err
			}
		}
		if err = m.config.Polynomials.RemoveZeroRoots(v, m.tmp2); err != nil {
			return err
		}
		v, err = m.config.Polynomials.wholeView(m.tmp2)
		if err != nil {
			return err
		}
	}
	full, err := m.factorPolynomial(v)
	if err != nil {
		return err
	}
	n, err := m.factors.DistinctFactors()
	if err != nil {
		return err
	}
	for i := uint32(0); i < n; i++ {
		f, e := m.factors.FactorView(i)
		if e != nil {
			return e
		}
		if f.count <= 1 {
			continue
		}
		if err = m.beginQ(); err != nil {
			return err
		}
		if f.count == 2 {
			x, e := m.config.Polynomials.read(f, 0)
			if e != nil {
				return m.finishQ(e)
			}
			y, e := m.config.Polynomials.read(f, 1)
			if e != nil {
				return m.finishQ(e)
			}
			if err = m.config.Rationals.SetInteger(&m.localQ, x); err != nil {
				return m.finishQ(err)
			}
			if err = m.config.Rationals.DivInt(&m.localQ, y, &m.localQ); err != nil {
				return m.finishQ(err)
			}
			if err = m.config.Rationals.Neg(&m.localQ); err != nil {
				return m.finishQ(err)
			}
			var a Algebraic
			if err = m.SetTake(&a, &m.localQ); err != nil {
				return m.finishQ(err)
			}
			err = m.pushNew(out, &a)
			if err = m.finishQ(err); err != nil {
				return err
			}
			continue
		}
		if polynomialSourceDebug && (m.roots.state.size != 0 || m.lowers.state.size != 0 || m.uppers.state.size != 0) {
			return m.finishQ(ErrUPolynomialAssertion)
		}
		if err = m.config.Polynomials.SquareFreeIsolateRoots(f, m.config.Binary, m.roots, m.lowers, m.uppers); err != nil {
			return m.finishQ(err)
		}
		for j := uint32(0); j < m.roots.state.size; j++ {
			b := &m.roots.state.values[j]
			if err = m.config.Binary.ToRational(m.config.Rationals, b, &m.localQ); err != nil {
				return m.finishQ(err)
			}
			var a Algebraic
			if err = m.SetTake(&a, &m.localQ); err != nil {
				return m.finishQ(err)
			}
			if err = m.pushNew(out, &a); err != nil {
				return m.finishQ(err)
			}
		}
		if polynomialSourceDebug && m.lowers.state.size != m.uppers.state.size {
			return m.finishQ(ErrAlgebraicAssertion)
		}
		for j := uint32(0); j < m.lowers.state.size; j++ {
			if j >= m.uppers.state.size {
				return m.finishQ(arena.ErrBounds)
			}
			lo, hi := &m.lowers.state.values[j], &m.uppers.state.values[j]
			irr, e := m.config.Polynomials.IsolatingToRefinable(f, m.config.Binary, lo, hi)
			if e != nil {
				return m.finishQ(e)
			}
			var a Algebraic
			if irr {
				err = m.SetRoot(&a, f, lo, hi, full)
			} else {
				err = m.config.Binary.ToRational(m.config.Rationals, lo, &m.localQ)
				if err == nil {
					err = m.SetTake(&a, &m.localQ)
				}
			}
			if err != nil {
				return m.finishQ(err)
			}
			if err = m.pushNew(out, &a); err != nil {
				return m.finishQ(err)
			}
		}
		if err = m.roots.Reset(); err != nil {
			return m.finishQ(err)
		}
		if err = m.lowers.Reset(); err != nil {
			return m.finishQ(err)
		}
		if err = m.uppers.Reset(); err != nil {
			return m.finishQ(err)
		}
		if err = m.finishQ(nil); err != nil {
			return err
		}
	}
	// Concrete heap sort is the documented external std::sort backend adapter;
	// it sorts the entire append-only output without extra heap scratch.
	for start := int(out.size)/2 - 1; start >= 0; start-- {
		if err = m.rootSift(out, uint32(start), out.size); err != nil {
			return err
		}
	}
	for end := out.size; end > 1; end-- {
		out.values[0], out.values[end-1] = out.values[end-1], out.values[0]
		if err = m.rootSift(out, 0, end-1); err != nil {
			return err
		}
	}
	return nil
}
func (m *AlgebraicRootCore) rootSift(v *AlgebraicVector, root, end uint32) error {
	for uint64(root)*2+1 < uint64(end) {
		child := root*2 + 1
		if child+1 < end {
			d, e := m.Compare(&v.values[child], &v.values[child+1])
			if e != nil {
				return e
			}
			if d < 0 {
				child++
			}
		}
		d, e := m.Compare(&v.values[root], &v.values[child])
		if e != nil {
			return e
		}
		if d >= 0 {
			return nil
		}
		v.values[root], v.values[child] = v.values[child], v.values[root]
		root = child
	}
	return nil
}
func (m *AlgebraicRootCore) MkRoot(v UPolynomialView, index uint32, out *Algebraic) (err error) {
	if e := m.check(); e != nil {
		return e
	}
	if index == 0 {
		return errors.New("invalid root object, root index must be greater than 0")
	}
	if v.count == 0 {
		return errors.New("invalid root object, polynomial must not be the zero polynomial")
	}
	if m.rootSelection != nil {
		return ErrAlgebraicPending
	}
	r, err := (m).NewVector()
	if err != nil {
		return err
	}
	m.rootSelection = r
	defer func() {
		e := r.Close()
		if e == nil {
			m.rootSelection = nil
		} else {
			err = errors.Join(err, e)
		}
	}()
	if err = m.IsolateRoots(v, r); err != nil {
		return err
	}
	if index > r.size {
		return errors.New("invalid root object, polynomial does have sufficient roots")
	}
	return m.Copy(out, &r.values[index-1])
}

// CleanupIsolation retries only explicitly retained source lexical/vector roots.
func (m *AlgebraicRootCore) CleanupIsolation() error {
	if m.rootSelection != nil {
		if e := m.rootSelection.Close(); e != nil {
			return e
		}
		m.rootSelection = nil
	}
	if m.roots != nil {
		if e := m.roots.Reset(); e != nil {
			return e
		}
	}
	if m.lowers != nil {
		if e := m.lowers.Reset(); e != nil {
			return e
		}
	}
	if m.uppers != nil {
		if e := m.uppers.Reset(); e != nil {
			return e
		}
	}
	return nil
}

var _ = arena.ErrBounds
