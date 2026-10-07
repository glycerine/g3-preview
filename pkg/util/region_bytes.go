// Source: src/util/hash.cpp:26-102; src/util/str_hashtable.h:28-30.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// RegionBytes borrows the actual region payload without cloning it. The codec
// reads ascending bytes within each source whole-word load, then the literal
// descending signed-char tail. Zero-length hashing reads no provider.

func RegionBytes(region *arena.BytesRegion, ref arena.BytesRegionRef, offset uint64, n, seed uint32) (uint32, error) {
	return runtimeHashBytes(runtimeHashSource{region: region, regionRef: ref, offset: offset, regionMode: true}, n, seed)
}
func strKeyByte(region *arena.BytesRegion, k StrKey, i uint64) (byte, error) {
	if region == nil || region.OwnerID() != k.origin {
		return 0, arena.ErrInvalidSlot
	}
	offset := uint64(k.offset) + i
	if offset < uint64(k.offset) || offset > uint64(^uint32(0)) {
		return 0, arena.ErrBounds
	}
	return region.Read(k.ref, uint32(offset))
}
func strKeyLength(region *arena.BytesRegion, k StrKey) (uint32, error) {
	for i := uint64(0); ; i++ {
		b, e := strKeyByte(region, k, i)
		if e != nil {
			return 0, e
		}
		if b == 0 {
			return uint32(i), nil
		}
	}
}

func strKeyEqualString(region *arena.BytesRegion, k StrKey, s string) (bool, error) {
	for i := uint64(0); ; i++ {
		x, e := strKeyByte(region, k, i)
		if e != nil {
			return false, e
		}
		var y byte
		if i < uint64(len(s)) {
			y = s[i]
		}
		if x != y {
			return false, nil
		}
		if x == 0 {
			return true, nil
		}
	}
}
func strKeyEqualBytes(region *arena.BytesRegion, k StrKey, s []byte) (bool, error) {
	for i := uint64(0); ; i++ {
		x, e := strKeyByte(region, k, i)
		if e != nil {
			return false, e
		}
		var y byte
		if i < uint64(len(s)) {
			y = s[i]
		}
		if x != y {
			return false, nil
		}
		if x == 0 {
			return true, nil
		}
	}
}
