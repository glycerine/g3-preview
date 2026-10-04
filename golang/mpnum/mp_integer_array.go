// Source: src/util/mpz.cpp array gcd; src/util/mpzzp.h array gcd. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
)

// A call-local concrete borrowed view, never a duplicated owning numeral array.
// The slice route retains the existing API's aliasing; the codec route loads
// only the source-reached numeral field from an actual Runtime backing.
type integerArrayView struct {
	values         []Int
	runtime        *memory.Runtime
	raw            memory.Block
	count          uint32
	offset, stride uint64
}

func (v integerArrayView) at(i uint32) (Int, error) {
	if i >= v.count {
		return Int{}, arena.ErrBounds
	}
	if v.runtime == nil {
		if uint64(i) >= uint64(len(v.values)) {
			return Int{}, arena.ErrBounds
		}
		return v.values[i], nil
	}
	if v.stride != 0 && uint64(i) > (math.MaxUint64-v.offset)/v.stride {
		return Int{}, arena.ErrBounds
	}
	o := v.offset + uint64(i)*v.stride
	if o > math.MaxUint64-12 {
		return Int{}, arena.ErrBounds
	}
	value, e := v.runtime.ReadUint32(v.raw, o)
	if e != nil {
		return Int{}, e
	}
	lo, e := v.runtime.ReadUint32(v.raw, o+4)
	if e != nil {
		return Int{}, e
	}
	hi, e := v.runtime.ReadUint32(v.raw, o+8)
	if e != nil {
		return Int{}, e
	}
	ref := uint64(lo) | uint64(hi)<<32
	return Int{val: int32(value), cell: numericCellRef(ref)}, nil
}
func (v integerArrayView) reference(i uint32, local *Int) (*Int, error) {
	if i >= v.count {
		return nil, arena.ErrBounds
	}
	if v.runtime == nil {
		if uint64(i) >= uint64(len(v.values)) {
			return nil, arena.ErrBounds
		}
		return &v.values[i], nil
	}
	a, e := v.at(i)
	if e != nil {
		return nil, e
	}
	*local = a
	return local, nil
}
func (m *IntegerManager) GCDManyIndexed(as integerArrayView, g *Int) error {
	if e := m.check(g); e != nil {
		return e
	}
	var a, b Int
	switch as.count {
	case 0:
		return m.Reset(g)
	case 1:
		pa, e := as.reference(0, &a)
		if e != nil {
			return e
		}
		if e = m.Copy(g, pa); e != nil {
			return e
		}
		return m.Abs(g)
	}
	pa, e := as.reference(0, &a)
	if e != nil {
		return e
	}
	pb, e := as.reference(1, &b)
	if e != nil {
		return e
	}
	if e = m.GCD(pa, pb, g); e != nil {
		return e
	}
	for i := uint32(2); i < as.count; i++ {
		if isOneInteger(g) {
			return nil
		}
		pa, e = as.reference(i, &a)
		if e != nil {
			return e
		}
		if e = m.GCD(g, pa, g); e != nil {
			return e
		}
	}
	return nil
}
func (m *IntegerModularManager) GCDManyIndexed(as integerArrayView, g *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.GCDManyIndexed(as, g); e != nil {
		return e
	}
	return m.assertNormalized(g)
}
