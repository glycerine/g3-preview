package polynomial

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type upolySSFrame struct {
	lower, upper     util.BinaryRational
	lowerSV, upperSV uint32
}
type upolySSStack struct {
	values []upolySSFrame
}

func numericBinaryRationalManager_pushSS(b *util.BinaryRationalManager, s *upolySSStack) (*upolySSFrame, error) {
	if uint64(len(s.values)) >= math.MaxUint32 || len(s.values) == int(^uint(0)>>1) {
		return nil, arena.ErrBounds
	}
	old := s.values
	s.values = append(s.values, upolySSFrame{})
	if len(old) != 0 && &old[0] != &s.values[0] {
		clear(old)
	}
	return &s.values[len(s.values)-1], nil
}
func numericBinaryRationalManager_popSS(b *util.BinaryRationalManager, s *upolySSStack) error {
	if uint32(len(s.values)) == 0 {
		return arena.ErrBounds
	}
	f := &s.values[uint32(len(s.values))-1]
	if e := b.Del(&f.lower); e != nil {
		return e
	}
	if e := b.Del(&f.upper); e != nil {
		return e
	}
	s.values = s.values[:len(s.values)-1]
	return nil
}
func numericBinaryRationalManager_dropSS(b *util.BinaryRationalManager, s *upolySSStack) {
	for i := uint32(0); i < uint32(len(s.values)); i++ {
		b.Del(&s.values[i].lower)
		b.Del(&s.values[i].upper)
	}
	*s = upolySSStack{}
}
func numericBinaryRationalManager_addSSInterval(b *util.BinaryRationalManager, lower, upper *util.BinaryRational, l, u *BinaryRationalVector) error {
	if _, e := numericBinaryRationalManager_binaryPushDefault(b, &l.state); e != nil {
		return e
	}
	if _, e := numericBinaryRationalManager_binaryPushDefault(b, &u.state); e != nil {
		return e
	}
	if e := b.Copy(&l.state.values[l.state.size-1], lower); e != nil {
		return e
	}
	return b.Copy(&u.state.values[u.state.size-1], upper)
}
func (m *UPolynomialManager) pushSSFrame(v UPolynomialView, b *util.BinaryRationalManager, s *upolySSStack, r, l, u *BinaryRationalVector, lowerSV, upperSV uint32, lower, upper *util.BinaryRational) error {
	if polynomialSourceDebug {
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
			if _, e = numericBinaryRationalManager_binaryPushDefault(b, &r.state); e != nil {
				return e
			}
			return b.Copy(&r.state.values[r.state.size-1], upper)
		}
		return numericBinaryRationalManager_addSSInterval(b, lower, upper, l, u)
	}
	f, e := numericBinaryRationalManager_pushSS(b, s)
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
func (m *UPolynomialManager) SturmIsolateRootsWithBounds(v UPolynomialView, neg, pos uint32, b *util.BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.isolationOwners(b, r, l, u); e != nil {
		return e
	}
	if polynomialSourceDebug {
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
	var mid, lower, upper util.BinaryRational
	defer b.Del(&mid)
	defer b.Del(&lower)
	defer b.Del(&upper)
	if e := m.sturmUnary(v, &seq); e != nil {
		return e
	}
	var s upolySSStack
	defer numericBinaryRationalManager_dropSS(b, &s)
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
	two := util.NewBinaryInt(2)
	if e = b.Power(&two, neg, &lower); e != nil {
		return e
	}
	if e = b.Neg(&lower); e != nil {
		return e
	}
	if e = b.Power(&two, pos, &upper); e != nil {
		return e
	}
	zero := util.NewBinaryInt(0)
	if e = m.pushSSFrame(v, b, &s, r, l, u, minus, zeroSV, &lower, &zero); e != nil {
		return e
	}
	if e = m.pushSSFrame(v, b, &s, r, l, u, zeroSV, plus, &zero, &upper); e != nil {
		return e
	}
	for uint32(len(s.values)) != 0 {
		if e = m.checkpoint(); e != nil {
			return e
		}
		f := &s.values[uint32(len(s.values))-1]
		minus, plus = f.lowerSV, f.upperSV
		if e = b.Swap(&lower, &f.lower); e != nil {
			return e
		}
		if e = b.Swap(&upper, &f.upper); e != nil {
			return e
		}
		if e = numericBinaryRationalManager_popSS(b, &s); e != nil {
			return e
		}
		if polynomialSourceDebug && minus <= plus+1 {
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
func (m *UPolynomialManager) SturmIsolateRoots(v UPolynomialView, b *util.BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.isolationOwners(b, r, l, u); e != nil {
		return e
	}
	if polynomialSourceDebug && l.state.size != u.state.size {
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
