// Source: src/math/polynomial/polynomial.cpp:2807-2952.
package polynomial

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type polynomialNewtonScope struct {
	refsConstructed, cConstructed bool
	kind, stage                   uint8
	product, aux, c               util.Int
	temp, auxPoly, u              polynomialReference
	body, cleanup                 error
}

// PolynomialNewtonInterpolator is the real modular incremental Newton algorithm.
// Its heap owner is a charged Go adapter for the source stack or heap object.
type PolynomialNewtonInterpolator struct {
	self                          *PolynomialNewtonInterpolator
	owner                         *MultivariatePolynomialCore
	inputs, invs                  polynomialVector
	vs                            polynomialReferenceVector
	scope                         polynomialNewtonScope
	resetCause                    error
	header                        uint64
	resetStage, closeStage        uint8
	inputDrain, invDrain          uint32
	constructing, closing, closed bool
}

func NewPolynomialNewtonInterpolator(m *MultivariatePolynomialCore) (*PolynomialNewtonInterpolator, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialNewtonInterpolator{}).Size())
	if e := m.integer.Budget().Reserve(h); e != nil {
		return nil, e
	}
	n := &PolynomialNewtonInterpolator{owner: m, header: h, constructing: true}
	n.self = n
	n.inputs.width = 12
	n.invs.width = 12
	n.vs.values.width = 16
	zero := util.Int{}
	if e := n.pushNumeral(&n.invs, &zero); e != nil {
		return n, e
	}
	n.constructing = false
	return n, nil
}
func (n *PolynomialNewtonInterpolator) open() error {
	if n == nil || n.self != n {
		return ErrPolynomialOwner
	}
	if n.closed || n.closing || n.constructing {
		return arena.ErrClosed
	}
	if n.scope.kind != 0 || n.resetStage != 0 {
		return ErrPolynomialPending
	}
	return n.owner.check()
}
func (n *PolynomialNewtonInterpolator) NumSamplePoints() (uint32, error) {
	if n == nil || n.self != n {
		return 0, ErrPolynomialOwner
	}
	if n.closed {
		return 0, arena.ErrClosed
	}
	return n.inputs.size(n.owner)
}

type PolynomialNewtonInputs struct{ owner *PolynomialNewtonInterpolator }

func (n *PolynomialNewtonInterpolator) Inputs() PolynomialNewtonInputs {
	return PolynomialNewtonInputs{owner: n}
}
func (v PolynomialNewtonInputs) Size() (uint32, error)          { return v.owner.NumSamplePoints() }
func (v PolynomialNewtonInputs) Get(i uint32) (util.Int, error) { return v.owner.Input(i) }

