// Licensed under the MIT license.
// Translation of sat::uint_set and literal_set in src/sat/sat_types.h.
package g3

// satUintSet owns scalar backing arrays. Assign copies; plain Go assignment
// aliases storage and is not the source copy operation. Iteration follows values
// in source order; membership never chooses an iteration order.
type satUintSet struct {
	inSet  []bool
	values []uint32
}

func (s *satUintSet) insert(v uint32) {
	n := uint64(v) + 1
	if n > uint64(^uint(0)>>1) || n > uint64(^uint32(0)) {
		panic("sat: set size overflow")
	}
	if n > uint64(len(s.inSet)) {
		old := len(s.inSet)
		if n > uint64(cap(s.inSet)) {
			a := make([]bool, int(n))
			copy(a, s.inSet)
			s.inSet = a
		} else {
			s.inSet = s.inSet[:int(n)]
			clear(s.inSet[old:])
		}
	}
	if s.inSet[v] {
		return
	}
	s.inSet[v] = true
	s.values = append(s.values, v)
}
func (s *satUintSet) contains(v uint32) bool { return uint64(v) < uint64(len(s.inSet)) && s.inSet[v] }
func (s *satUintSet) remove(v uint32) {
	if !s.contains(v) {
		return
	}
	s.inSet[v] = false
	i := 0
	for i < len(s.values) && s.values[i] != v {
		i++
	}
	if i == len(s.values) {
		panic("sat: membership/list inconsistency")
	}
	s.values[i] = s.values[len(s.values)-1]
	s.values = s.values[:len(s.values)-1]
}
func (s *satUintSet) assign(other *satUintSet) {
	if s == other {
		return
	}
	s.inSet = append(s.inSet[:0], other.inSet...)
	s.values = append(s.values[:0], other.values...)
}
func (s *satUintSet) empty() bool  { return len(s.values) == 0 }
func (s *satUintSet) size() uint32 { return uint32(len(s.values)) }
func (s *satUintSet) erase() uint32 {
	if s.empty() {
		panic("sat: erase from empty set")
	}
	v := s.values[len(s.values)-1]
	s.values = s.values[:len(s.values)-1]
	s.inSet[v] = false
	return v
}
func (s *satUintSet) reset()    { s.values = s.values[:0]; s.inSet = s.inSet[:0] }
func (s *satUintSet) finalize() { s.values = nil; s.inSet = nil }
func (s *satUintSet) intersect(other *satUintSet) {
	j := 0
	for _, v := range s.values {
		if other.contains(v) {
			s.values[j] = v
			j++
		} else {
			s.inSet[v] = false
		}
	}
	s.values = s.values[:j]
}
func (s *satUintSet) union(other *satUintSet) {
	for _, v := range other.values {
		s.insert(v)
	}
}

type satLiteralSet struct{ set satUintSet }

func satMakeLiteralSet(ls []satLiteral) satLiteralSet {
	var s satLiteralSet
	for _, l := range ls {
		s.insert(l)
	}
	return s
}
func (s *satLiteralSet) assignVector(ls []satLiteral) {
	s.reset()
	for _, l := range ls {
		s.insert(l)
	}
}
func (s *satLiteralSet) assign(other *satLiteralSet)    { s.set.assign(&other.set) }
func (s *satLiteralSet) insert(l satLiteral)            { s.set.insert(l.index()) }
func (s *satLiteralSet) remove(l satLiteral)            { s.set.remove(l.index()) }
func (s *satLiteralSet) pop() satLiteral                { return satLiteralFromUint(s.set.erase()) }
func (s *satLiteralSet) contains(l satLiteral) bool     { return s.set.contains(l.index()) }
func (s *satLiteralSet) empty() bool                    { return s.set.empty() }
func (s *satLiteralSet) size() uint32                   { return s.set.size() }
func (s *satLiteralSet) reset()                         { s.set.reset() }
func (s *satLiteralSet) finalize()                      { s.set.finalize() }
func (s *satLiteralSet) intersect(other *satLiteralSet) { s.set.intersect(&other.set) }
func (s *satLiteralSet) union(other *satLiteralSet)     { s.set.union(&other.set) }
func (s *satLiteralSet) toVector() []satLiteral {
	ls := make([]satLiteral, len(s.set.values))
	for i, v := range s.set.values {
		ls[i] = satLiteralFromUint(v)
	}
	return ls
}
