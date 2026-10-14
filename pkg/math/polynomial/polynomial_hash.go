// Source: src/math/polynomial/polynomial.cpp:2464-2480,7026-7032.
package polynomial

import uhash "github.com/glycerine/g3/pkg/util/hash"

// Hash computes the source polynomial hash after lexicographically sorting its
// terms. The source cache uses this value to bucket structurally equal values.
func (m *MultivariatePolynomialManager) Hash(p Polynomial) (uint32, error) {
	if e := m.ready(); e != nil {
		return 0, e
	}
	info, e := m.core.Info(p)
	if e != nil {
		return 0, e
	}
	if info.Size == 0 {
		return 31, nil
	}
	if e = m.LexSort(p); e != nil {
		return 0, e
	}
	children := make([]uint32, info.Size)
	for i := range children {
		mono, e := m.core.monomialCore(p, uint32(i))
		if e != nil {
			return 0, e
		}
		mi, e := m.core.monomials.Info(mono)
		if e != nil {
			return 0, e
		}
		coefficient, e := m.core.coefficientCore(p, uint32(i))
		if e != nil {
			return 0, e
		}
		valueHash, e := m.core.modular.Hash(&coefficient)
		if e != nil {
			return 0, e
		}
		children[i] = uhash.UintPair(mi.Hash, valueHash)
	}
	return uhash.Composite(17, info.Size, func(i uint32) uint32 { return children[i] }), nil
}