func (n *PolynomialNewtonInterpolator) Input(i uint32) (util.Int, error) {
	if n == nil || n.self != n {
		return util.Int{}, ErrPolynomialOwner
	}
	if n.closed || n.closing || n.constructing {
		return util.Int{}, arena.ErrClosed
	}
	if e := n.owner.check(); e != nil {
		return util.Int{}, e
	}
	s, e := n.inputs.size(n.owner)
	if e != nil {
		return util.Int{}, e
	}
	if polynomialSourceDebug && i >= s {
		return util.Int{}, ErrPolynomialAssertion
	}
	if i >= n.inputs.initialized {
		return util.Int{}, arena.ErrBounds
	}
	return n.inputs.coefficient(n.owner, i)
}
func (n *PolynomialNewtonInterpolator) pushNumeral(v *polynomialVector, a *util.Int) error {
	if e := v.appendCoefficient(n.owner, util.Int{}); e != nil {
		return e
	}
	s, e := v.size(n.owner)
	if e != nil {
		return e
	}
	b, e := v.coefficient(n.owner, s-1)
	if e != nil {
		return e
	}
	e = n.owner.modular.Copy(&b, a)
	if w := v.putCoefficient(n.owner, s-1, b); w != nil {
		return polynomialNewtonJoin(e, w)
	}
	return e
}
func (n *PolynomialNewtonInterpolator) destroyNumerals(v *polynomialVector, target uint32, cursor *uint32) error {
	count, e := v.size(n.owner)
	if e != nil {
		return e
	}
	if *cursor < target {
		*cursor = target
	}
	for *cursor < count {
		a, e := v.coefficient(n.owner, *cursor)
		if e != nil {
			return e
		}
		e = n.owner.modular.Del(&a)
		if w := v.putCoefficient(n.owner, *cursor, a); w != nil {
			return polynomialNewtonJoin(e, w)
		}
		if e != nil {
			return e
		}
		*cursor++
	}
	if v.raw == (util.Block{}) {
		*cursor = 0
		return nil
	}
	if e = v.publishSize(n.owner, target); e != nil {
		return e
	}
	*cursor = 0
	return nil
}
func (n *PolynomialNewtonInterpolator) scopeCleanup(owning, abort bool) error {
	s := &n.scope
	m := n.owner
	for {
		if s.kind == 1 {
			switch s.stage {
			case 0:
				if s.refsConstructed {
					if e := s.auxPoly.closeValue(m, owning); e != nil {
						return e
					}
				}
			case 1:
				if s.refsConstructed {
					if e := s.temp.closeValue(m, owning); e != nil {
						return e
					}
				}
			case 2:
				if e := m.modular.Del(&s.aux); e != nil {
					return e
				}
			case 3:
				if e := m.modular.Del(&s.product); e != nil {
					return e
				}
			default:
				if !abort && (s.auxPoly.assigning || s.temp.assigning) {
					return nil
				}
				if e := s.auxPoly.abort(m); e != nil {
					return e
				}
				if e := s.temp.abort(m); e != nil {
					return e
				}
				*s = polynomialNewtonScope{}
				return nil
			}
		} else if s.kind == 2 {
			switch s.stage {
			case 0:
				if s.cConstructed {
					if e := m.modular.Del(&s.c); e != nil {
						return e
					}
				}
			case 1:
				if e := s.auxPoly.closeValue(m, owning); e != nil {
					return e
				}
			case 2:
				if e := s.u.closeValue(m, owning); e != nil {
					return e
				}
			default:
				if !abort && (s.auxPoly.assigning || s.u.assigning) {
					return nil
				}
				if e := s.auxPoly.abort(m); e != nil {
					return e
				}
				if e := s.u.abort(m); e != nil {
					return e
				}
				*s = polynomialNewtonScope{}
				return nil
			}
		} else {
			return nil
		}
		s.stage++
	}
}
func (n *PolynomialNewtonInterpolator) finish(e error) error {
	n.scope.body = e
	d := n.scopeCleanup(false, false)
	if d != nil {
		n.scope.cleanup = d
	}
	return polynomialNewtonJoin(e, d)
}
func (n *PolynomialNewtonInterpolator) Add(input *util.Int, output Polynomial) (err error) {
	if e := n.open(); e != nil {
		return e
	}
	m := n.owner
	if polynomialSourceDebug && m.modular.IntegerMode() {
		return ErrPolynomialAssertion
	}
	sz, e := n.inputs.size(m)
	if e != nil {
		return e
	}
	if sz == 0 {
		if e = n.pushNumeral(&n.inputs, input); e != nil {
			return e
		}
		return n.vs.push(m, output)
	}
	n.scope = polynomialNewtonScope{kind: 1}
	defer func() { err = n.finish(err) }()
	s := &n.scope
	first, e := n.inputs.coefficient(m, 0)
	if e != nil {
		return e
	}
	if polynomialSourceDebug {
		same, e := m.modular.Equal(input, &first)
		if e != nil {
			return e
		}
		if same {
			return ErrPolynomialAssertion
		}
	}
	if e = m.modular.Sub(input, &first, &s.product); e != nil {
		return e
	}
	for i := uint32(1); i < sz; i++ {
		a, e := n.inputs.coefficient(m, i)
		if e != nil {
			return e
		}
		if polynomialSourceDebug {
			same, e := m.modular.Equal(input, &a)
			if e != nil {
				return e
			}
			if same {
				return ErrPolynomialAssertion
			}
		}
		if e = m.modular.Sub(input, &a, &s.aux); e != nil {
			return e
		}
		if e = m.modular.Mul(&s.product, &s.aux, &s.product); e != nil {
			return e
		}
	}
	if e = m.modular.Inv(&s.product); e != nil {
		return e
	}
	if e = n.pushNumeral(&n.inputs, input); e != nil {
		return e
	}
	if e = n.pushNumeral(&n.invs, &s.product); e != nil {
		return e
	}
	if polynomialSourceDebug {
		a, e := n.inputs.coefficient(m, sz)
		if e != nil {
			return e
		}
		same, e := m.modular.Equal(&a, input)
		if e != nil {
			return e
		}
		if !same {
			return ErrPolynomialAssertion
		}
	}
	s.refsConstructed = true
	p, e := n.vs.get(m, sz-1)
	if e != nil {
		return e
	}
	if e = s.temp.assign(m, p); e != nil {
		return e
	}
	for j := int32(sz - 2); j >= 0; j-- {
		a, e := n.inputs.coefficient(m, uint32(j))
		if e != nil {
			return e
		}
		if e = m.modular.Sub(input, &a, &s.aux); e != nil {
			return e
		}
		if e = n.assertNumeralNormalized(&s.aux); e != nil {
			return e
		}
		p, e = n.scale(&s.aux, s.temp.value)
		if e != nil {
			return e
		}
		if e = s.auxPoly.assign(m, p); e != nil {
			return e
		}
		v, e := n.vs.get(m, uint32(j))
		if e != nil {
			return e
		}
		p, e = m.Add(s.auxPoly.value, v)
		if e != nil {
			return e
		}
		if e = s.temp.assign(m, p); e != nil {
			return e
		}
		if e = n.assertPolynomialNormalized(s.temp.value); e != nil {
			return e
		}
	}
	p, e = m.Sub(output, s.temp.value)
	if e != nil {
		return e
	}
	if e = s.auxPoly.assign(m, p); e != nil {
		return e
	}
	if e = n.assertPolynomialNormalized(s.auxPoly.value); e != nil {
		return e
	}
	a, e := n.invs.coefficient(m, sz)
	if e != nil {
		return e
	}
	p, e = n.scale(&a, s.auxPoly.value)
	if e != nil {
		return e
	}
	if e = s.auxPoly.assign(m, p); e != nil {
		return e
	}
	if e = n.assertPolynomialNormalized(s.auxPoly.value); e != nil {
		return e
	}
	return n.vs.push(m, s.auxPoly.value)
}
func (n *PolynomialNewtonInterpolator) scale(a *util.Int, p Polynomial) (Polynomial, error) {
	u, e := n.owner.monomials.Unit()
	if e != nil {
		return Polynomial{}, e
	}
	return n.owner.Scale(a, u, p)
}
func (n *PolynomialNewtonInterpolator) Mk(x uint32, r *PolynomialRef) (err error) {
	if e := n.open(); e != nil {
		return e
	}
	if r == nil || r.owner != n.owner {
		return ErrPolynomialOwner
	}
	if e := r.open(); e != nil {
		return e
	}
	m := n.owner
	if polynomialSourceDebug && m.modular.IntegerMode() {
		return ErrPolynomialAssertion
	}
	n.scope = polynomialNewtonScope{kind: 2, refsConstructed: true}
	defer func() { err = n.finish(err) }()
	s := &n.scope
	rawNum, e := n.inputs.size(m)
	if e != nil {
		return e
	}
	num := int32(rawNum)
	if num == math.MinInt32 {
		return arena.ErrBounds
	} // Reached signed d=num-1 UB domain.
	d := num - 1
	if polynomialSourceDebug && num <= 0 {
		return ErrPolynomialAssertion
	}
	p, e := n.vs.get(m, uint32(d))
	if e != nil {
		return e
	}
	if e = s.u.assign(m, p); e != nil {
		return e
	}
	s.c = util.Int{}
	s.cConstructed = true
	if d == math.MinInt32 {
		return arena.ErrBounds
	}
	for k := d - 1; k >= 0; k-- {
		a, e := n.inputs.coefficient(m, uint32(k))
		if e != nil {
			return e
		}
		if e = m.modular.Copy(&s.c, &a); e != nil {
			return e
		}
		if e = m.modular.Neg(&s.c); e != nil {
			return e
		}
		one := [1]util.Int{util.NewInt(1)}
		p, e = m.MkLinearTake(one[:], []uint32{x}, &s.c)
		if e != nil {
			return e
		}
		if e = s.auxPoly.assign(m, p); e != nil {
			return e
		}
		p, e = m.Mul(s.u.value, s.auxPoly.value)
		if e != nil {
			return e
		}
		if e = s.auxPoly.assign(m, p); e != nil {
			return e
		}
		aPoly, e := n.vs.get(m, uint32(k))
		if e != nil {
			return e
		}
		p, e = m.Add(s.auxPoly.value, aPoly)
		if e != nil {
			return e
		}
		if e = s.u.assign(m, p); e != nil {
			return e
		}
	}
	return r.Assign(s.u.value)
}
func (n *PolynomialNewtonInterpolator) Reset() (err error) {
	if n != nil && n.self == n && n.resetCause != nil {
		return n.resetCause
	}
	defer func() {
		if err != nil && n != nil && n.self == n && n.resetStage != 0 {
			n.resetCause = err
		}
	}()
	if n == nil || n.self != n {
		return ErrPolynomialOwner
	}
	if n.closed || n.closing || n.constructing {
		return arena.ErrClosed
	}
	if n.scope.kind != 0 {
		return ErrPolynomialPending
	}
	if n.resetStage == 0 {
		n.resetStage = 1
	}
	if n.resetStage == 1 {
		if e := n.destroyNumerals(&n.inputs, 0, &n.inputDrain); e != nil {
			return e
		}
		n.resetStage = 2
	}
	if n.resetStage == 2 {
		if e := n.destroyNumerals(&n.invs, 1, &n.invDrain); e != nil {
			return e
		}
		n.resetStage = 3
	}
	if n.resetStage == 3 {
		if e := n.vs.reset(n.owner); e != nil {
			return e
		}
		n.resetStage = 4
	}
	if polynomialSourceDebug {
		a, e := n.invs.coefficient(n.owner, 0)
		if e != nil {
			return e
		}
		zero, e := n.owner.modular.IsZero(&a)
		if e != nil {
			return e
		}
		if !zero {
			return ErrPolynomialAssertion
		}
	}
	n.resetStage = 0
	return nil
}

