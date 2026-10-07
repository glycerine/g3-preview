// Source: src/math/polynomial/polynomial.cpp:1315-1366,1440-1460,2462-2464,7022-7024;
// src/util/permutation.h:57-91; src/util/vector.h:60-98,422-437.
// Licensed under the MIT license. Author: Leonardo de Moura.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

// Runtime slots encode concrete child bindings; the charged directory contains
// the actual unsigned-vector owners. It never duplicates numeral storage.
func (b *polynomialImpBuckets) reserve(n uint32) error {
	m := b.owner
	if b.data.interrupted || b.data.pending != (util.Block{}) || b.data.old != (util.Block{}) {
		return ErrPolynomialPending
	}
	size, e := b.data.size(m)
	if e != nil {
		return e
	}
	if n <= size {
		return nil
	}
	for {
		cap, e := b.data.capacity(m)
		if e != nil {
			return e
		}
		if n <= cap {
			break
		}
		if e = b.expandClasses(); e != nil {
			return e
		}
	}
	if uint64(n) > uint64(cap(b.children)) {
		stride := uint64(reflect.TypeOf(polynomialVector{}).Size())
		if e = m.integer.Budget().Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]polynomialVector, len(b.children), int(n))
		copy(next, b.children)
		m.integer.Budget().Release(uint64(cap(b.children)) * stride)
		b.children = next
	}
	if e = b.data.publishSize(m, n); e != nil {
		return e
	}
	for b.data.initialized < n {
		i := b.data.initialized
		// Source placement constructs a default unsigned_vector at this slot.
		if e = b.data.write64(m, 8+uint64(i)*8, uint64(i)+1); e != nil {
			return e
		}
		b.children = append(b.children, polynomialVector{width: 4})
		b.data.initialized++
	}
	return nil
}
func (b *polynomialImpBuckets) expandClasses() error {
	m := b.owner
	v := &b.data
	if polynomialSourceDebug && v.raw != (util.Block{}) {
		cap, e := v.capacity(m)
		if e != nil {
			return e
		}
		if cap == 0 {
			return ErrPolynomialAssertion
		}
	}
	oldCap, e := v.capacity(m)
	if e != nil {
		return e
	}
	n := uint32(2)
	if v.raw != (util.Block{}) {
		n = (3*oldCap + 1) >> 1
	}
	bytes := uint32(8*n + 8)
	oldBytes := uint32(8*oldCap + 8)
	if v.raw != (util.Block{}) && (n <= oldCap || bytes <= oldBytes) {
		return util.Failure{Kind: util.FailureVectorOverflow}
	}
	v.pending, e = m.memory.Allocate(uint64(bytes))
	if e != nil {
		return e
	}
	if v.raw == (util.Block{}) {
		if e = m.memory.WriteUint32(v.pending, 0, n); e != nil {
			return e
		}
		if e = m.memory.WriteUint32(v.pending, 4, 0); e != nil {
			return e
		}
		v.raw = v.pending
		v.pending = util.Block{}
		v.initialized = 0
		return nil
	}
	size, e := v.size(m)
	if e != nil {
		return e
	}
	if e = m.memory.WriteUint32(v.pending, 4, size); e != nil {
		return e
	}
	v.old = v.raw
	v.oldInitialized = v.initialized
	v.raw = v.pending
	v.pending = util.Block{}
	v.initialized = 0
	v.moved = 0
	v.interrupted = true
	old := polynomialVector{raw: v.old, width: 8}
	for v.moved < size {
		id, e := old.read64(m, 8+uint64(v.moved)*8)
		if e != nil {
			return e
		}
		if e = v.write64(m, 8+uint64(v.moved)*8, id); e != nil {
			return e
		}
		// unsigned_vector's move constructor leaves its old pointer null.
		if e = old.write64(m, 8+uint64(v.moved)*8, 0); e != nil {
			return e
		}
		v.moved++
		v.initialized++
	}
	if v.old != (util.Block{}) {
		if e = m.memory.Deallocate(&v.old); e != nil {
			return e
		}
	}
	if e = m.memory.WriteUint32(v.raw, 0, n); e != nil {
		return e
	}
	v.interrupted = false
	return nil
}
func (b *polynomialImpBuckets) at(i uint32) (*polynomialVector, error) {
	n, e := b.data.size(b.owner)
	if e != nil {
		return nil, e
	}
	if polynomialSourceDebug && i >= n {
		return nil, ErrPolynomialAssertion
	}
	id, e := b.data.read64(b.owner, 8+uint64(i)*8)
	if e != nil {
		return nil, e
	}
	if id == 0 || id > uint64(len(b.children)) {
		return nil, ErrPolynomialPartial
	}
	return &b.children[id-1], nil
}
func polynomialUnsignedAppend(m *MultivariatePolynomialCore, v *polynomialVector, x uint32) error {
	i, e := v.appendSlot(m)
	if e != nil {
		return e
	}
	if e = m.memory.WriteUint32(v.raw, 8+uint64(i)*4, x); e != nil {
		return e
	}
	return v.publishSize(m, i+1)
}
func polynomialUnsignedRead(m *MultivariatePolynomialCore, v *polynomialVector, i uint32) (uint32, error) {
	if polynomialSourceDebug {
		n, e := v.size(m)
		if e != nil {
			return 0, e
		}
		if i >= n {
			return 0, ErrPolynomialAssertion
		}
	}
	return m.memory.ReadUint32(v.raw, 8+uint64(i)*4)
}
func (m *MultivariatePolynomialManager) lexMaximumBelow(p Polynomial, start, end, x uint32) (uint32, error) {
	max := ^uint32(0)
	ms, e := m.core.captureMonomialPointer(p, 0)
	if e != nil {
		return 0, e
	}
	for i := start; i < end; i++ {
		a, e := ms.read(i)
		if e != nil {
			return 0, e
		}
		y, e := m.core.monomials.MaxSmallerThan(a, x)
		if e != nil {
			return 0, e
		}
		if y != ^uint32(0) && (max == ^uint32(0) || y > max) {
			max = y
		}
	}
	return max, nil
}
func (m *MultivariatePolynomialManager) lexRange(p Polynomial, start, end, x uint32) error {
	if polynomialSourceDebug && end <= start {
		return ErrPolynomialAssertion
	}
	c := m.core
	ms, e := c.captureMonomialPointer(p, 0)
	if e != nil {
		return e
	}
	maxDegree := uint32(0)
	for i, j := start, uint32(0); i < end; i, j = i+1, j+1 {
		a, e := ms.read(i)
		if e != nil {
			return e
		}
		d, e := c.monomials.DegreeOf(a, x)
		if e != nil {
			return e
		}
		if e = m.lexBuckets.reserve(d + 1); e != nil {
			return e
		}
		b, e := m.lexBuckets.at(d)
		if e != nil {
			return e
		}
		if e = polynomialUnsignedAppend(c, b, j); e != nil {
			return e
		}
		if d > maxDegree {
			maxDegree = d
		}
	}
	if e := m.lexPermutation.reset(c); e != nil {
		return e
	}
	for i := maxDegree + 1; i > 0; {
		i--
		b, e := m.lexBuckets.at(i)
		if e != nil {
			return e
		}
		for j := uint32(0); ; j++ {
			n, e := b.size(c)
			if e != nil {
				return e
			}
			if j >= n {
				break
			}
			v, e := polynomialUnsignedRead(c, b, j)
			if e != nil {
				return e
			}
			if e = polynomialUnsignedAppend(c, &m.lexPermutation, v); e != nil {
				return e
			}
		}
		if e = b.reset(c); e != nil {
			return e
		}
	}
	if polynomialSourceDebug {
		n, e := m.lexPermutation.size(c)
		if e != nil {
			return e
		}
		if n != end-start {
			return ErrPolynomialAssertion
		}
	}
	n, e := m.lexPermutation.size(c)
	if e != nil {
		return e
	}
	if e = m.lexApplyPermutation(p, start, n, true); e != nil {
		return e
	}
	n, e = m.lexPermutation.size(c)
	if e != nil {
		return e
	}
	if e = m.lexApplyPermutation(p, start, n, false); e != nil {
		return e
	}
	for i := start; i < end; {
		a, e := ms.read(i)
		if e != nil {
			return e
		}
		d, e := c.monomials.DegreeOf(a, x)
		if e != nil {
			return e
		}
		if d == 0 {
			y, e := m.lexMaximumBelow(p, i, end, x)
			if e != nil {
				return e
			}
			if y != ^uint32(0) {
				return m.lexRange(p, i, end, y)
			}
			return nil
		}
		j := i + 1
		for ; j < end; j++ {
			a, e := ms.read(j)
			if e != nil {
				return e
			}
			next, e := c.monomials.DegreeOf(a, x)
			if e != nil {
				return e
			}
			if polynomialSourceDebug && next > d {
				return ErrPolynomialAssertion
			}
			if next < d {
				break
			}
		}
		if polynomialSourceDebug && j != end {
			a, e := ms.read(j)
			if e != nil {
				return e
			}
			next, e := c.monomials.DegreeOf(a, x)
			if e != nil {
				return e
			}
			if next >= d {
				return ErrPolynomialAssertion
			}
		}
		if j > i+1 {
			y, e := m.lexMaximumBelow(p, i, j, x)
			if e != nil {
				return e
			}
			if y != ^uint32(0) {
				if e = m.lexRange(p, i, j, y); e != nil {
					return e
				}
			}
		}
		i = j
	}
	return nil
}
func (m *MultivariatePolynomialManager) LexSort(p Polynomial) error {
	if e := m.ready(); e != nil {
		return e
	}
	flags, e := m.core.readWord(p, 4)
	if e != nil {
		return e
	}
	if flags>>31 != 0 {
		return nil
	}
	n, e := m.core.readWord(p, 8)
	if e != nil {
		return e
	}
	if n <= 1 {
		return m.core.writeWord(p, 4, flags|0x80000000)
	}
	a, e := m.core.monomialCore(p, 0)
	if e != nil {
		return e
	}
	x, e := m.core.monomials.MaxVariable(a)
	if e != nil {
		return e
	}
	if m.lexPending {
		return ErrPolynomialPending
	}
	if m.lexGeneration == ^uint64(0) {
		return ErrPolynomialPending
	}
	m.lexGeneration++
	if e = m.lexRange(p, 0, n, x); e != nil {
		m.lexPending = true
		m.lexCause = e
		return e
	}
	if e = m.core.writeWord(p, 4, flags|0x80000000); e != nil {
		return e
	}
	if polynomialSourceDebug {
		ms, e := m.core.captureMonomialPointer(p, 0)
		if e != nil {
			return e
		}
		for i := uint32(0); i < n-1; i++ {
			a, e := ms.read(i)
			if e != nil {
				return e
			}
			b, e := ms.read(i + 1)
			if e != nil {
				return e
			}
			order, e := m.core.monomials.Lex(a, b)
			if e != nil {
				return e
			}
			if order <= 0 {
				return ErrPolynomialAssertion
			}
		}
	}
	return nil
}

// CleanupLexSort is an explicit owning abort of actual primitive scratch roots.
// It does not undo polynomial publications or reexecute the stopped source call.
func (m *MultivariatePolynomialManager) CleanupLexSort() error {
	if e := m.ready(); e != nil {
		return e
	}
	return m.cleanupLexSort()
}
func (m *MultivariatePolynomialManager) cleanupLexSort() error {
	if !m.lexPending {
		return nil
	}
	if e := m.lexBuckets.close(); e != nil {
		return e
	}
	if e := m.lexPermutation.close(m.core); e != nil {
		return e
	}
	m.lexBuckets.data = polynomialVector{width: 8}
	m.lexBuckets.cursor = 0
	m.lexPermutation = polynomialVector{width: 4}
	m.lexPending = false
	return nil
}
