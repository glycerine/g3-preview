// Copyright (c) 2011 Microsoft Corporation
// Translated from src/util/mpn.h and src/util/mpn.cpp.
// Original author: Christoph Wintersteiger, 2011-11-16.

// Package mpnum contains the fork's independent exact numerical algorithms.
package mpnum

import "strconv"

// NaturalManager implements mpn_manager with little-endian 32-bit limbs.
// Operands and destinations are borrowed caller-owned buffers, never retained.
// The four slices are reusable, manager-owned scratch, not persistent numerals.
// A manager may not be copied after use. Callers serialize access to Div, which
// replaces the source manager's OpenMP critical section with owner confinement.
// Compare/Add/Sub/Mul/String do not modify manager scratch.
type NaturalManager struct{ u, v, ms, ab []uint32 }

func limb(a []uint32, i int) uint32 {
	if i < len(a) {
		return a[i]
	}
	return 0
}

// Compare translates mpn_manager::compare; leading zero limbs are allowed.
func (*NaturalManager) Compare(a, b []uint32) int {
	for j := max(len(a), len(b)) - 1; j >= 0; j-- {
		u, v := limb(a, j), limb(b, j)
		if u > v {
			return 1
		}
		if u < v {
			return -1
		}
	}
	return 0
}

// Add translates Knuth's Algorithm A. c must have exactly max lengths + 1
// limbs; at least one input is nonempty. Exact input/output aliasing is allowed;
// shifted overlapping slices are outside the source contract. The returned
// normalized size is at least one, including for zero.
func (*NaturalManager) Add(a, b, c []uint32) int {
	n := max(len(a), len(b))
	if n == 0 || len(c) != n+1 {
		panic("mpnum: invalid addition buffer")
	}
	var k uint32
	for j := 0; j < n; j++ {
		u, v := limb(a, j), limb(b, j)
		r := u + v
		c1 := r < u
		c[j] = r + k
		c2 := c[j] < r
		k = 0
		if c1 || c2 {
			k = 1
		}
	}
	c[n] = k
	size := n + 1
	for size > 1 && c[size-1] == 0 {
		size--
	}
	return size
}

// Sub translates Knuth's Algorithm S. c has max input lengths limbs; it
// receives subtraction modulo 2^(32*len(c)). Return 1 denotes a borrow.
// Exact input/output aliasing is allowed, shifted overlap is not.
func (*NaturalManager) Sub(a, b, c []uint32) uint32 {
	n := max(len(a), len(b))
	if len(c) != n {
		panic("mpnum: invalid subtraction buffer")
	}
	var k uint32
	for j := 0; j < n; j++ {
		u, v := limb(a, j), limb(b, j)
		r := u - v
		c1 := r > u
		c[j] = r - k
		c2 := c[j] > r
		k = 0
		if c1 || c2 {
			k = 1
		}
	}
	return k
}

// Mul translates Knuth's Algorithm M. c has len(a)+len(b) limbs and must
// not overlap either input (mpz handles its own aliasing through scratch).
func (*NaturalManager) Mul(a, b, c []uint32) {
	if len(c) != len(a)+len(b) {
		panic("mpnum: invalid multiplication buffer")
	}
	for i := range a {
		c[i] = 0
	}
	for j, v := range b {
		if v == 0 {
			c[j+len(a)] = 0
			continue
		}
		var k uint32
		for i, u := range a {
			t := uint64(u)*uint64(v) + uint64(c[i+j]) + uint64(k)
			c[i+j] = uint32(t)
			k = uint32(t >> 32)
		}
		c[j+len(a)] = k
	}
}

func resize(a []uint32, n int) []uint32 {
	if cap(a) < n {
		return make([]uint32, n)
	}
	return a[:n]
}

// Div translates mpn_manager::div, including its boolean return (false in
// single-limb and smaller-leading-limb fast paths, true after normalization).
// The denominator is nonempty with a nonzero high limb, len(a)>=len(b),
// q has len(a)-len(b)+1 limbs and r has len(b). Output buffers must not
// overlap each other or inputs. The source's unsigned-underflow branch for
// shorter numerators is excluded by an explicit precondition; mpz handles
// that case before calling mpn. These checks also replace source assertions.
func (m *NaturalManager) Div(a, b, q, r []uint32) bool {
	if len(b) == 0 || len(a) < len(b) || b[len(b)-1] == 0 || len(q) != len(a)-len(b)+1 || len(r) != len(b) {
		panic("mpnum: invalid division operands or buffers")
	}
	if len(a) == 1 && len(b) == 1 {
		q[0] = a[0] / b[0]
		r[0] = a[0] % b[0]
		return false
	}
	if len(a) == len(b) && a[len(a)-1] < b[len(b)-1] {
		q[0] = 0
		copy(r, a)
		return false
	}
	d := m.normalizeDivision(a, b)
	if len(b) == 1 {
		divideOne(m.u, m.v[0], q)
	} else {
		m.divideMany(q)
	}
	unnormalizeDivision(m.u, len(m.v), d, r)
	return true
}

