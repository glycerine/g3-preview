// Source: src/util/hashtable.h:581-600; src/util/str_hashtable.h:24-31.
// Source author: Leonardo de Moura (2006-09-11; 2006-09-13).
// Licensed under the MIT license; see LICENSE.txt.
package utilhash

import "github.com/glycerine/g3/golang/arena"

// StrHashtableOutput is a borrowed insertion chain, used only for this call.
// The CString insertion retains the actual pointer and reads its spelling at
// insertion time, including when the output stream already has badbit set.
type StrHashtableOutput interface {
	AppendCString(string) error
	AppendStrHashtableCString(*arena.BytesRegion, StrKey) error
}

func (h *StrHashtable) DumpInto(out StrHashtableOutput) error {
	if e := h.open(); e != nil {
		return e
	}
	// Capture both iterator endpoints before the first insertion. Reads below
	// use this physical backing, rather than reloading the table field.
	if h.config.Runtime != nil {
		block, e := h.runtimeTable()
		if e != nil {
			return e
		}
		cap, e := h.capacityChecked()
		if e != nil {
			return e
		}
		if e = out.AppendCString("["); e != nil {
			return e
		}
		first := true
		for i := uint32(0); i < cap; i++ {
			state, e := h.stateAt(block, i)
			if e != nil {
				return e
			}
			if state != strHashUsed {
				continue
			}
			if first {
				first = false
			} else if e = out.AppendCString(" "); e != nil {
				return e
			}
			key, e := h.keyAt(block, i)
			if e != nil {
				return e
			}
			region, e := h.keyRegion(key)
			if e != nil {
				return e
			}
			if e = out.AppendStrHashtableCString(region, key); e != nil {
				return e
			}
		}
		return out.AppendCString("]")
	}
	table := h.table
	cap := h.Capacity()
	if e := out.AppendCString("["); e != nil {
		return e
	}
	first := true
	for i := uint32(0); i < cap; i++ {
		entry, e := legacyEntryAt(table, i)
		if e != nil {
			return e
		}
		if entry.state != strHashUsed {
			continue
		}
		if first {
			first = false
		} else if e := out.AppendCString(" "); e != nil {
			return e
		}
		key := entry.key
		region, e := h.keyRegion(key)
		if e != nil {
			return e
		}
		if e = out.AppendStrHashtableCString(region, key); e != nil {
			return e
		}
	}
	return out.AppendCString("]")
}
