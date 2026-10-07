// Source: src/math/polynomial/polynomial.cpp:3134-3182;
// src/util/scoped_numeral_vector.h; src/math/polynomial/linear_eq_solver.h.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/util"
)

// Concrete source locals are charged in their enclosing owner, and constructed
// at their declaration sites. The scalar cursor stops before older destruction.
type polynomialSparseInterpolationScope struct {
	vectors                                              [3]polynomialVector // cs, new_as, as
	mons                                                 skeletonMonomialBuffer
	aux                                                  util.Int
	solver                                               ModularLinearSolver
	active, solverConstructed, sourceFinalized, topOwner bool
	cursor                                               uint8
	numCursor                                            [3]uint32
	generation                                           uint64
	bodyCause, cleanupCause                              error
}

func (s *PolynomialSparseInterpolator) scopedPush(v *polynomialVector, a *util.Int) error {
	m := s.owner.core
	if e := v.appendCoefficient(m, util.Int{}); e != nil {
		return e
	}
	n, e := v.size(m)
	if e != nil {
		return e
	}
	return s.copyField(v, n-1, a)
}
func (s *PolynomialSparseInterpolator) scopedReset(v *polynomialVector) error {
	var cursor uint32
	if e := s.deleteFields(v, &cursor, false); e != nil {
		return e
	}
	return v.reset(s.owner.core)
}
func (s *PolynomialSparseInterpolator) scopedResize(v *polynomialVector, n uint32) error {
	m := s.owner.core
	old, e := v.size(m)
	if e != nil {
		return e
	}
	if n <= old {
		cursor := n
		if e = s.deleteFields(v, &cursor, false); e != nil {
			return e
		}
		return v.publishSize(m, n)
	}
	for {
		capacity, e := v.capacity(m)
		if e != nil {
			return e
		}
		if n <= capacity {
			break
		}
		if e = v.expand(m); e != nil {
			return e
		}
	}
	// vector::resize publishes all new size before ascending default placement.
	if e = v.publishSize(m, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = v.putCoefficient(m, i, util.Int{}); e != nil {
			return e
		}
		if i >= v.initialized {
			v.initialized = i + 1
		}
	}
	return nil
}
func (s *PolynomialSparseInterpolator) finishInterpolationSource() (err error) {
	f := &s.interpolation
	m := s.owner.core
	if !f.active || f.sourceFinalized {
		return nil
	}
	if f.cleanupCause != nil {
		return f.cleanupCause
	}
	if e := m.pscSourceStopped(); e != nil {
		f.cleanupCause = e
		return e
	}
	defer func() {
		if err != nil {
			f.cleanupCause = err
		}
	}()
	if f.cursor == 0 {
		if f.solverConstructed {
			if e := f.solver.SourceFinalize(); e != nil {
				return e
			}
		}
		f.cursor = 1
	}
	if f.cursor == 1 {
		if _, e := util.SourceNumeralDel(m.integer, &f.aux); e != nil {
			return e
		}
		f.cursor = 2
	}
	if f.cursor == 2 {
		if f.mons.raw != (util.Block{}) {
			if e := m.memory.Deallocate(&f.mons.raw); e != nil {
				return e
			}
		}
		f.cursor = 3
	}
	for f.cursor < 6 {
		index := 5 - f.cursor
		v := &f.vectors[index]
		if e := s.deleteFields(v, &f.numCursor[index], false); e != nil {
			return e
		}
		if v.raw != (util.Block{}) {
			e := m.memory.Deallocate(&v.raw)
			if v.raw == (util.Block{}) {
				v.initialized = 0
			}
			if e != nil {
				return e
			}
		}
		f.cursor++
	}
	f.sourceFinalized = true
	return nil
}
func (s *PolynomialSparseInterpolator) CleanupInterpolation() error {
	if s == nil || s.self != s {
		return ErrPolynomialOwner
	}
	f := &s.interpolation
	if !f.active {
		return nil
	}
	m := s.owner.core
	// Owning continuation follows completed reverse lexical destruction, without
	// replaying a stopped source destructor or discarding published fields.
	if f.solverConstructed {
		// This exact completed local already attempted its source destructor. An
		// owning abort resumes the existing owning cursor without replaying it.
		if f.solver.sourceCause != nil {
			if e := f.solver.identity(); e != nil {
				return e
			}
			if f.solver.manager != m.modular {
				return ErrPolynomialOwner
			}
			f.solver.closing = true
		}
		if e := f.solver.Close(); e != nil {
			return e
		}
	}
	if e := s.cleanupNumeric(); e != nil {
		return e
	}
	if e := m.deleteNumeral(&f.aux); e != nil {
		return e
	}
	for _, p := range []*util.Block{&f.mons.raw, &f.mons.pending} {
		if *p != (util.Block{}) {
			if e := m.memory.Deallocate(p); e != nil {
				return e
			}
		}
	}
	for i := 2; i >= 0; i-- {
		if e := f.vectors[i].close(m); e != nil {
			return e
		}
	}
	if f.topOwner && f.generation != 0 && m.pscReceipts.generation == f.generation {
		if e := m.CleanupPSC(); e != nil {
			return e
		}
	}
	*f = polynomialSparseInterpolationScope{}
	return nil
}
func (s *PolynomialSparseInterpolator) Interpolate(out *PolynomialRef) (ok bool, err error) {
	if e := s.open(); e != nil {
		return false, e
	}
	m := s.owner.core
	if out == nil || out.owner != m {
		return false, ErrPolynomialOwner
	}
	if e := out.open(); e != nil {
		return false, e
	}
	if s.skeleton == nil {
		if polynomialSourceDebug {
			return false, ErrPolynomialAssertion
		}
		return false, ErrPolynomialOwner
	}
	f := &s.interpolation
	if f.active {
		return false, ErrPolynomialPending
	}
	topOwner := m.pscReceipts.depth == 0
	if e := m.pscEnter(); e != nil {
		return false, e
	}
	*f = polynomialSparseInterpolationScope{active: true, topOwner: topOwner, generation: m.pscReceipts.generation, mons: skeletonMonomialBuffer{capacity: 128}}
	for i := range f.vectors {
		f.vectors[i].width = 12
	}
	defer func() {
		m.pscLeave()
		f.bodyCause = err
		if e := s.finishInterpolationSource(); e != nil {
			err = errors.Join(err, e)
			return
		}
		pending := f.mons.pending != (util.Block{}) || (f.topOwner && m.pscPending()) || s.numeric.PendingObject() != (util.ObjectRef{})
		for _, root := range s.numeric.PendingAllocatorBlocks() {
			pending = pending || root != (util.Block{})
		}
		for i := range f.vectors {
			v := &f.vectors[i]
			pending = pending || v.pending != (util.Block{}) || v.old != (util.Block{}) || v.retiredActive != 0
		}
		if !pending {
			if e := s.CleanupInterpolation(); e != nil {
				err = errors.Join(err, e)
			}
		}
	}()
	// cs, new_as, as, mons and aux have completed default construction above.
	// The actual solver is constructed last, only after its source field guard.
	if e := f.solver.constructDefault(m.modular); e != nil {
		return false, e
	}
	f.solverConstructed = true
	count, e := s.skeleton.NumEntries()
	if e != nil {
		return false, e
	}
	cs, newAs, as := &f.vectors[0], &f.vectors[1], &f.vectors[2]
	for k := uint32(0); k < count; k++ {
		entry, e := s.skeleton.Entry(k)
		if e != nil {
			return false, e
		}
		n := entry.NumPowers
		if e = f.solver.Resize(n); e != nil {
			return false, e
		}
		if e = s.scopedResize(newAs, n); e != nil {
			return false, e
		}
		for i := uint32(0); i < n; i++ {
			in, e := s.inputs.coefficient(m, i)
			if e != nil {
				return false, e
			}
			if e = s.scopedReset(cs); e != nil {
				return false, e
			}
			for j := uint32(0); j < n; j++ {
				power, e := s.skeleton.Power(entry, j)
				if e != nil {
					return false, e
				}
				before, x := s.numericBefore()
				if x != nil {
					return false, x
				}
				e = m.modular.Power(&in, power, &f.aux)
				e = polynomialNewtonJoin(e, s.numericAfter(before))
				if e != nil {
					return false, e
				}
				if e = s.scopedPush(cs, &f.aux); e != nil {
					return false, e
				}
			}
			b, e := s.outputs.coefficient(m, entry.FirstPower+i)
			if e != nil {
				return false, e
			}
			before, x := s.numericBefore()
			if x != nil {
				return false, x
			}
			e = f.solver.AddIndexed(i, util.IntegerArrayView{Runtime: m.memory, Raw: cs.raw, Count: n, Offset: 8, Stride: 12}, &b)
			e = polynomialNewtonJoin(e, s.numericAfter(before))
			if e != nil {
				return false, e
			}
		}
		before, x := s.numericBefore()
		if x != nil {
			return false, x
		}
		solved, e := f.solver.SolveIndexed(util.IntegerArrayView{Runtime: m.memory, Raw: newAs.raw, Count: n, Offset: 8, Stride: 12})
		e = polynomialNewtonJoin(e, s.numericAfter(before))
		if e != nil {
			return false, e
		}
		if !solved {
			return false, nil
		}
		for i := uint32(0); i < n; i++ {
			a, e := newAs.coefficient(m, i)
			if e != nil {
				return false, e
			}
			zero, e := m.modular.IsZero(&a)
			if e != nil {
				return false, e
			}
			if zero {
				continue
			}
			if e = s.scopedPush(as, &a); e != nil {
				return false, e
			}
			mono, e := s.skeleton.Original(entry, i)
			if e != nil {
				return false, e
			}
			if e = f.mons.push(m, mono); e != nil {
				return false, e
			}
		}
	}
	view := monomialArrayView{inline: &f.mons.initial, count: f.mons.size}
	if f.mons.raw != (util.Block{}) {
		view = monomialArrayView{runtime: m.memory, raw: f.mons.raw, count: f.mons.size, stride: 16}
	}
	before, x := s.numericBefore()
	if x != nil {
		return false, x
	}
	result, e := m.mkTermsIndexed(as, view)
	e = polynomialNewtonJoin(e, s.numericAfter(before))
	if e != nil {
		return false, e
	}
	// Reference assignment may invoke a caller deletion handler. Its newly
	// allocated numeric roots belong to that caller, never this lexical scope.
	e = out.root.assign(m, result)
	if e != nil {
		return false, e
	}
	return true, nil
}
