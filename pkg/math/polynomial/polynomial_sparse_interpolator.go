// Source: src/math/polynomial/polynomial.cpp:3072-3140.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// This owner retains actual numerals in typed Go slices. The skeleton is
// borrowed for its source lifetime; native slice storage owns no Runtime blocks.
type PolynomialSparseInterpolator struct {
	self                                                         *PolynomialSparseInterpolator
	owner                                                        *MultivariatePolynomialManager
	skeleton                                                     *PolynomialSkeleton
	inputs, outputs                                              []unumeral.Int
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

	running  bool
	previous *PolynomialSparseInterpolator
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
	*s = PolynomialSparseInterpolator{owner: m, skeleton: sk, header: header}
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
				s.outputs = append(s.outputs, unumeral.Int{})
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
	if s != nil && s.self == s && s.running {
		return ErrPolynomialPending
	}
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
	n := uint32(len(s.inputs))
	if s.skeleton == nil {
		return false, ErrPolynomialOwner
	}
	max, e := s.skeleton.MaxNumPowers()
	return n == max, e
}
func (s *PolynomialSparseInterpolator) copyField(v []unumeral.Int, i uint32, a *unumeral.Int) error {
	if uint64(i) >= uint64(len(v)) {
		if polynomialSourceDebug {
			return ErrPolynomialAssertion
		}
		return arena.ErrBounds
	}
	before, e := s.numericBefore()
	if e != nil {
		return e
	}
	e = s.owner.core.modular.Copy(&v[i], a)
	return polynomialNewtonJoin(e, s.numericAfter(before))
}
func (s *PolynomialSparseInterpolator) Add(in *unumeral.Int, q Polynomial) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	s.beginExecution()
	defer s.endExecution()
	if s.skeleton == nil {
		if polynomialSourceDebug {
			return false, ErrPolynomialAssertion
		}
		return false, ErrPolynomialOwner
	}
	m := s.owner.core
	inputIndex := uint32(len(s.inputs))
	if polynomialSourceDebug {
		n, e := s.skeleton.MaxNumPowers()
		if e != nil {
			return false, e
		}
		if inputIndex >= n {
			return false, ErrPolynomialAssertion
		}
	}
	s.inputs = append(s.inputs, unumeral.Int{})
	if e := s.copyField(s.inputs, inputIndex, in); e != nil {
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
		if e = s.copyField(s.outputs, entry.FirstPower+inputIndex, &a); e != nil {
			return false, e
		}
	}
	return true, nil
}
func (s *PolynomialSparseInterpolator) deleteFields(v []unumeral.Int, cursor *uint32, owning bool) error {
	m := s.owner.core
	for uint64(*cursor) < uint64(len(v)) {
		var e error
		if owning {
			e = m.deleteNumeral(&v[*cursor])
		} else {
			_, e = unumeral.SourceNumeralDel(m.integer, &v[*cursor])
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
			if e := s.deleteFields(s.inputs, &s.inputCursor, false); e != nil {
				return e
			}
		}
		s.sourceStage = 1
	}
	if s.sourceStage == 1 {
		if s.skeleton != nil {
			if e := s.deleteFields(s.outputs, &s.outputCursor, false); e != nil {
				return e
			}
		}
		s.sourceStage = 2
	}
	if s.sourceStage == 2 {
		s.outputs = nil
		s.sourceStage = 3
	}
	if s.sourceStage == 3 {
		s.inputs = nil
		s.sourceStage = 4
	}
	s.sourceFinalized = true
	return nil
}
func (s *PolynomialSparseInterpolator) SourceFinalize() error {
	if s != nil && s.self == s && s.running {
		return ErrPolynomialPending
	}
	if e := s.identity(); e != nil {
		return e
	}
	s.beginExecution()
	defer s.endExecution()
	if s.sourceFinalized {
		return nil
	}
	return s.sourceFinish()
}
func (s *PolynomialSparseInterpolator) CloseChecked() error {
	if s == nil || s.self != s {
		return ErrPolynomialOwner
	}
	if s.running {
		return ErrPolynomialPending
	}
	if s.closed {
		return nil
	}
	s.beginExecution()
	defer s.endExecution()
	s.closing = true
	if e := s.cleanupInterpolation(); e != nil {
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
		if e := s.deleteFields(s.inputs, &s.inputCursor, true); e != nil {
			return e
		}
		if e := s.deleteFields(s.outputs, &s.outputCursor, true); e != nil {
			return e
		}
	}
	if e := s.cleanupNumeric(); e != nil {
		return e
	}
	s.outputs = nil
	s.inputs = nil
	m.integer.Budget().Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}

func (s *PolynomialSparseInterpolator) Reset() (err error) {
	if e := s.open(); e != nil {
		return e
	}
	s.beginExecution()
	defer s.endExecution()
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
	for uint64(s.resetCursor) < uint64(len(s.inputs)) {
		if _, e := unumeral.SourceNumeralDel(m.integer, &s.inputs[s.resetCursor]); e != nil {
			return e
		}
		s.resetCursor++
	}
	s.inputs = s.inputs[:0]
	s.resetActive = false
	s.resetCursor = 0
	return nil
}

// The active caller chain binds only actual operations to the exact borrowed
// core and skeleton. Nested operations on another interpolator restore it.
func (s *PolynomialSparseInterpolator) beginExecution() {
	core := s.owner.core
	s.previous = core.sparseCaller
	s.running = true
	core.sparseCaller = s
}
func (s *PolynomialSparseInterpolator) endExecution() {
	core := s.owner.core
	if core.sparseCaller != s {
		panic(ErrPolynomialOwner)
	}
	core.sparseCaller = s.previous
	s.previous = nil
	s.running = false
}
func (m *MultivariatePolynomialCore) sparseExecuting(sk *PolynomialSkeleton) bool {
	if m == nil || m.self != m {
		return false
	}
	for s := m.sparseCaller; s != nil; s = s.previous {
		if s.self == s && s.owner != nil && s.owner.self == s.owner && s.owner.core == m && s.running && (sk == nil || s.skeleton == sk) {
			return true
		}
	}
	return false
}
