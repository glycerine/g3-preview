// Source: src/util/hash.cpp:26-102.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-09-11.
package utilhash

import "github.com/glycerine/g3/golang/memory"

// RuntimeBytes reads actual little-endian raw words and source signed-char tails.
// The last eleven bytes are read in descending switch fallthrough order.
func RuntimeBytes(r *memory.Runtime, block memory.Block, offset uint64, n, seed uint32) (uint32, error) {
	a, b, c := uint32(0x9e3779b9), uint32(0x9e3779b9), seed
	i := uint32(0)
	for n-i >= 12 {
		x, e := r.ReadUint32(block, offset+uint64(i))
		if e != nil {
			return 0, e
		}
		a += x
		x, e = r.ReadUint32(block, offset+uint64(i)+4)
		if e != nil {
			return 0, e
		}
		b += x
		x, e = r.ReadUint32(block, offset+uint64(i)+8)
		if e != nil {
			return 0, e
		}
		c += x
		a, b, c = mix(a, b, c)
		i += 12
	}
	c += n
	for j := n - i; j > 0; {
		j--
		x, e := r.Read(block, offset+uint64(i)+uint64(j))
		if e != nil {
			return 0, e
		}
		v := uint32(int32(int8(x)))
		if j < 4 {
			a += v << (j * 8)
		} else if j < 8 {
			b += v << ((j - 4) * 8)
		} else {
			c += v << ((j - 7) * 8)
		}
	}
	return Mix(a, b, c), nil
}
