// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Concrete finite-field square-free decomposition and Berlekamp source bodies.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
)

// ErrUPolynomialNontermination guards a source uint32 loop at its wrap boundary.
var ErrUPolynomialDomain = errors.New("mpnum: polynomial source domain")
var ErrUPolynomialNontermination = errors.New("upolynomial source loop does not terminate")

func (m *UPolynomialManager) factorPrime() (uint32, error) {
	p, e := m.modular.borrowedModulus()
	if e != nil {
		return 0, e
	}
	ok, e := m.config.Integer.IsUint64(p)
	if e != nil {
		return 0, e
	}
	if !ok {
		return 0, ErrUPolynomialDomain
	}
	n, e := m.config.Integer.Uint64(p)
	if e != nil {
		return 0, e
	}
	if uint64(uint32(n)) != n {
		return 0, ErrUPolynomialDomain
	}
	return uint32(n), nil
}
func (m *UPolynomialManager) checkFactors(fs *UPolynomialFactors) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := fs.check(); e != nil {
		return e
	}
	if fs.manager != m {
		return ErrUPolynomialOwner
	}
	return nil
}
func (m *UPolynomialManager) scopedPush(s *upolyVectorState, value *Int) error {
	out, e := m.pushDefault(s)
	if e != nil {
		return e
	}
	return m.modular.Copy(out, value)
}

// The source matrix is lexical: its slice headers stay on the caller stack;
// both geometrically grown pivot arrays and the numeral backing are individually charged.
type upolyBerlekampMatrix struct {
	m             *UPolynomialManager
	values        upolyVectorState
	size, nullRow uint32
	column, row   []int32
}

