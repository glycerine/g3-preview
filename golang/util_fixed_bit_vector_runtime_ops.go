// Source: src/util/fixed_bit_vector.h; src/util/fixed_bit_vector.cpp.
// Licensed under the MIT license. Source authors: Nikolaj Bjorner, 2014-9-15;
// Leonardo de Moura, 2006-10-03.
package g3

import "io"

func (m *RuntimeFixedBitVectorManager) SetBool(v RuntimeFixedBitVector, i uint32, b bool) error {
	return m.SetBoolView(RuntimeFixedBitVectorView{manager: m, value: v}, i, b)
}
func (m *RuntimeFixedBitVectorManager) SetBit(v RuntimeFixedBitVector, i uint32) error {
	return m.SetView(RuntimeFixedBitVectorView{manager: m, value: v}, i)
}
func (m *RuntimeFixedBitVectorManager) Unset(v RuntimeFixedBitVector, i uint32) error {
	return m.UnsetView(RuntimeFixedBitVectorView{manager: m, value: v}, i)
}
func (m *RuntimeFixedBitVectorManager) ResetValue(v RuntimeFixedBitVector) error { return m.Fill0(v) }
func (m *RuntimeFixedBitVectorManager) Copy(dst, src RuntimeFixedBitVector) error {
	return m.CopyFrom(dst, m, src)
}
func (m *RuntimeFixedBitVectorManager) CopyFrom(dst RuntimeFixedBitVector, s *RuntimeFixedBitVectorManager, src RuntimeFixedBitVector) error {
	return m.CopyViews(RuntimeFixedBitVectorView{manager: m, value: dst}, RuntimeFixedBitVectorView{manager: s, value: src})
}
func (m *RuntimeFixedBitVectorManager) Fill0(v RuntimeFixedBitVector) error {
	return m.Fill0View(RuntimeFixedBitVectorView{manager: m, value: v})
}
func (m *RuntimeFixedBitVectorManager) Fill1(v RuntimeFixedBitVector) error {
	return m.Fill1View(RuntimeFixedBitVectorView{manager: m, value: v})
}
func (m *RuntimeFixedBitVectorManager) Union(dst, src RuntimeFixedBitVector) error {
	return m.UnionFrom(dst, m, src)
}
func (m *RuntimeFixedBitVectorManager) UnionFrom(dst RuntimeFixedBitVector, s *RuntimeFixedBitVectorManager, src RuntimeFixedBitVector) error {
	return m.UnionViews(RuntimeFixedBitVectorView{manager: m, value: dst}, RuntimeFixedBitVectorView{manager: s, value: src})
}
func (m *RuntimeFixedBitVectorManager) Intersect(dst, src RuntimeFixedBitVector) error {
	return m.IntersectFrom(dst, m, src)
}
func (m *RuntimeFixedBitVectorManager) IntersectFrom(dst RuntimeFixedBitVector, s *RuntimeFixedBitVectorManager, src RuntimeFixedBitVector) error {
	return m.IntersectViews(RuntimeFixedBitVectorView{manager: m, value: dst}, RuntimeFixedBitVectorView{manager: s, value: src})
}
func (m *RuntimeFixedBitVectorManager) Neg(v RuntimeFixedBitVector) error {
	return m.NegView(RuntimeFixedBitVectorView{manager: m, value: v})
}
func (m *RuntimeFixedBitVectorManager) LastWord(v RuntimeFixedBitVector) (uint32, error) {
	return m.LastWordView(RuntimeFixedBitVectorView{manager: m, value: v})
}
func (m *RuntimeFixedBitVectorManager) Equal(a, b RuntimeFixedBitVector) (bool, error) {
	return m.EqualFrom(a, m, b)
}
func (m *RuntimeFixedBitVectorManager) EqualFrom(a RuntimeFixedBitVector, s *RuntimeFixedBitVectorManager, b RuntimeFixedBitVector) (bool, error) {
	return m.EqualViews(RuntimeFixedBitVectorView{manager: m, value: a}, RuntimeFixedBitVectorView{manager: s, value: b})
}
func (m *RuntimeFixedBitVectorManager) Contains(a, b RuntimeFixedBitVector) (bool, error) {
	return m.ContainsFrom(a, m, b)
}
func (m *RuntimeFixedBitVectorManager) ContainsFrom(a RuntimeFixedBitVector, s *RuntimeFixedBitVectorManager, b RuntimeFixedBitVector) (bool, error) {
	return m.ContainsViews(RuntimeFixedBitVectorView{manager: m, value: a}, RuntimeFixedBitVectorView{manager: s, value: b})
}
func (m *RuntimeFixedBitVectorManager) SetRange(dst, src RuntimeFixedBitVector, hi, lo uint32) error {
	return m.SetRangeFrom(dst, m, src, hi, lo)
}
func (m *RuntimeFixedBitVectorManager) SetRangeFrom(dst RuntimeFixedBitVector, s *RuntimeFixedBitVectorManager, src RuntimeFixedBitVector, hi, lo uint32) error {
	return m.SetRangeViews(RuntimeFixedBitVectorView{manager: m, value: dst}, RuntimeFixedBitVectorView{manager: s, value: src}, hi, lo)
}
func (m *RuntimeFixedBitVectorManager) Hash(v RuntimeFixedBitVector) (uint32, error) {
	return m.HashView(RuntimeFixedBitVectorView{manager: m, value: v})
}
func (m *RuntimeFixedBitVectorManager) Display(w io.Writer, v RuntimeFixedBitVector) error {
	return m.DisplayView(w, RuntimeFixedBitVectorView{manager: m, value: v})
}
