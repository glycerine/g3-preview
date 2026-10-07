// Source: src/math/polynomial/polynomial.cpp:3072-3140.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// This is the actual source sparse-interpolator owner. Both numeral arrays
// reside in Runtime bytes; the skeleton is borrowed for its source lifetime.
type PolynomialSparseInterpolator struct {
	self                                                         *PolynomialSparseInterpolator
	owner                                                        *MultivariatePolynomialManager
	skeleton                                                     *PolynomialSkeleton
	inputs, outputs                                              polynomialVector
	header                                                       uint64
	constructed, closing, closed, sourceStarted, sourceFinalized bool
	sourceStage                                                  uint8
	inputCursor, outputCursor                                    uint32
	originalCause, cleanupCause                                  error
	resetCursor                                                  uint32
	resetCause                                                   error
	resetActive                                                  bool
	interpolation                                                polynomialSparseInterpolationScope
	numeric                                                      polynomialNumericRoots
}

func NewPolynomialSparseInterpolator(m *MultivariatePolynomialManager, sk *PolynomialSkeleton) (s *PolynomialSparseInterpolator, err error) {
	if e := m.ready(); e != nil {
		return nil, e
	}
	if sk != nil && sk.owner != m.core {
		return nil, ErrPolynomialOwner
	}
	header := uint64(reflect.TypeOf(PolynomialSparseInterpolator{}).Size())
	if e := m.core.integer.Budget().Reserve(header); e != nil {
		return nil, e
	}
	s = &PolynomialSparseInterpolator{}
	err = initializePolynomialSparseInterpolator(m, sk, s, header)
	return s, err
}

func constructPolynomialSparseInterpolator(m *MultivariatePolynomialManager, sk *PolynomialSkeleton, s *PolynomialSparseInterpolator) error {
	if e := m.ready(); e != nil {
		return e
	}
	if sk != nil && sk.owner != m.core {
		return ErrPolynomialOwner
	}
	if s == nil {
		return ErrPolynomialOwner
	}
	if s.self != nil && !s.closed {
		return ErrPolynomialPending
	}
	return initializePolynomialSparseInterpolator(m, sk, s, 0)
}

