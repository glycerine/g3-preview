// Source: src/util/str_hashtable.h:24-31; src/util/hashtable.h:375-435,465-522.
// Source author: Leonardo de Moura (2006-09-13 / 2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
// Borrowed byte-input adapter for the source CString query. No byte slice is
// retained; the hash/probe/equality and prepare/publication actions are those
// of the existing concrete source string table.
package utilhash

import "github.com/glycerine/g3/golang/arena"

func strQueryBytesPrefix(s []byte) []byte {
	for i, c := range s {
		if c == 0 {
			return s[:i]
		}
	}
	return s
}
func strQueryBytesHash(s []byte, init uint32) uint32 {
	return hashBytes(uint32(len(s)), init,
		func(i uint32) uint32 {
			return uint32(s[i]) | uint32(s[i+1])<<8 | uint32(s[i+2])<<16 | uint32(s[i+3])<<24
		},
		func(i uint32) uint32 { return uint32(int32(int8(s[i]))) })
}
func CStringBytesHash(s []byte) (uint32, error) {
	s = strQueryBytesPrefix(s)
	if uint64(len(s)) > uint64(^uint32(0)) {
		return 0, arena.ErrBounds
	}
	return strQueryBytesHash(s, 17), nil
}
func (h *StrHashtable) FindCStringBytes(s []byte) (StrHashtableEntryRef, bool, error) {
	if e := h.open(); e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	s = strQueryBytesPrefix(s)
	if uint64(len(s)) > uint64(^uint32(0)) {
		return StrHashtableEntryRef{}, false, arena.ErrBounds
	}
	hash := strQueryBytesHash(s, 17)
	capacity, e := h.capacityChecked()
	if e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	mask := capacity - 1
	i := hash & mask
	for step := uint32(0); step < capacity; step++ {
		entry, e := h.probeEntry(i)
		if e != nil {
			return StrHashtableEntryRef{}, false, e
		}
		if entry.state == strHashUsed && entry.hash == hash {
			entry.key, e = h.entryKey(i)
			if e != nil {
				return StrHashtableEntryRef{}, false, e
			}
			if e := h.validateKey(entry.key); e != nil {
				return StrHashtableEntryRef{}, false, e
			}
			region, e := h.keyRegion(entry.key)
			if e != nil {
				return StrHashtableEntryRef{}, false, e
			}
			equal := entry.key.length == uint32(len(s))
			if equal {
				for j := uint32(0); j < entry.key.length; j++ {
					b, e := region.Read(entry.key.ref, entry.key.offset+j)
					if e != nil {
						return StrHashtableEntryRef{}, false, e
					}
					if b != s[j] {
						equal = false
						break
					}
				}
			}
			if equal {
				return h.ref(i), true, nil
			}
		}
		if entry.state == strHashUsed {
			if e := h.incrementCollision(); e != nil {
				return StrHashtableEntryRef{}, false, e
			}
		}
		if entry.state == strHashFree {
			return StrHashtableEntryRef{}, false, nil
		}
		if e := h.incrementCollision(); e != nil {
			return StrHashtableEntryRef{}, false, e
		}
		if h.config.Runtime != nil {
			i++
			if i == capacity {
				i = 0
			}
		} else {
			i = (i + 1) & mask
		}
	}
	return StrHashtableEntryRef{}, false, nil
}
func (h *StrHashtable) PrepareFindCStringBytes(s []byte) (StrHashtableEntryRef, bool, error) {
	if e := h.prepareInsert(); e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	r, found, e := h.FindCStringBytes(s)
	if e != nil || found {
		return r, found, e
	}
	if e = h.canBump(); e != nil {
		return StrHashtableEntryRef{}, false, e
	}
	return StrHashtableEntryRef{}, false, nil
}
