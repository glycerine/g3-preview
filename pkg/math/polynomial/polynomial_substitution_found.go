// Source: src/math/polynomial/polynomial.cpp:3185,6253-6298; src/util/vector.h:59-102,407-442.
// Licensed under the MIT license.
// Author: Leonardo de Moura.
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Actual persistent svector<bool>: eight-byte cap/size prefix followed by
// one initialized byte per reached bool constructor, not an unsigned-vector
// surrogate. The cold continuation fields are charged by the parent header.
type polynomialFoundVariables struct {
	runtime                            *util.Runtime
	raw, pending                       util.Block
	interrupted, finalizing, finalized bool
	sourceCause                        error
}

func (v *polynomialFoundVariables) size() (uint32, error) {
	if v.raw == (util.Block{}) {
		return 0, nil
	}
	return v.runtime.ReadUint32(v.raw, 4)
}
func (v *polynomialFoundVariables) capacity() (uint32, error) {
	if v.raw == (util.Block{}) {
		return 0, nil
	}
	return v.runtime.ReadUint32(v.raw, 0)
}
func (v *polynomialFoundVariables) expand() error {
	if v.pending != (util.Block{}) || v.interrupted {
		return ErrPolynomialPending
	}
	old, e := v.capacity()
	if e != nil {
		return e
	}
	if v.raw == (util.Block{}) {
		v.pending, e = v.runtime.Allocate(10)
		if e != nil {
			return e
		}
		if e = v.runtime.WriteUint32(v.pending, 0, 2); e != nil {
			return e
		}
		if e = v.runtime.WriteUint32(v.pending, 4, 0); e != nil {
			return e
		}
		v.raw = v.pending
		v.pending = util.Block{}
		return nil
	}
	if polynomialSourceDebug && old == 0 {
		return ErrPolynomialAssertion
	}
	nc := (3*old + 1) >> 1
	oldBytes, newBytes := old+8, nc+8
	if nc <= old || newBytes <= oldBytes {
		return util.Failure{Kind: util.FailureVectorOverflow}
	}
	before := v.raw
	if e = v.runtime.Reallocate(&v.raw, uint64(newBytes)); e != nil {
		if before != v.raw {
			v.interrupted = true
		}
		return e
	}
	if e = v.runtime.WriteUint32(v.raw, 0, nc); e != nil {
		v.interrupted = true
		return e
	}
	return nil
}
func (v *polynomialFoundVariables) reserve(n uint32) error {
	if v.finalizing {
		return arena.ErrClosed
	}
	if v.pending != (util.Block{}) || v.interrupted {
		return ErrPolynomialPending
	}
	old, e := v.size()
	if e != nil {
		return e
	}
	if n <= old {
		return nil
	}
	for {
		c, e := v.capacity()
		if e != nil {
			return e
		}
		if n <= c {
			break
		}
		if e = v.expand(); e != nil {
			return e
		}
	}
	if e = v.runtime.WriteUint32(v.raw, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = v.runtime.Write(v.raw, 8+uint64(i), 0); e != nil {
			v.interrupted = true
			return e
		}
	}
	return nil
}
func (v *polynomialFoundVariables) at(x uint32) (bool, error) {
	if polynomialSourceDebug {
		n, e := v.size()
		if e != nil {
			return false, e
		}
		if x >= n {
			return false, ErrPolynomialAssertion
		}
	}
	b, e := v.runtime.Read(v.raw, 8+uint64(x))
	return b != 0, e
}
func (v *polynomialFoundVariables) put(x uint32, b bool) error {
	if polynomialSourceDebug {
		n, e := v.size()
		if e != nil {
			return e
		}
		if x >= n {
			return ErrPolynomialAssertion
		}
	}
	val := byte(0)
	if b {
		val = 1
	}
	return v.runtime.Write(v.raw, 8+uint64(x), val)
}
func (v *polynomialFoundVariables) sourceFinalize() error {
	if v.sourceCause != nil {
		return v.sourceCause
	}
	if v.finalized {
		return nil
	}
	v.finalizing = true
	if v.raw != (util.Block{}) {
		if e := v.runtime.Deallocate(&v.raw); e != nil {
			v.sourceCause = e
			return e
		}
	}
	v.finalized = true
	return nil
}
func (v *polynomialFoundVariables) close() error {
	if v.raw != (util.Block{}) {
		if e := v.runtime.Deallocate(&v.raw); e != nil {
			return e
		}
	}
	if v.pending != (util.Block{}) {
		if e := v.runtime.Deallocate(&v.pending); e != nil {
			return e
		}
	}
	v.finalizing = true
	v.finalized = true
	return nil
}
