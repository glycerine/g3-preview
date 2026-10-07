// Source: src/math/polynomial/polynomial.cpp:6179-6232,6307-6344; src/util/vector.h.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// PolynomialVariablesView is a by-value borrowed source var const* pointer.
// It captures backing identity, not a vector owner, size, or copied values.
type PolynomialVariablesView struct {
	runtime *util.Runtime
	raw     util.Block
	offset  uint64
}

func BorrowPolynomialVariables(r *util.Runtime, b util.Block, offset uint64) PolynomialVariablesView {
	return PolynomialVariablesView{r, b, offset}
}
func (v PolynomialVariablesView) At(i uint32) (uint32, error) {
	if v.runtime == nil {
		return 0, util.ErrRuntimeOwner
	}
	o := v.offset + uint64(i)*4
	if o < v.offset {
		return 0, arena.ErrBounds
	}
	return v.runtime.ReadUint32(v.raw, o)
}
func (v PolynomialVariablesView) IsNull() bool { return v.raw == (util.Block{}) }
func variableEmbedded(d *PolynomialVariableDegrees, r *util.Runtime, complete bool) {
	*d = PolynomialVariableDegrees{runtime: r, constructed: complete}
	d.self = d
}
func variableGet(d *PolynomialVariableDegrees, x, def uint32) (uint32, error) {
	n, e := d.size()
	if e != nil {
		return 0, e
	}
	if x >= n {
		return def, nil
	}
	return d.runtime.ReadUint32(d.raw, 8+uint64(x)*4)
}
func variableAt(d *PolynomialVariableDegrees, x uint32) (uint32, error) {
	if polynomialSourceDebug {
		n, e := d.size()
		if e != nil {
			return 0, e
		}
		if x >= n {
			return 0, ErrPolynomialAssertion
		}
	}
	return d.runtime.ReadUint32(d.raw, 8+uint64(x)*4)
}
func variablePut(d *PolynomialVariableDegrees, x, v uint32) error {
	if polynomialSourceDebug {
		n, e := d.size()
		if e != nil {
			return e
		}
		if x >= n {
			return ErrPolynomialAssertion
		}
	}
	return d.runtime.WriteUint32(d.raw, 8+uint64(x)*4, v)
}

// The filled resize overload grows through ALL allocations with old SIZE,
// publishes requested SIZE once, then constructs its unsigned values ascending.
func variableResizeFilled(d *PolynomialVariableDegrees, n, fill uint32) error {
	old, e := d.size()
	if e != nil {
		return e
	}
	if n <= old {
		if d.raw == (util.Block{}) {
			return nil
		}
		return d.runtime.WriteUint32(d.raw, 4, n)
	}
	for {
		cap, e := d.capacity()
		if e != nil {
			return e
		}
		if n <= cap {
			break
		}
		if e = d.expand(); e != nil {
			return e
		}
	}
	if e = d.runtime.WriteUint32(d.raw, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = d.runtime.WriteUint32(d.raw, 8+uint64(i)*4, fill); e != nil {
			d.interrupted = true
			return e
		}
	}
	return nil
}
func variableSetFilled(d *PolynomialVariableDegrees, x, value, fill uint32) error {
	if d.interrupted || d.pending != (util.Block{}) {
		return ErrPolynomialPending
	}
	n, e := d.size()
	if e != nil {
		return e
	}
	if x >= n {
		if e = variableResizeFilled(d, x+1, fill); e != nil {
			return e
		}
	}
	return d.runtime.WriteUint32(d.raw, 8+uint64(x)*4, value)
}
func variablePush(d *PolynomialVariableDegrees, value uint32) error {
	if d.interrupted || d.pending != (util.Block{}) {
		return ErrPolynomialPending
	}
	if d.raw == (util.Block{}) {
		if e := d.expand(); e != nil {
			return e
		}
	} else {
		n, e := d.size()
		if e != nil {
			return e
		}
		cap, e := d.capacity()
		if e != nil {
			return e
		}
		if n == cap {
			if e = d.expand(); e != nil {
				return e
			}
		}
	}
	n, e := d.size()
	if e != nil {
		return e
	}
	if e = d.runtime.WriteUint32(d.raw, 8+uint64(n)*4, value); e != nil {
		return e
	}
	n, e = d.size()
	if e != nil {
		return e
	}
	return d.runtime.WriteUint32(d.raw, 4, n+1)
}
func variableReset(d *PolynomialVariableDegrees) error {
	if d.raw == (util.Block{}) {
		return nil
	}
	return d.runtime.WriteUint32(d.raw, 4, 0)
}
func variableView(d *PolynomialVariableDegrees) PolynomialVariablesView {
	offset := uint64(0)
	if d.raw != (util.Block{}) {
		offset = 8
	}
	return BorrowPolynomialVariables(d.runtime, d.raw, offset)
}

