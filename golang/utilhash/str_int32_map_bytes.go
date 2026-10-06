// Concrete source specialization: src/util/map.h:24-196; src/util/hashtable.h:44-63.
// Source: src/util/str_hashtable.h:24-31; src/util/hashtable.h:375-435,465-522.
// Source author: Leonardo de Moura (2006-09-13 / 2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
// Borrowed byte-input adapter for the source CString query. No byte slice is
// retained; the hash/probe/equality and prepare/publication actions are those
// of the existing concrete source string table.
package utilhash

func (h *StrInt32Map) FindCStringBytes(s []byte) (StrInt32MapEntryRef, bool, error) {
	return h.FindCStringByteSource(BorrowBytesCString(s))
}
func (h *StrInt32Map) FindCStringByteSource(s CStringByteSource) (StrInt32MapEntryRef, bool, error) {
	if e := h.open(); e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	hash, e := s.Hash()
	if e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	capacity, e := h.capacityChecked()
	if e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	table := h.table
	mask := capacity - 1
	i := hash & mask
	for step := uint32(0); step < capacity; step++ {
		var entry strInt32Entry
		if h.config.Runtime == nil {
			p, re := mapLegacyEntryAt(table, i)
			if re != nil {
				e = re
			} else {
				entry = *p
			}
		} else {
			entry, e = h.probeEntry(i)
		}
		if e != nil {
			return StrInt32MapEntryRef{}, false, e
		}
		if entry.state == strHashUsed && entry.hash == hash {
			if h.config.Runtime != nil {
				entry.key, e = h.entryKey(i)
			}
			if e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
			region, e := h.keyRegion(entry.key)
			if e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
			equal, e := strKeyEqualByteSource(region, entry.key, s)
			if e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
			if equal {
				return h.ref(i), true, nil
			}
		}
		if entry.state == strHashUsed {
			if e := h.incrementCollision(); e != nil {
				return StrInt32MapEntryRef{}, false, e
			}
		}
		if entry.state == strHashFree {
			return StrInt32MapEntryRef{}, false, nil
		}
		if e := h.incrementCollision(); e != nil {
			return StrInt32MapEntryRef{}, false, e
		}
		i++
		if i == capacity {
			i = 0
		}
	}
	return StrInt32MapEntryRef{}, false, nil
}
func (h *StrInt32Map) PrepareFindCStringBytes(s []byte) (StrInt32MapEntryRef, bool, error) {
	return h.PrepareFindCStringByteSource(BorrowBytesCString(s))
}
func (h *StrInt32Map) PrepareFindCStringByteSource(s CStringByteSource) (StrInt32MapEntryRef, bool, error) {
	if e := h.prepareInsert(); e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	r, found, e := h.FindCStringByteSource(s)
	if e != nil || found {
		return r, found, e
	}
	if e = h.canBump(); e != nil {
		return StrInt32MapEntryRef{}, false, e
	}
	return StrInt32MapEntryRef{}, false, nil
}
