// Source: src/util/str_hashtable.h:28-30; src/util/hash.cpp:26-102.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// Call-local borrowed pointer representation. It owns no bytes and is never
// stored in a hot table entry. Runtime payload reads retain exact generation
// and initialized-byte checks; byte slices retain their existing CString adapter.

type CStringByteSource struct {
	bytes          []byte
	initialized    *uint16
	runtime        *Runtime
	block          Block
	offset, extent uint64
	raw            bool
}

func BorrowBytesCString(s []byte) CStringByteSource { return CStringByteSource{bytes: s} }
func BorrowInitializedBytesCString(s []byte, mask *uint16) CStringByteSource {
	return CStringByteSource{bytes: s, initialized: mask}
}
func BorrowRuntimeCString(r *Runtime, b Block, offset, extent uint64) CStringByteSource {
	return CStringByteSource{runtime: r, block: b, offset: offset, extent: extent, raw: true}
}
func (s CStringByteSource) Byte(i uint64) (byte, error) {
	if !s.raw {
		if i >= uint64(len(s.bytes)) {
			return 0, nil
		}
		if s.initialized != nil {
			if i >= 16 {
				return 0, arena.ErrBounds
			}
			if *s.initialized&(uint16(1)<<i) == 0 {
				return 0, ErrRuntimeUninitialized
			}
		}
		return s.bytes[i], nil
	}
	if i >= s.extent || s.offset+i < s.offset {
		return 0, arena.ErrBounds
	}
	if s.runtime == nil {
		return 0, ErrRuntimeOwner
	}
	return s.runtime.Read(s.block, s.offset+i)
}
func (s CStringByteSource) Length() (uint32, error) {
	for i := uint64(0); ; i++ {
		v, e := s.Byte(i)
		if e != nil {
			return 0, e
		}
		if v == 0 {
			if i > uint64(^uint32(0)) {
				return 0, arena.ErrBounds
			}
			return uint32(i), nil
		}
	}
}
func (s CStringByteSource) Hash() (uint32, error) {
	n, e := s.Length()
	if e != nil {
		return 0, e
	}
	if !s.raw {
		return strQueryBytesHash(s.bytes[:n], 17), nil
	}
	if uint64(n)+s.offset < s.offset {
		return 0, arena.ErrBounds
	}
	return RuntimeBytesBounded(s.runtime, s.block, s.offset, s.extent, n, 17)
}
func strKeyEqualByteSource(region *arena.BytesRegion, k StrKey, s CStringByteSource) (bool, error) {
	for i := uint64(0); ; i++ {
		x, e := strKeyByte(region, k, i)
		if e != nil {
			return false, e
		}
		y, e := s.Byte(i)
		if e != nil {
			return false, e
		}
		if x != y {
			return false, nil
		}
		if x == 0 {
			return true, nil
		}
	}
}
