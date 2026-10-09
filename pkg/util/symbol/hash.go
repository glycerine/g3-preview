// Source: src/util/hash.cpp; src/util/symbol.cpp:48-52.
package symbol

import "encoding/binary"

func mix(a, b, c uint32) (uint32, uint32, uint32) {
	a -= b
	a -= c
	a ^= c >> 13
	b -= c
	b -= a
	b ^= a << 8
	c -= a
	c -= b
	c ^= b >> 13
	a -= b
	a -= c
	a ^= c >> 12
	b -= c
	b -= a
	b ^= a << 16
	c -= a
	c -= b
	c ^= b >> 5
	a -= b
	a -= c
	a ^= c >> 3
	b -= c
	b -= a
	b ^= a << 10
	c -= a
	c -= b
	c ^= b >> 15
	return a, b, c
}

func spellingHash(s string, init uint32) uint32 {
	n := uint32(len(s))
	word := func(i uint32) uint32 { return binary.LittleEndian.Uint32([]byte(s[i : i+4])) }
	character := func(i uint32) uint32 { return uint32(int32(int8(s[i]))) }
	a, b, c := uint32(0x9e3779b9), uint32(0x9e3779b9), init
	i := uint32(0)
	for n-i >= 12 {
		a += word(i)
		b += word(i + 4)
		c += word(i + 8)
		a, b, c = mix(a, b, c)
		i += 12
	}
	c += n
	for j := uint32(0); j < n-i; j++ {
		v := character(i + j)
		switch {
		case j < 4:
			a += v << (j * 8)
		case j < 8:
			b += v << ((j - 4) * 8)
		default:
			c += v << ((j - 8 + 1) * 8)
		}
	}
	_, _, c = mix(a, b, c)
	return c
}