// Each member is a real uint-vector state embedded in one charged stable cold
// owner. Its header charge is zero; no per-vector Go owner allocation occurs.
type PolynomialMaxDegrees struct {
	self                                                *PolynomialMaxDegrees
	core                                                *MultivariatePolynomialCore
	runtime                                             *util.Runtime
	maxDegree, variables                                PolynomialVariableDegrees
	header                                              uint64
	constructed, closing, finalizing, finalized, closed bool
	sourceStage                                         uint8
	bodyCause, sourceCause                              error
}

func newPolynomialMaxDegrees(m *MultivariatePolynomialCore, r *util.Runtime, complete bool) (*PolynomialMaxDegrees, error) {
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialMaxDegrees{}).Size())
	if e := r.Budget().Reserve(h); e != nil {
		return nil, e
	}
	d := &PolynomialMaxDegrees{core: m, runtime: r, header: h, constructed: complete}
	d.self = d
	variableEmbedded(&d.maxDegree, r, complete)
	variableEmbedded(&d.variables, r, complete)
	return d, nil
}
func NewPolynomialMaxDegrees(m *MultivariatePolynomialCore) (*PolynomialMaxDegrees, error) {
	if m == nil {
		return nil, ErrPolynomialOwner
	}
	if e := m.check(); e != nil {
		return nil, e
	}
	return newPolynomialMaxDegrees(m, m.memory, true)
}
func (d *PolynomialMaxDegrees) identity() error {
	if d == nil || d.self != d {
		return ErrPolynomialOwner
	}
	if d.closed {
		return arena.ErrClosed
	}
	return d.runtime.CheckOpen()
}
func (d *PolynomialMaxDegrees) open() error {
	if e := d.identity(); e != nil {
		return e
	}
	if !d.constructed {
		return ErrPolynomialPending
	}
	if d.closing || d.finalizing {
		return arena.ErrClosed
	}
	return nil
}
func (d *PolynomialMaxDegrees) Init(p Polynomial) error {
	if e := d.open(); e != nil {
		return e
	}
	m := d.core
	if e := m.check(); e != nil {
		return e
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		term, e := m.monomialCore(p, i)
		if e != nil {
			return e
		}
		count, e := m.monomials.sizeCore(term)
		if e != nil {
			return e
		}
		for j := uint32(0); j < count; j++ {
			x, e := m.monomials.varCore(term, j)
			if e != nil {
				return e
			}
			k, e := m.monomials.degreeAtCore(term, j)
			if e != nil {
				return e
			}
			old, e := variableGet(&d.maxDegree, x, 0)
			if e != nil {
				return e
			}
			if k > old {
				if old == 0 {
					if e = variablePush(&d.variables, x); e != nil {
						d.bodyCause = e
						return e
					}
				}
				k, e = m.monomials.degreeAtCore(term, j)
				if e != nil {
					d.bodyCause = e
					return e
				}
				if e = variableSetFilled(&d.maxDegree, x, k, 0); e != nil {
					d.bodyCause = e
					return e
				}
			}
		}
	}
	return nil
}
func (d *PolynomialMaxDegrees) Reset() error {
	if e := d.open(); e != nil {
		return e
	}
	n, e := d.variables.size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		x, e := variableAt(&d.variables, i)
		if e != nil {
			return e
		}
		if e = variablePut(&d.maxDegree, x, 0); e != nil {
			return e
		}
	}
	return variableReset(&d.variables)
}
func (d *PolynomialMaxDegrees) Degree(x uint32) (uint32, error) {
	if e := d.open(); e != nil {
		return 0, e
	}
	return variableGet(&d.maxDegree, x, 0)
}
func (d *PolynomialMaxDegrees) NumVariables() (uint32, error) {
	if e := d.open(); e != nil {
		return 0, e
	}
	return d.variables.size()
}
func (d *PolynomialMaxDegrees) Vars() (PolynomialVariablesView, error) {
	if e := d.open(); e != nil {
		return PolynomialVariablesView{}, e
	}
	return variableView(&d.variables), nil
}