// These are named Go owning-abort fields, not constructed source members.
func (n *PolynomialNewtonInterpolator) hasAbortRoots() bool {
	if n.vs.incoming != (Polynomial{}) || n.scope.kind != 0 {
		return true
	}
	for _, v := range [3]*polynomialVector{&n.vs.values, &n.invs, &n.inputs} {
		if v.pending != (util.Block{}) || v.old != (util.Block{}) || v.retiredActive != 0 {
			return true
		}
	}
	return false
}
func (n *PolynomialNewtonInterpolator) sourceBacking(v *polynomialVector) error {
	if v.raw != (util.Block{}) {
		return n.owner.memory.Deallocate(&v.raw)
	}
	return nil
}

// sourceClose is the outer lexical boundary. An actual stopped younger
// destructor prevents reaching class member destruction. Owning acknowledgement
// belongs to the caller's later Close/CleanupNewtonInterpolation operation.
func (n *PolynomialNewtonInterpolator) sourceClose() error {
	if n.scope.cleanup != nil {
		return n.scope.cleanup
	}
	return n.closeMode(false, false)
}

func (n *PolynomialNewtonInterpolator) Close() error { return n.closeMode(true, true) }
func (n *PolynomialNewtonInterpolator) closeMode(owning, abort bool) error {
	if n == nil || n.self != n {
		return ErrPolynomialOwner
	}
	if n.closed {
		return nil
	}
	n.closing = true
	if e := n.scopeCleanup(owning, false); e != nil {
		return e
	}
	if n.closeStage == 0 {
		if e := n.vs.resetMode(n.owner, owning); e != nil {
			return e
		}
		if e := n.sourceBacking(&n.vs.values); e != nil {
			return e
		}
		n.closeStage++
	}
	if n.closeStage == 1 {
		if e := n.destroyNumerals(&n.invs, 0, &n.invDrain); e != nil {
			return e
		}
		if e := n.sourceBacking(&n.invs); e != nil {
			return e
		}
		n.closeStage++
	}
	if n.closeStage == 2 {
		if e := n.destroyNumerals(&n.inputs, 0, &n.inputDrain); e != nil {
			return e
		}
		if e := n.sourceBacking(&n.inputs); e != nil {
			return e
		}
		n.closeStage++
	}
	// A failed push's extra INC is not a constructed source vector member.
	// Its separate owning abort follows all completed class member destructors.
	if !abort && n.hasAbortRoots() {
		return nil
	}
	if e := n.scopeCleanup(owning, true); e != nil {
		return e
	}
	if e := n.vs.abortIncoming(n.owner); e != nil {
		return e
	}
	for _, v := range [3]*polynomialVector{&n.vs.values, &n.invs, &n.inputs} {
		if e := v.close(n.owner); e != nil {
			return e
		}
	}
	n.owner.integer.Budget().Release(n.header)
	n.header = 0
	n.closed = true
	return nil
}

