// Source: src/util/hash.cpp:26-102; src/util/fixed_bit_vector.h:32-69.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
	uhash "github.com/glycerine/g3/pkg/util/hash"
)

// Concrete call-local readers own no storage. The inline variant addresses the
// real source cold m_0 word and its checked initialized-byte mask.

type runtimeHashSource struct {
	runtime         *Runtime
	region          *arena.BytesRegion
	regionRef       arena.BytesRegionRef
	regionMode      bool
	block           Block
	offset, extent  uint64
	bounded, inline bool
	word            *uint32
	mask            *uint8
}

func (s runtimeHashSource) byteAt(i uint64) (byte, error) {
	if s.bounded && i >= s.extent || s.inline && i >= 4 {
		return 0, arena.ErrBounds
	}
	if s.inline {
		if s.word == nil || s.mask == nil {
			return 0, ErrRuntimeOwner
		}
		if *s.mask&(1<<i) == 0 {
			return 0, ErrRuntimeUninitialized
		}
		return byte(*s.word >> (8 * i)), nil
	}
	if s.regionMode {
		offset := s.offset + i
		if offset < s.offset || offset > uint64(^uint32(0)) {
			return 0, arena.ErrBounds
		}
		if s.region == nil {
			return 0, arena.ErrInvalidSlot
		}
		return s.region.Read(s.regionRef, uint32(offset))
	}
	return s.runtime.Read(s.block, s.offset+i)
}
func (s runtimeHashSource) wordAt(i uint64) (uint32, error) {
	if s.bounded && (i > s.extent || s.extent-i < 4) || s.inline && i != 0 {
		return 0, arena.ErrBounds
	}
	if s.inline {
		if s.word == nil || s.mask == nil {
			return 0, ErrRuntimeOwner
		}
		if *s.mask != 0xF {
			return 0, ErrRuntimeUninitialized
		}
		return *s.word, nil
	}
	if s.regionMode {
		var v uint32
		for j := uint64(0); j < 4; j++ {
			b, e := s.byteAt(i + j)
			if e != nil {
				return 0, e
			}
			v |= uint32(b) << (8 * j)
		}
		return v, nil
	}
	return s.runtime.ReadUint32(s.block, s.offset+i)
}
func RuntimeBytesBounded(r *Runtime, block Block, offset, extent uint64, n, seed uint32) (uint32, error) {
	return runtimeHashBytes(runtimeHashSource{runtime: r, block: block, offset: offset, extent: extent, bounded: true}, n, seed)
}
func InlineWordBytes(word *uint32, initialized *uint8, extent uint64, n, seed uint32) (uint32, error) {
	return runtimeHashBytes(runtimeHashSource{word: word, mask: initialized, extent: extent, bounded: true, inline: true}, n, seed)
}
func runtimeHashBytes(s runtimeHashSource, n, seed uint32) (uint32, error) {
	a, b, c := uint32(0x9e3779b9), uint32(0x9e3779b9), seed
	i := uint32(0)
	for n-i >= 12 {
		x, e := s.wordAt(uint64(i))
		if e != nil {
			return 0, e
		}
		a += x
		x, e = s.wordAt(uint64(i) + 4)
		if e != nil {
			return 0, e
		}
		b += x
		x, e = s.wordAt(uint64(i) + 8)
		if e != nil {
			return 0, e
		}
		c += x
		a, b, c = uhash.MixState(a, b, c)
		i += 12
	}
	c += n
	for j := n - i; j > 0; {
		j--
		x, e := s.byteAt(uint64(i) + uint64(j))
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
	return uhash.Mix(a, b, c), nil
}