type PolynomialVariablePositions struct {
	self                                                *PolynomialVariablePositions
	runtime                                             *util.Runtime
	positions                                           PolynomialVariableDegrees
	header                                              uint64
	constructed, closing, finalizing, finalized, closed bool
	bodyCause, sourceCause                              error
}

func newPolynomialVariablePositions(r *util.Runtime, complete bool) (*PolynomialVariablePositions, error) {
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialVariablePositions{}).Size())
	if e := r.Budget().Reserve(h); e != nil {
		return nil, e
	}
	d := &PolynomialVariablePositions{runtime: r, header: h, constructed: complete}
	d.self = d
	variableEmbedded(&d.positions, r, complete)
	return d, nil
}
func NewPolynomialVariablePositions(r *util.Runtime) (*PolynomialVariablePositions, error) {
	return newPolynomialVariablePositions(r, true)
}
func (d *PolynomialVariablePositions) identity() error {
	if d == nil || d.self != d {
		return ErrPolynomialOwner
	}
	if d.closed {
		return arena.ErrClosed
	}
	return d.runtime.CheckOpen()
}
func (d *PolynomialVariablePositions) open() error {
	if e := d.identity(); e != nil {
		return e
	}
	if !d.constructed {
		return ErrPolynomialPending
	}
	if d.closing || d.finalizing {
		return arena.ErrClosed
	}
	return nil
}
func (d *PolynomialVariablePositions) Init(n uint32, xs PolynomialVariablesView) error {
	if e := d.open(); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if polynomialSourceDebug {
			x, e := xs.At(i)
			if e != nil {
				return e
			}
			old, e := variableGet(&d.positions, x, ^uint32(0))
			if e != nil {
				return e
			}
			if old != ^uint32(0) {
				return ErrPolynomialAssertion
			}
		}
		x, e := xs.At(i)
		if e != nil {
			return e
		}
		if e = variableSetFilled(&d.positions, x, i, ^uint32(0)); e != nil {
			d.bodyCause = e
			return e
		}
	}
	return nil
}
func (d *PolynomialVariablePositions) Reset(n uint32, xs PolynomialVariablesView) error {
	if e := d.open(); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if polynomialSourceDebug {
			x, e := xs.At(i)
			if e != nil {
				return e
			}
			old, e := variableGet(&d.positions, x, ^uint32(0))
			if e != nil {
				return e
			}
			if old == ^uint32(0) {
				return ErrPolynomialAssertion
			}
		}
		x, e := xs.At(i)
		if e != nil {
			return e
		}
		if e = variablePut(&d.positions, x, ^uint32(0)); e != nil {
			return e
		}
	}
	return nil
}
func (d *PolynomialVariablePositions) Position(x uint32) (uint32, error) {
	if e := d.open(); e != nil {
		return 0, e
	}
	return variableGet(&d.positions, x, ^uint32(0))
}