func (n *PolynomialNewtonInterpolator) assertNumeralNormalized(a *util.Int) error {
	if !polynomialSourceDebug {
		return nil
	}
	ok, e := n.owner.modular.IsPNormalized(a)
	if e != nil {
		return e
	}
	if !ok {
		return ErrPolynomialAssertion
	}
	return nil
}
func (n *PolynomialNewtonInterpolator) assertPolynomialNormalized(p Polynomial) error {
	if !polynomialSourceDebug {
		return nil
	}
	count, e := n.owner.readWord(p, 8)
	if e != nil {
		return e
	}
	for i := uint32(0); i < count; i++ {
		a, e := n.owner.coefficientCore(p, i)
		if e != nil {
			return e
		}
		if e = n.assertNumeralNormalized(&a); e != nil {
			return e
		}
	}
	return nil
}

// NewtonInterpolation preserves the actual method-local interpolator until its
// reached member destruction succeeds; its field is not reusable warm scratch.
func (m *MultivariatePolynomialCore) NewtonInterpolation(x, d uint32, inputs []util.Int, outputs []Polynomial, r *PolynomialRef) (err error) {
	if e := m.check(); e != nil {
		return e
	}
	if m.newtonLocal != nil {
		return ErrPolynomialPending
	}
	if polynomialSourceDebug && m.modular.IntegerMode() {
		return ErrPolynomialAssertion
	}
	n, e := NewPolynomialNewtonInterpolator(m)
	if n != nil {
		m.newtonLocal = n
	}
	if e != nil {
		if n != nil {
			// Failed construction destroys only completed source members. A
			// TLS returned allocation stays named until explicit owning abort.
			c := n.closeMode(false, false)
			if n.closed {
				m.newtonLocal = nil
			}
			return polynomialNewtonJoin(e, c)
		}
		return e
	}
	defer func() {
		c := n.sourceClose()
		if n.closed {
			m.newtonLocal = nil
		}
		err = polynomialNewtonJoin(err, c)
	}()
	for i := uint32(0); i <= d; i++ {
		if uint64(i) >= uint64(len(inputs)) || uint64(i) >= uint64(len(outputs)) {
			return arena.ErrBounds
		}
		if e = n.Add(&inputs[i], outputs[i]); e != nil {
			return e
		}
		if i == ^uint32(0) {
			return arena.ErrBounds
		}
	}
	return n.Mk(x, r)
}
func (m *MultivariatePolynomialCore) CleanupNewtonInterpolation() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	if m.newtonLocal == nil {
		return nil
	}
	if e := m.newtonLocal.Close(); e != nil {
		return e
	}
	m.newtonLocal = nil
	return nil
}
