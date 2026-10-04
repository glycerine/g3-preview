// Source: src/math/polynomial/polynomial.cpp:2217-2245,3168-3180; src/util/buffer.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
)

// A source-reached borrowed pointer span. Inline and Runtime backing remain
// authoritative; this view owns neither monomial references nor an array copy.
type monomialArrayView struct {
	values         []Monomial
	vector         *polynomialVector
	inline         *[128]Monomial
	runtime        *memory.Runtime
	raw            memory.Block
	offset, stride uint64
	count          uint32
}

func (v monomialArrayView) at(m *MultivariatePolynomialCore, i uint32) (Monomial, error) {
	if v.vector != nil {
		return v.vector.monomial(m, i)
	}
	if i >= v.count {
		return Monomial{}, arena.ErrBounds
	}
	if v.runtime != nil {
		if v.runtime != m.memory {
			return Monomial{}, memory.ErrRuntimeOwner
		}
		if v.stride != 0 && uint64(i) > (math.MaxUint64-v.offset)/v.stride {
			return Monomial{}, arena.ErrBounds
		}
		o := v.offset + uint64(i)*v.stride
		if o > math.MaxUint64-16 {
			return Monomial{}, arena.ErrBounds
		}
		return skeletonReadMonomial(v.runtime, v.raw, o)
	}
	if v.inline != nil {
		if i >= 128 {
			return Monomial{}, arena.ErrBounds
		}
		return v.inline[i], nil
	}
	if uint64(i) >= uint64(len(v.values)) {
		return Monomial{}, arena.ErrBounds
	}
	return v.values[i], nil
}