// normalizeDivision translates div_normalize, shifting the high denominator
// limb until its top bit is set and extending the numerator with one limb.
func (m *NaturalManager) normalizeDivision(a, b []uint32) uint {
	var d uint
	for b[len(b)-1]<<d&0x80000000 == 0 {
		d++
	}
	m.u = resize(m.u, len(a)+1)
	m.v = resize(m.v, len(b))
	if d == 0 {
		m.u[len(a)] = 0
		copy(m.u, a)
		copy(m.v, b)
	} else {
		m.u[len(a)] = a[len(a)-1] >> (32 - d)
		for i := len(a) - 1; i > 0; i-- {
			m.u[i] = a[i]<<d | a[i-1]>>(32-d)
		}
		m.u[0] = a[0] << d
		for i := len(b) - 1; i > 0; i-- {
			m.v[i] = b[i]<<d | b[i-1]>>(32-d)
		}
		m.v[0] = b[0] << d
	}
	return d
}

// unnormalizeDivision translates div_unnormalize.
func unnormalizeDivision(a []uint32, n int, d uint, r []uint32) {
	if d == 0 {
		copy(r, a[:n])
		return
	}
	mask := uint32(1)<<d - 1
	for i := 0; i < n-1; i++ {
		r[i] = a[i]>>d | (a[i+1]&mask)<<(32-d)
	}
	r[n-1] = a[n-1] >> d
}

// divideOne translates div_1; the normalized numerator is updated to remainder.
func divideOne(a []uint32, b uint32, q []uint32) {
	for j := len(a) - 1; j > 0; j-- {
		temp := uint64(a[j])<<32 | uint64(a[j-1])
		qh := temp / uint64(b)
		if qh >= uint64(1)<<32 {
			panic("mpnum: unnormalized single-limb division")
		}
		ms := temp - qh*uint64(b)
		borrow := ms > temp
		a[j-1] = uint32(ms)
		a[j] = uint32(ms >> 32)
		q[j-1] = uint32(qh)
		if borrow {
			q[j-1]--
			a[j] = a[j-1] + b
		}
	}
}

// divideMany translates div_n, Knuth's Algorithm D, including quotient
// estimation correction and the subtract/add-back branch.
func (m *NaturalManager) divideMany(q []uint32) {
	n := len(m.v)
	count := len(m.u) - n
	m.ms = resize(m.ms, n+1)
	const base = uint64(1) << 32
	for j := count - 1; j >= 0; j-- {
		temp := uint64(m.u[j+n])<<32 | uint64(m.u[j+n-1])
		qh := temp / uint64(m.v[n-1])
		rh := temp % uint64(m.v[n-1])
		for {
			if qh < base && qh*uint64(m.v[n-2]) <= (rh<<32)+uint64(m.u[j+n-2]) {
				break
			}
			qh--
			rh += uint64(m.v[n-1])
			if rh >= base {
				break
			}
		}
		if qh >= base {
			panic("mpnum: invalid quotient estimate")
		}
		small := uint32(qh)
		m.Mul([]uint32{small}, m.v, m.ms)
		borrow := m.Sub(m.u[j:j+n+1], m.ms, m.u[j:j+n+1])
		q[j] = small
		if borrow != 0 {
			q[j]--
			m.ab = resize(m.ab, n+2)
			m.Add(m.v, m.u[j:j+n+1], m.ab)
			copy(m.u[j:j+n+1], m.ab[:n+1])
		}
	}
}

// String translates to_string's repeated division by ten, using local scratch
// as in the source. Inputs must be nonempty and either one limb or have a
// nonzero high limb. Go owns/grows the output instead of borrowing an unchecked
// char buffer. Source one-limb snprintf becomes strconv's unsigned formatting.
func (*NaturalManager) String(a []uint32) string {
	if len(a) == 0 || len(a) > 1 && a[len(a)-1] == 0 {
		panic("mpnum: unnormalized decimal input")
	}
	if len(a) == 1 {
		return strconv.FormatUint(uint64(a[0]), 10)
	}
	temp := append([]uint32(nil), a...)
	local := new(NaturalManager)
	var digits []byte
	rem := make([]uint32, 1)
	for len(temp) > 0 && (len(temp) > 1 || temp[0] != 0) {
		d := local.normalizeDivision(temp, []uint32{10})
		divideOne(local.u, local.v[0], temp)
		unnormalizeDivision(local.u, 1, d, rem)
		digits = append(digits, '0'+byte(rem[0]))
		for len(temp) > 0 && temp[len(temp)-1] == 0 {
			temp = temp[:len(temp)-1]
		}
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
