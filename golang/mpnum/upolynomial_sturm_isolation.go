// Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

type upolySSFrame struct {
	lower, upper     BinaryRational
	lowerSV, upperSV uint32
}
type upolySSStack struct {
	values []upolySSFrame
	size   uint32
}

func (b *BinaryRationalManager) pushSS(s *upolySSStack) (*upolySSFrame, error) {
	if s.size == math.MaxUint32 {
		return nil, arena.ErrBounds
	}
	if s.size == uint32(cap(s.values)) {
		stride := uint64(reflect.TypeOf(upolySSFrame{}).Size())
		n, bytes, e := upolyIsolationNextCapacity(uint32(cap(s.values)), 56, stride, uint64(int(^uint(0)>>1)))
		if e != nil {
			return nil, e
		}
		if e = b.z.budget.Reserve(bytes); e != nil {
			return nil, e
		}
		old := s.values
		next := make([]upolySSFrame, n)
		s.values = next
		for i := uint32(0); i < s.size; i++ {
			next[i] = old[i]
			old[i] = upolySSFrame{}
		}
		b.z.budget.Release(uint64(cap(old)) * stride)
	}
	i := s.size
	s.values[i] = upolySSFrame{}
	s.size++
	return &s.values[i], nil
}
func (b *BinaryRationalManager) popSS(s *upolySSStack) error {
	if s.size == 0 {
		return arena.ErrBounds
	}
	f := &s.values[s.size-1]
	if e := b.Del(&f.lower); e != nil {
		return e
	}
	if e := b.Del(&f.upper); e != nil {
		return e
	}
	s.size--
	return nil
}
func (b *BinaryRationalManager) dropSS(s *upolySSStack) {
	for i := uint32(0); i < s.size; i++ {
		b.Del(&s.values[i].lower)
		b.Del(&s.values[i].upper)
	}
	b.z.budget.Release(uint64(cap(s.values)) * uint64(reflect.TypeOf(upolySSFrame{}).Size()))
	*s = upolySSStack{}
}
func (b *BinaryRationalManager) addSSInterval(lower, upper *BinaryRational, l, u *BinaryRationalVector) error {
	if _, e := b.binaryPushDefault(&l.state); e != nil {
		return e
	}
	if _, e := b.binaryPushDefault(&u.state); e != nil {
		return e
	}
	if e := b.Copy(&l.state.values[l.state.size-1], lower); e != nil {
		return e
	}
	return b.Copy(&u.state.values[u.state.size-1], upper)
}
func (m *UPolynomialManager) pushSSFrame(v UPolynomialView, b *BinaryRationalManager, s *upolySSStack, r, l, u *BinaryRationalVector, lowerSV, upperSV uint32, lower, upper *BinaryRational) error {
	if modularSourceDebug {
		same, e := b.Equal(lower, upper)
		if e != nil {
			return e
		}
		if same {
			return ErrUPolynomialAssertion
		}
	}
	if lowerSV == upperSV {
		return nil
	}
	if lowerSV == upperSV+1 {
		sign, e := m.EvalSignBinary(v, upper)
		if e != nil {
			return e
		}
		if sign == 0 {
			if _, e = b.binaryPushDefault(&r.state); e != nil {
				return e
			}
			return b.Copy(&r.state.values[r.state.size-1], upper)
		}
		return b.addSSInterval(lower, upper, l, u)
	}
	f, e := b.pushSS(s)
	if e != nil {
		return e
	}
	if e = b.Copy(&f.lower, lower); e != nil {
		return e
	}
	if e = b.Copy(&f.upper, upper); e != nil {
		return e
	}
	f.lowerSV = lowerSV
	f.upperSV = upperSV
	return nil
}
func (m *UPolynomialManager) SturmIsolateRootsWithBounds(v UPolynomialView, neg, pos uint32, b *BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.isolationOwners(b, r, l, u); e != nil {
		return e
	}
	if modularSourceDebug {
		zero, e := m.HasZeroRoots(v)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	var seq upolySequenceState
	defer m.sequenceDestroy(&seq)
	var mid, lower, upper BinaryRational
	defer b.Del(&mid)
	defer b.Del(&lower)
	defer b.Del(&upper)
	if e := m.sturmUnary(v, &seq); e != nil {
		return e
	}
	var s upolySSStack
	defer b.dropSS(&s)
	minus, e := m.sequenceVariations(&seq, 1, nil)
	if e != nil {
		return e
	}
	zeroSV, e := m.sequenceVariations(&seq, 2, nil)
	if e != nil {
		return e
	}
	plus, e := m.sequenceVariations(&seq, 0, nil)
	if e != nil {
		return e
	}
	if plus >= minus {
		return nil
	}
	two := NewBinaryInt(2)
	if e = b.Power(&two, neg, &lower); e != nil {
		return e
	}
	if e = b.Neg(&lower); e != nil {
		return e
	}
	if e = b.Power(&two, pos, &upper); e != nil {
		return e
	}
	zero := NewBinaryInt(0)
	if e = m.pushSSFrame(v, b, &s, r, l, u, minus, zeroSV, &lower, &zero); e != nil {
		return e
	}
	if e = m.pushSSFrame(v, b, &s, r, l, u, zeroSV, plus, &zero, &upper); e != nil {
		return e
	}
	for s.size != 0 {
		if e = m.checkpoint(); e != nil {
			return e
		}
		f := &s.values[s.size-1]
		minus, plus = f.lowerSV, f.upperSV
		if e = b.Swap(&lower, &f.lower); e != nil {
			return e
		}
		if e = b.Swap(&upper, &f.upper); e != nil {
			return e
		}
		if e = b.popSS(&s); e != nil {
			return e
		}
		if modularSourceDebug && minus <= plus+1 {
			return ErrUPolynomialAssertion
		}
		if e = b.Add(&lower, &upper, &mid); e != nil {
			return e
		}
		if e = b.Div2(&mid); e != nil {
			return e
		}
		midSV, e := m.sequenceVariations(&seq, 3, &mid)
		if e != nil {
			return e
		}
		if e = m.pushSSFrame(v, b, &s, r, l, u, minus, midSV, &lower, &mid); e != nil {
			return e
		}
		if e = m.pushSSFrame(v, b, &s, r, l, u, midSV, plus, &mid, &upper); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) SturmIsolateRoots(v UPolynomialView, b *BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.isolationOwners(b, r, l, u); e != nil {
		return e
	}
	if modularSourceDebug && l.state.size != u.state.size {
		return ErrUPolynomialAssertion
	}
	var p upolyVectorState
	defer m.dropLocalVector(&p)
	if e := m.setView(&p, v); e != nil {
		return e
	}
	if e := m.normalize(m.stateView(&p)); e != nil {
		return e
	}
	pos, e := m.KnuthPositiveRootUpperBound(v)
	if e != nil {
		return e
	}
	neg, e := m.KnuthNegativeRootUpperBound(v)
	if e != nil {
		return e
	}
	return m.SturmIsolateRootsWithBounds(m.stateView(&p), neg, pos, b, r, l, u)
}