func initializePolynomialSparseInterpolator(m *MultivariatePolynomialManager, sk *PolynomialSkeleton, s *PolynomialSparseInterpolator, header uint64) (err error) {
	*s = PolynomialSparseInterpolator{owner: m, skeleton: sk, header: header, inputs: polynomialVector{width: 12}, outputs: polynomialVector{width: 12}}
	s.self = s
	defer func() {
		if err != nil {
			s.originalCause = err
			s.sourceStage = 2
			if e := s.sourceFinish(); e != nil {
				err = polynomialNewtonJoin(err, e)
			}
		}
	}()
	if sk != nil {
		n, e := sk.NumEntries()
		if e != nil {
			return e
		}
		for i := uint32(0); i < n; i++ {
			entry, e := sk.Entry(i)
			if e != nil {
				return e
			}
			for j := uint32(0); j < entry.NumPowers; j++ {
				if e = s.outputs.appendCoefficient(m.core, util.Int{}); e != nil {
					return e
				}
			}
		}
	}
	s.constructed = true
	return nil
}
func (s *PolynomialSparseInterpolator) identity() error {
	if s == nil || s.self != s || s.owner == nil {
		return ErrPolynomialOwner
	}
	return s.owner.ready()
}
func (s *PolynomialSparseInterpolator) open() error {
	if e := s.identity(); e != nil {
		return e
	}
	if !s.constructed || s.closed || s.closing || s.sourceStarted || s.sourceFinalized {
		return arena.ErrClosed
	}
	return nil
}
func (s *PolynomialSparseInterpolator) Ready() (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	n, e := s.inputs.size(s.owner.core)
	if e != nil {
		return false, e
	}
	if s.skeleton == nil {
		return false, ErrPolynomialOwner
	}
	max, e := s.skeleton.MaxNumPowers()
	return n == max, e
}
func (s *PolynomialSparseInterpolator) copyField(v *polynomialVector, i uint32, a *util.Int) error {
	m := s.owner.core
	b, e := v.coefficient(m, i)
	if e != nil {
		return e
	}
	before, x := s.numericBefore()
	if x != nil {
		return x
	}
	e = m.modular.Copy(&b, a)
	e = polynomialNewtonJoin(e, s.numericAfter(before))
	return polynomialNewtonJoin(e, v.putCoefficient(m, i, b))
}
func (s *PolynomialSparseInterpolator) Add(in *util.Int, q Polynomial) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	if s.skeleton == nil {
		if polynomialSourceDebug {
			return false, ErrPolynomialAssertion
		}
		return false, ErrPolynomialOwner
	}
	m := s.owner.core
	inputIndex, e := s.inputs.size(m)
	if e != nil {
		return false, e
	}
	if polynomialSourceDebug {
		n, e := s.skeleton.MaxNumPowers()
		if e != nil {
			return false, e
		}
		if inputIndex >= n {
			return false, ErrPolynomialAssertion
		}
	}
	if e = s.inputs.appendCoefficient(m, util.Int{}); e != nil {
		return false, e
	}
	if e = s.copyField(&s.inputs, inputIndex, in); e != nil {
		return false, e
	}
	n, e := m.readWord(q, 8)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		term, e := m.monomialCore(q, i)
		if e != nil {
			return false, e
		}
		entryIndex, e := s.skeleton.EntryIndex(term)
		if e != nil {
			return false, e
		}
		if entryIndex == ^uint32(0) {
			return false, nil
		}
		entry, e := s.skeleton.Entry(entryIndex)
		if e != nil {
			return false, e
		}
		if inputIndex >= entry.NumPowers {
			continue
		}
		a, e := m.coefficientCore(q, i)
		if e != nil {
			return false, e
		}
		if e = s.copyField(&s.outputs, entry.FirstPower+inputIndex, &a); e != nil {
			return false, e
		}
	}
	return true, nil
}
func (s *PolynomialSparseInterpolator) deleteFields(v *polynomialVector, cursor *uint32, owning bool) error {
	m := s.owner.core
	n, e := v.size(m)
	if e != nil {
		return e
	}
	for *cursor < n {
		a, e := v.coefficient(m, *cursor)
		if e != nil {
			return e
		}
		if owning {
			e = m.deleteNumeral(&a)
		} else {
			_, e = util.SourceNumeralDel(m.integer, &a)
		}
		if w := v.putCoefficient(m, *cursor, a); w != nil {
			return polynomialNewtonJoin(e, w)
		}
		if e != nil {
			return e
		}
		*cursor++
	}
	return nil
}
func (s *PolynomialSparseInterpolator) sourceFinish() (err error) {
	if s.cleanupCause != nil {
		return s.cleanupCause
	}
	m := s.owner.core
	if e := s.finishInterpolationSource(); e != nil {
		return e
	}
	s.sourceStarted = true
	defer func() {
		if err != nil {
			s.cleanupCause = err
		}
	}()
	if s.sourceStage == 0 {
		if s.skeleton != nil {
			if e := s.deleteFields(&s.inputs, &s.inputCursor, false); e != nil {
				return e
			}
		}
		s.sourceStage = 1
	}
	if s.sourceStage == 1 {
		if s.skeleton != nil {
			if e := s.deleteFields(&s.outputs, &s.outputCursor, false); e != nil {
				return e
			}
		}
		s.sourceStage = 2
	}
	if s.sourceStage == 2 {
		if s.outputs.raw != (util.Block{}) {
			e := m.memory.Deallocate(&s.outputs.raw)
			if s.outputs.raw == (util.Block{}) {
				s.outputs.initialized = 0
			}
			if e != nil {
				return e
			}
		}
		s.sourceStage = 3
	}
	if s.sourceStage == 3 {
		if s.inputs.raw != (util.Block{}) {
			e := m.memory.Deallocate(&s.inputs.raw)
			if s.inputs.raw == (util.Block{}) {
				s.inputs.initialized = 0
			}
			if e != nil {
				return e
			}
		}
		s.sourceStage = 4
	}
	s.sourceFinalized = true
	return nil
}
func (s *PolynomialSparseInterpolator) SourceFinalize() error {
	if e := s.identity(); e != nil {
		return e
	}
	if s.sourceFinalized {
		return nil
	}
	return s.sourceFinish()
}
func (s *PolynomialSparseInterpolator) CloseChecked() error {
	if s == nil || s.self != s {
		return ErrPolynomialOwner
	}
	if s.closed {
		return nil
	}
	s.closing = true
	if e := s.CleanupInterpolation(); e != nil {
		return e
	}
	m := s.owner.core
	// Completed source destruction precedes owning abort. Owning retries use
	// exact published fields and cursors; constructor failure has no body dtor.
	if !s.sourceFinalized && s.cleanupCause == nil {
		if e := s.sourceFinish(); e != nil {
			return e
		}
	}
	if s.sourceStage < 2 {
		if e := s.deleteFields(&s.inputs, &s.inputCursor, true); e != nil {
			return e
		}
		if e := s.deleteFields(&s.outputs, &s.outputCursor, true); e != nil {
			return e
		}
	}
	if e := s.cleanupNumeric(); e != nil {
		return e
	}
	if e := s.outputs.close(m); e != nil {
		return e
	}
	if e := s.inputs.close(m); e != nil {
		return e
	}
	m.integer.Budget().Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}

func (s *PolynomialSparseInterpolator) Reset() (err error) {
	if e := s.open(); e != nil {
		return e
	}
	if s.resetCause != nil {
		return s.resetCause
	}
	if s.skeleton == nil {
		return ErrPolynomialOwner
	}
	s.resetActive = true
	defer func() {
		if err != nil {
			s.resetCause = err
		}
	}()
	m := s.owner.core
	n, e := s.inputs.size(m)
	if e != nil {
		return e
	}
	for s.resetCursor < n {
		a, e := s.inputs.coefficient(m, s.resetCursor)
		if e != nil {
			return e
		}
		_, e = util.SourceNumeralDel(m.integer, &a)
		if w := s.inputs.putCoefficient(m, s.resetCursor, a); w != nil {
			return polynomialNewtonJoin(e, w)
		}
		if e != nil {
			return e
		}
		s.resetCursor++
	}
	if e = s.inputs.reset(m); e != nil {
		return e
	}
	s.resetActive = false
	s.resetCursor = 0
	return nil
}
