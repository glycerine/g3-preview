// Source: src/util/approx_set.h:43-158,162-233; src/util/approx_set.cpp:22-51.
// Source author: Leonardo de Moura (2007-03-02).
// Licensed under the MIT license; see LICENSE.txt.
package util

// UintApproxSet is the concrete unsigned-element, uint64-word specialization.
// The inline word owns no payload; predicates preserve the source collisions.
type UintApproxSet struct{ bits uint64 }

func NewUintApproxSet(element uint32) UintApproxSet {
	return UintApproxSet{uint64(1) << (element & 63)}
}
func UintApproxSetFrom(elements []uint32) UintApproxSet {
	var s UintApproxSet
	for _, e := range elements {
		s.Insert(e)
	}
	return s
}
func (s *UintApproxSet) Insert(e uint32)                    { s.bits |= uint64(1) << (e & 63) }
func (s UintApproxSet) MayContain(e uint32) bool            { return s.bits&(uint64(1)<<(e&63)) != 0 }
func (s UintApproxSet) MustNotContain(e uint32) bool        { return !s.MayContain(e) }
func (s UintApproxSet) Union(o UintApproxSet) UintApproxSet { return UintApproxSet{s.bits | o.bits} }
func (s UintApproxSet) Intersection(o UintApproxSet) UintApproxSet {
	return UintApproxSet{s.bits & o.bits}
}
func (s *UintApproxSet) UnionAssign(o UintApproxSet)           { s.bits |= o.bits }
func (s *UintApproxSet) IntersectAssign(o UintApproxSet)       { s.bits &= o.bits }
func (s *UintApproxSet) SubtractAssign(o UintApproxSet)        { s.bits &^= o.bits }
func (s UintApproxSet) Empty() bool                            { return s.bits == 0 }
func (s UintApproxSet) MustNotSubset(o UintApproxSet) bool     { return s.bits&^o.bits != 0 }
func (s UintApproxSet) MustNotSubsume(o UintApproxSet) bool    { return s.MustNotSubset(o) }
func (s UintApproxSet) MustNotEqual(o UintApproxSet) bool      { return s.bits != o.bits }
func (s UintApproxSet) MayEqual(o UintApproxSet) bool          { return s.bits == o.bits }
func (s UintApproxSet) Equivalent(o UintApproxSet) bool        { return s.bits == o.bits }
func (s UintApproxSet) ApproxSubset(o UintApproxSet) bool      { return o.Equivalent(s.Union(o)) }
func (s *UintApproxSet) Reset()                                { s.bits = 0 }
func (s UintApproxSet) EmptyIntersection(o UintApproxSet) bool { return s.Intersection(o).Empty() }

type UintApproxSetIterator struct {
	rest  uint64
	value uint32
}

func (i *UintApproxSetIterator) moveToNext() {
	for i.rest > 0 {
		if i.rest&1 != 0 {
			return
		}
		i.value++
		i.rest >>= 1
	}
}
func (s UintApproxSet) Begin() UintApproxSetIterator {
	i := UintApproxSetIterator{rest: s.bits}
	i.moveToNext()
	return i
}
func UintApproxSetEnd() UintApproxSetIterator                    { return UintApproxSetIterator{} }
func (i UintApproxSetIterator) Current() uint32                  { return i.value }
func (i *UintApproxSetIterator) Next()                           { i.value++; i.rest >>= 1; i.moveToNext() }
func (i *UintApproxSetIterator) PostNext() UintApproxSetIterator { old := *i; i.Next(); return old }

// Source comparison uses only the shifted remaining word, not the position.
func (i UintApproxSetIterator) Equal(o UintApproxSetIterator) bool { return i.rest == o.rest }

func (s UintApproxSet) Display(out *NumericTextStream) error {
	if e := out.AppendCString("{"); e != nil {
		return e
	}
	first := true
	word := s.bits
	for i := uint32(0); i < 64; i++ {
		if word&1 != 0 {
			if first {
				first = false
			} else {
				if e := out.AppendCString(", "); e != nil {
					return e
				}
			}
			if e := out.AppendUint32(i); e != nil {
				return e
			}
		}
		word >>= 1
	}
	return out.AppendCString("}")
}
func (s UintApproxSet) Size() uint32 {
	word := s.bits
	var count uint32
	for word > 0 {
		if word&1 != 0 {
			count++
		}
		word >>= 1
	}
	return count
}
