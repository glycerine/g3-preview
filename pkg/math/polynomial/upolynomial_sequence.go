// Licensed under the MIT license.
// Actual flat upolynomial_sequence and svector<unsigned> specialization.
package polynomial

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type upolySequenceUnsigned struct {
	values            []uint32
	size, initialized uint32
}
type upolySequenceState struct {
	coefficients  upolyVectorState
	begins, sizes upolySequenceUnsigned
}
type UPolynomialSequence struct {
	callerBinding util.NumericCallerBinding
	self          *UPolynomialSequence
	manager       *UPolynomialManager
	state         upolySequenceState
	header        uint64
	closed        bool
	displaying    bool
}
type UPolynomialSequenceStats struct {
	OwnerBytes, MetadataBytes, ManagerBackingBytes, TotalBytes                                                   uint64
	BeginCount, SizeCount, CoefficientCount, BeginCapacity, SizeCapacity, CoefficientCapacity, OwnedNumeralCells uint32
}

func upolySequenceSourceCapacity(old uint32) (n, oldBytes, newBytes uint32, e error) {
	if old == 0 {
		return 2, 0, 16, nil
	}
	oldBytes = 4*old + 8
	n = (3*old + 1) >> 1
	newBytes = 4*n + 8
	if n <= old || newBytes <= oldBytes {
		e = arena.ErrBounds
	}
	return
}
func upolySequenceNextCapacity(old uint32, host uint64) (uint32, uint64, error) {
	n, _, _, e := upolySequenceSourceCapacity(old)
	if e != nil {
		return 0, 0, e
	}
	if uint64(n)*4+8 > math.MaxUint32 || uint64(n) > host/4 {
		return 0, 0, arena.ErrBounds
	}
	return n, uint64(n) * 4, nil
}
func (m *UPolynomialManager) sequenceUnsignedPush(s *upolySequenceUnsigned, v uint32) error {
	if s.size == math.MaxUint32 {
		return arena.ErrBounds
	}
	if uint64(s.size) == uint64(cap(s.values)) {
		n, bytes, e := upolySequenceNextCapacity(uint32(cap(s.values)), uint64(int(^uint(0)>>1)))
		if e != nil {
			return e
		}
		if e = m.config.Integer.Budget().Reserve(bytes); e != nil {
			return e
		}
		next := make([]uint32, n)
		copy(next, s.values)
		m.config.Integer.Budget().Release(uint64(cap(s.values)) * 4)
		s.values = next
	}
	s.values[s.size] = v
	s.size++
	if s.size > s.initialized {
		s.initialized = s.size
	}
	return nil
}
func (m *UPolynomialManager) NewSequence() (*UPolynomialSequence, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	if m.liveSequences == math.MaxUint32 {
		return nil, arena.ErrSlotLimit
	}
	n := uint64(reflect.TypeOf(UPolynomialSequence{}).Size())
	if e := m.config.Integer.Budget().Reserve(n); e != nil {
		return nil, e
	}
	s := &UPolynomialSequence{manager: m, header: n}
	s.self = s
	m.liveSequences++
	return s, nil
}
func (s *UPolynomialSequence) check() error {
	if s == nil || s.self != s || s.manager == nil {
		return ErrUPolynomialOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return s.manager.check()
}
func (m *UPolynomialManager) sequencePush(s *upolySequenceState, v UPolynomialView, move bool) error {
	if e := m.sequenceUnsignedPush(&s.begins, s.coefficients.size); e != nil {
		return e
	}
	if e := m.sequenceUnsignedPush(&s.sizes, v.count); e != nil {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		destination, e := m.pushDefault(&s.coefficients)
		if e != nil {
			return e
		}
		source, e := m.read(v, i)
		if e != nil {
			return e
		}
		if move {
			*destination, *source = *source, *destination
		} else if e = m.modular.Copy(destination, source); e != nil {
			return e
		}
	}
	return nil
}
func (s *UPolynomialSequence) PushCopy(v UPolynomialView) error {
	if s != nil && s.self == s && s.displaying {
		return ErrUPolynomialCleanupPending
	}
	if e := s.check(); e != nil {
		return e
	}
	return s.manager.sequencePush(&s.state, v, false)
}
func (s *UPolynomialSequence) PushMove(v UPolynomialView) error {
	if s != nil && s.self == s && s.displaying {
		return ErrUPolynomialCleanupPending
	}
	if e := s.check(); e != nil {
		return e
	}
	return s.manager.sequencePush(&s.state, v, true)
}
func (s *UPolynomialSequence) Size() (uint32, error) {
	if e := s.check(); e != nil {
		return 0, e
	}
	return s.state.sizes.size, nil
}
func upolySequenceUnsignedRead(s *upolySequenceUnsigned, i uint32) (uint32, error) {
	if polynomialSourceDebug && i >= s.size {
		return 0, ErrUPolynomialAssertion
	}
	if i >= s.initialized {
		return 0, arena.ErrBounds
	}
	return s.values[i], nil
}
func (s *UPolynomialSequence) RowSize(i uint32) (uint32, error) {
	if e := s.check(); e != nil {
		return 0, e
	}
	return upolySequenceUnsignedRead(&s.state.sizes, i)
}
func (m *UPolynomialManager) sequenceRow(s *upolySequenceState, i uint32) (UPolynomialView, error) {
	begin, e := upolySequenceUnsignedRead(&s.begins, i)
	if e != nil {
		return UPolynomialView{}, e
	}
	n, e := upolySequenceUnsignedRead(&s.sizes, i)
	if e != nil {
		return UPolynomialView{}, e
	}
	v := m.stateView(&s.coefficients)
	v.start = begin
	v.count = n
	return v, nil
}
func (s *UPolynomialSequence) RowView(i uint32) (UPolynomialView, error) {
	if e := s.check(); e != nil {
		return UPolynomialView{}, e
	}
	return s.manager.sequenceRow(&s.state, i)
}
func (m *UPolynomialManager) sequenceReset(s *upolySequenceState) error {
	if e := m.reset(&s.coefficients); e != nil {
		return e
	}
	s.sizes.size = 0
	s.begins.size = 0
	return nil
}
func (s *UPolynomialSequence) Reset() error {
	if s != nil && s.self == s && s.displaying {
		return ErrUPolynomialCleanupPending
	}
	if e := s.check(); e != nil {
		return e
	}
	return s.manager.sequenceReset(&s.state)
}
func (m *UPolynomialManager) sequenceDestroy(s *upolySequenceState) error {
	if e := m.sequenceReset(s); e != nil {
		return e
	}
	m.config.Integer.Budget().Release(uint64(cap(s.sizes.values)) * 4)
	s.sizes = upolySequenceUnsigned{}
	m.config.Integer.Budget().Release(uint64(cap(s.begins.values)) * 4)
	s.begins = upolySequenceUnsigned{}
	m.freeBacking(&s.coefficients)
	return nil
}
func (s *UPolynomialSequence) Close() error {
	if s != nil && s.self == s && s.displaying {
		return ErrUPolynomialCleanupPending
	}
	if e := s.check(); e != nil {
		return e
	}
	if e := s.manager.sequenceDestroy(&s.state); e != nil {
		return e
	}
	s.manager.liveSequences--
	s.manager.config.Integer.Budget().Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}

// Coefficient backing is already charged in manager Stats. TotalBytes here
// counts only the distinct cold owner and metadata; ManagerBackingBytes is
// attribution, never a second reservation or a term in the combined sum.
func (s *UPolynomialSequence) Stats() UPolynomialSequenceStats {
	if s == nil || s.self != s || s.closed || s.manager.check() != nil {
		return UPolynomialSequenceStats{}
	}
	a := &s.state
	r := UPolynomialSequenceStats{OwnerBytes: s.header, MetadataBytes: uint64(cap(a.begins.values)+cap(a.sizes.values)) * 4, BeginCount: a.begins.size, SizeCount: a.sizes.size, CoefficientCount: a.coefficients.size, BeginCapacity: uint32(cap(a.begins.values)), SizeCapacity: uint32(cap(a.sizes.values)), CoefficientCapacity: a.coefficients.capacity}
	r.TotalBytes = r.OwnerBytes + r.MetadataBytes
	r.ManagerBackingBytes = uint64(a.coefficients.capacity) * uint64(reflect.TypeOf(util.Int{}).Size())
	for i := uint32(0); i < a.coefficients.size; i++ {
		if s.manager.config.Integer.IsLiveCellToken(s.manager.values(&a.coefficients)[i].NativeCellToken()) {
			r.OwnedNumeralCells++
		}
	}
	return r
}
