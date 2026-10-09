// Source: src/util/permutation.h:59-111.
package util

import (
	"errors"
	"math"
)

var ErrPermutationBounds = errors.New("permutation: index out of bounds")
var ErrPermutationAssertion = errors.New("permutation: source assertion")

func permutationRead(words []uint32, i uint32) (uint32, error) {
	if uint64(i) >= uint64(len(words)) {
		return 0, ErrPermutationBounds
	}
	return words[i], nil
}

func permutationWrite(words []uint32, i, value uint32) error {
	if uint64(i) >= uint64(len(words)) {
		return ErrPermutationBounds
	}
	words[i] = value
	return nil
}

// ApplyPermutationCoreUint32 specializes the source cycle algorithm for actual
// borrowed uint32 data. It marks the caller's mapping in place and allocates no
// scratch storage. Signed source counts/markers use the explicit int32 profile.
func ApplyPermutationCoreUint32(size uint32, data, p []uint32) error {
	for i := int32(0); i < int32(size); i++ {
		v, e := permutationRead(p, uint32(i))
		if e != nil {
			return e
		}
		if int32(v) < 0 {
			continue
		}
		j := i
		for {
			if sourceDebug && j < 0 {
				return ErrPermutationAssertion
			}
			v, e = permutationRead(p, uint32(j))
			if e != nil {
				return e
			}
			pj := int32(v)
			if sourceDebug && (pj < 0 || pj >= int32(size)) {
				return ErrPermutationAssertion
			}
			// Native signed negation of INT_MIN is undefined. Refuse only at
			// that reached operation, before its nonexistent defined result.
			if pj == math.MinInt32 {
				return ErrPermutationBounds
			}
			if e = permutationWrite(p, uint32(j), uint32(-pj-1)); e != nil {
				return e
			}
			if pj == i {
				break
			}
			a, e := permutationRead(data, uint32(j))
			if e != nil {
				return e
			}
			b, e := permutationRead(data, uint32(pj))
			if e != nil {
				return e
			}
			if e = permutationWrite(data, uint32(j), b); e != nil {
				return e
			}
			if e = permutationWrite(data, uint32(pj), a); e != nil {
				return e
			}
			j = pj
		}
	}
	return nil
}

// ApplyPermutationUint32 restores markers only after its destructive child
// completes. A stopped child preserves actual completed marks and swaps.
func ApplyPermutationUint32(size uint32, data, p []uint32) error {
	if e := ApplyPermutationCoreUint32(size, data, p); e != nil {
		return e
	}
	for i := uint32(0); i < size; i++ {
		v, e := permutationRead(p, i)
		if e != nil {
			return e
		}
		if int32(v) == math.MinInt32 {
			return ErrPermutationBounds
		}
		if e = permutationWrite(p, i, uint32(-int32(v)-1)); e != nil {
			return e
		}
	}
	return nil
}