func (q *upolyBerlekampMatrix) close() {
	m := q.m
	// Source destructor body deletes matrix numerals ascending, then implicit
	// member destructors release row, column and matrix physical storage.
	m.reset(&q.values)
	m.config.Integer.budget.Release(uint64(cap(q.row)) * 4)
	q.row = nil
	m.config.Integer.budget.Release(uint64(cap(q.column)) * 4)
	q.column = nil
	m.freeBacking(&q.values)
}
func (q *upolyBerlekampMatrix) get(i, j uint32) (*Int, error) {
	if modularSourceDebug && (i >= q.size || j >= q.size) {
		return nil, ErrUPolynomialAssertion
	}
	// Typed backing/domain adapter: avoid a wrapped matrix index aliasing cells.
	index := uint64(i)*uint64(q.size) + uint64(j)
	if index > math.MaxUint32 {
		return nil, arena.ErrBounds
	}
	return q.m.read(q.m.stateView(&q.values), uint32(index))
}
func (q *upolyBerlekampMatrix) init(m *UPolynomialManager, f UPolynomialView) error {
	q.m = m
	q.size = upolyViewDegree(f)
	q.nullRow = 1
	n := q.size
	if e := upolyFactorInt32Resize(&q.column, n, -1, m.config.Integer.budget); e != nil {
		return e
	}
	if e := upolyFactorInt32Resize(&q.row, n, -1, m.config.Integer.budget); e != nil {
		return e
	}
	p, e := m.factorPrime()
	if e != nil {
		return e
	}

	v, e := m.pushDefault(&q.values)
	if e != nil {
		return e
	}
	*v = NewInt(1)
	for j := uint32(0); j < n; j++ {
		if _, e = m.pushDefault(&q.values); e != nil {
			return e
		}
	}
	var tmp Int
	defer m.modular.Del(&tmp)
	row, previous := uint32(0), uint32(0)
	for k := uint32(1); ; k++ {
		if p == 0 {
			return ErrUPolynomialDomain
		} // actual undefined modulo read site
		if k%p == 1 {
			row++
			if row >= n {
				break
			}
			for j := uint32(0); j < n; j++ {
				if _, e = m.pushDefault(&q.values); e != nil {
					return e
				}
			}
		}
		top, e := q.get(previous, n-1)
		if e != nil {
			return e
		}
		if e = m.modular.Copy(&tmp, top); e != nil {
			return e
		}
		for j := n - 1; j > 0; j-- {
			old, e := q.get(previous, j-1)
			if e != nil {
				return e
			}
			fj, e := m.read(f, j)
			if e != nil {
				return e
			}
			out, e := q.get(row, j)
			if e != nil {
				return e
			}
			if e = m.modular.SubMul(old, &tmp, fj, out); e != nil {
				return e
			}
		}
		f0, e := m.read(f, 0)
		if e != nil {
			return e
		}
		out, e := q.get(row, 0)
		if e != nil {
			return e
		}
		if e = m.modular.Mul(f0, &tmp, out); e != nil {
			return e
		}
		if e = m.modular.Neg(out); e != nil {
			return e
		}
		previous = row
		if k == math.MaxUint32 {
			return ErrUPolynomialNontermination
		}
	}
	for i := uint32(0); i < n; i++ {
		a, e := q.get(i, i)
		if e != nil {
			return e
		}
		if e = m.modular.Dec(a); e != nil {
			return e
		}
	}
	return nil
}
func (q *upolyBerlekampMatrix) diagonalize() (uint32, error) {
	m := q.m
	var multiplier Int
	defer m.modular.Del(&multiplier)
	nullRank := uint32(0)
	for i := uint32(0); i < q.size; i++ {
		found := false
		for j := uint32(0); j < q.size; j++ {
			if q.column[j] >= 0 {
				continue
			}
			a, e := q.get(i, j)
			if e != nil {
				return 0, e
			}
			zero, e := m.modular.IsZero(a)
			if e != nil {
				return 0, e
			}
			if zero {
				continue
			}
			found = true
			q.column[j] = int32(i)
			q.row[i] = int32(j)
			if e = m.modular.Copy(&multiplier, a); e != nil {
				return 0, e
			}
			if e = m.modular.Inv(&multiplier); e != nil {
				return 0, e
			}
			if e = m.modular.Neg(&multiplier); e != nil {
				return 0, e
			}
			for k := q.nullRow; k < q.size; k++ {
				a, e := q.get(k, j)
				if e != nil {
					return 0, e
				}
				if e = m.modular.Mul(a, &multiplier, a); e != nil {
					return 0, e
				}
			}
			for other := uint32(0); other < q.size; other++ {
				if other == j {
					continue
				}
				a, e := q.get(i, other)
				if e != nil {
					return 0, e
				}
				if e = m.modular.Copy(&multiplier, a); e != nil {
					return 0, e
				}
				for k := q.nullRow; k < q.size; k++ {
					a, e := q.get(k, other)
					if e != nil {
						return 0, e
					}
					pivot, e := q.get(k, j)
					if e != nil {
						return 0, e
					}
					if e = m.modular.AddMul(a, &multiplier, pivot, a); e != nil {
						return 0, e
					}
				}
			}
		}
		if !found {
			nullRank++
		}
	}
	return nullRank, nil
}
func (q *upolyBerlekampMatrix) next(v *upolyVectorState) (bool, error) {
	m := q.m
	if modularSourceDebug && v.size > q.size {
		return false, ErrUPolynomialAssertion
	}
	// scoped_numeral_vector::resize, default zero cells; logical shrink deletes.
	if q.size <= v.size {
		if e := m.shrink(v, q.size); e != nil {
			return false, e
		}
	}
	if e := m.reserve(v, q.size); e != nil {
		return false, e
	}
	for ; q.nullRow < q.size; q.nullRow++ {
		if q.row[q.nullRow] >= 0 {
			continue
		}
		for j := uint32(0); j < q.size; j++ {
			out, e := m.read(m.stateView(v), j)
			if e != nil {
				return false, e
			}
			if q.row[j] >= 0 {
				a, e := q.get(q.nullRow, uint32(q.row[j]))
				if e != nil {
					return false, e
				}
				if e = m.modular.Copy(out, a); e != nil {
					return false, e
				}
			} else {
				n := int32(0)
				if j == q.nullRow {
					n = 1
				}
				if e = m.modular.SetInt32(out, n); e != nil {
					return false, e
				}
			}
		}
		q.nullRow++
		return true, nil
	}
	return false, nil
}
func (m *UPolynomialManager) FactorZPSquareFreeBerlekamp(f UPolynomialView, fs *UPolynomialFactors, randomized bool) (bool, error) {
	if e := m.checkFactors(fs); e != nil {
		return false, e
	}
	return m.factorZPBerlekamp(f, fs, randomized)
}
func (m *UPolynomialManager) FactorZPSquareFree(f UPolynomialView, fs *UPolynomialFactors) (bool, error) {
	return m.FactorZPSquareFreeBerlekamp(f, fs, false)
}
func (m *UPolynomialManager) factorZPBerlekamp(f UPolynomialView, fs *UPolynomialFactors, randomized bool) (bool, error) {
	// The actual source accepts randomized but implements the same deterministic
	// splitting body for either value; no RNG is substituted.
	if modularSourceDebug && upolyViewDegree(f) <= 1 {
		return false, ErrUPolynomialAssertion
	}
	p, e := m.factorPrime()
	if e != nil {
		return false, e
	}
	if modularSourceDebug {
		top, e := m.read(f, f.count-1)
		if e != nil {
			return false, e
		}
		one, e := m.modular.IsOne(top)
		if e != nil {
			return false, e
		}
		if !one {
			return false, ErrUPolynomialAssertion
		}
	}
	var matrix upolyBerlekampMatrix
	matrix.m = m
	defer matrix.close()
	if e = matrix.init(m, f); e != nil {
		return false, e
	}
	first := fs.count
	if e = fs.PushCopy(f, 1); e != nil {
		return false, e
	}
	rank, e := matrix.diagonalize()
	if e != nil {
		return false, e
	}
	if rank == 1 {
		return false, nil
	}
	var gcd, div, v upolyVectorState
	defer m.dropLocalVector(&gcd)
	defer m.dropLocalVector(&div)
	defer m.dropLocalVector(&v)
	for {
		has, e := matrix.next(&v)
		if e != nil {
			return false, e
		}
		if !has {
			break
		}
		if e = m.trim(&v); e != nil {
			return false, e
		}
		end := fs.count
		for i := first; i < end; i++ {
			if fs.factors[i].size == 2 {
				continue
			}
			for s := uint32(0); s < p; s++ {
				v0, e := m.read(m.stateView(&v), 0)
				if e != nil {
					return false, e
				}
				if e = m.modular.Dec(v0); e != nil {
					return false, e
				}
				current := &fs.factors[i]
				if e = m.gcd(m.stateView(&v), m.stateView(current), &gcd); e != nil {
					return false, e
				}
				if gcd.size != 1 && gcd.size != current.size {
					if e = m.divState(m.stateView(current), m.stateView(&gcd), &div); e != nil {
						return false, e
					}
					degree, e := fs.multiplicity(i)
					if e != nil {
						return false, e
					}
					fs.totalDegree -= upolyStateDegree(current) * degree
					fs.totalDegree += upolyStateDegree(&div) * degree
					*current, div = div, *current
					if e = fs.PushCopy(m.stateView(&gcd), 1); e != nil {
						return false, e
					}
				}
				if fs.count-first == rank {
					return true, nil
				}
			}
		}
	}
	if modularSourceDebug {
		return false, ErrUPolynomialAssertion
	}
	return true, nil
}
func (m *UPolynomialManager) SquareFreeFactorZP(f UPolynomialView, fs *UPolynomialFactors) error {
	if e := m.checkFactors(fs); e != nil {
		return e
	}
	return m.squareFreeFactorZP(f, fs)
}
func (m *UPolynomialManager) squareFreeFactorZP(f UPolynomialView, fs *UPolynomialFactors) error {
	p, e := m.factorPrime()
	if e != nil {
		return e
	}
	if p == 0 {
		return ErrUPolynomialDomain
	}
	var divTmp, t0, t0d, t, v, w, a upolyVectorState
	defer m.dropLocalVector(&divTmp)
	defer m.dropLocalVector(&t0)
	if modularSourceDebug && f.count <= 1 {
		return ErrUPolynomialAssertion
	}
	if e = m.setView(&t0, f); e != nil {
		return e
	}
	var constant Int
	defer m.modular.Del(&constant)
	// mk_monic(lc) has its own scoped inverse destroyed before output constant.
	if e = m.factorMonic(m.stateView(&t0), &constant); e != nil {
		return e
	}
	if e = fs.SetConstant(&constant); e != nil {
		return e
	}
	exponent := uint32(1)
	defer m.dropLocalVector(&t0d)
	defer m.dropLocalVector(&t)
	defer m.dropLocalVector(&v)
	defer m.dropLocalVector(&w)
	defer m.dropLocalVector(&a)
	for t0.size > 1 {
		k := uint32(0)
		if e = m.derivativeState(m.stateView(&t0), &t0d); e != nil {
			return e
		}
		if e = m.gcd(m.stateView(&t0), m.stateView(&t0d), &t); e != nil {
			return e
		}
		if e = m.divState(m.stateView(&t0), m.stateView(&t), &v); e != nil {
			return e
		}
		for v.size > 1 {
			k++
			if k%p == 0 {
				k++
				if e = m.divState(m.stateView(&t), m.stateView(&v), &t); e != nil {
					return e
				}
			}
			if e = m.gcd(m.stateView(&t), m.stateView(&v), &w); e != nil {
				return e
			}
			if e = m.divState(m.stateView(&v), m.stateView(&w), &a); e != nil {
				return e
			}
			v, w = w, v
			if e = m.divState(m.stateView(&t), m.stateView(&v), &t); e != nil {
				return e
			}
			if a.size > 1 {
				if e = fs.PushCopy(m.stateView(&a), exponent*k); e != nil {
					return e
				}
			}
		}
		exponent *= p
		if e = m.reset(&t0); e != nil {
			return e
		}
		for deg := uint32(0); deg < t.size; {
			zero := NewInt(0)
			if e = m.scopedPush(&t0, &zero); e != nil {
				return e
			}
			input, e := m.read(m.stateView(&t), deg)
			if e != nil {
				return e
			}
			out, e := m.read(m.stateView(&t0), t0.size-1)
			if e != nil {
				return e
			}
			if e = m.modular.Copy(out, input); e != nil {
				return e
			}
			if deg > math.MaxUint32-p {
				return ErrUPolynomialNontermination
			}
			deg += p
		}
	}
	return nil
}
func (m *UPolynomialManager) factorMonic(v UPolynomialView, leading *Int) error {
	var inverse Int
	defer m.modular.Del(&inverse)
	return m.monic(v, leading, &inverse)
}
func (m *UPolynomialManager) FactorZP(f UPolynomialView, fs *UPolynomialFactors) (bool, error) {
	if e := m.checkFactors(fs); e != nil {
		return false, e
	}
	sq, e := m.NewFactors()
	if e != nil {
		return false, e
	}
	defer sq.Close()
	if e = m.squareFreeFactorZP(f, sq); e != nil {
		return false, e
	}
	for i := uint32(0); i < sq.count; i++ {
		start := fs.count
		v := m.stateView(&sq.factors[i])
		if upolyViewDegree(v) > 1 {
			if _, e = m.factorZPBerlekamp(v, fs, false); e != nil {
				return false, e
			}
			for j := start; j < fs.count; j++ {
				k, e := fs.multiplicity(j)
				if e != nil {
					return false, e
				}
				mult, e := sq.multiplicity(i)
				if e != nil {
					return false, e
				}
				if e = fs.SetMultiplicity(j, mult*k); e != nil {
					return false, e
				}
			}
		} else {
			k, e := sq.multiplicity(i)
			if e != nil {
				return false, e
			}
			if e = fs.PushCopy(v, k); e != nil {
				return false, e
			}
		}
	}
	if e = fs.SetConstant(&sq.constant); e != nil {
		return false, e
	}
	return fs.totalFactors > 1, nil
}
